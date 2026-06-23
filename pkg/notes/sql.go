package notes

// SQL query fragments for the notes repository.
//
// These are the single source for the column projections and DML used by PostgresRepository.
// They are intentionally kept as raw SQL for full control over joins, window functions,
// tag aggregation, and conflict handling.
//
// When evolving the Note (usually after a proto change):
//  1. Add migration if the DB shape changes.
//  2. Update the relevant SELECT/INSERT/UPDATE here (often just extend noteColumns).
//  3. Add the field + `db:"col"` tag to the Note struct in model.go.
//  4. Update mappers.go if the field must cross the proto boundary.
//  5. Update service logic / validation if needed.
//  6. The named struct scanning in storage_postgres.go usually requires no change.
//
// Named scanning (pgx.RowToStructByNameLax) matches result columns to struct fields via the `db` tag.
const noteColumns = `
n.id, n.owner_user_id, n.title, n.body_markdown, n.category, n.status,
COALESCE(array_agg(nt.tag ORDER BY nt.tag) FILTER (WHERE nt.tag IS NOT NULL), ARRAY[]::text[]) AS tags,
n.created_at, n.updated_at`

// searchNoteColumns extends noteColumns with a window-function total count evaluated before LIMIT.
const searchNoteColumns = noteColumns + `,
COUNT(*) OVER() AS total_count`

const createNoteSQL = `
INSERT INTO notes (owner_user_id, title, body_markdown, category, status)
VALUES ($1, $2, $3, $4, $5)
RETURNING id;`

const insertTagSQL = `
INSERT INTO note_tags (note_id, owner_user_id, tag)
VALUES ($1, $2, $3)
ON CONFLICT (note_id, tag) DO NOTHING;`

const getNoteSQL = `
SELECT ` + noteColumns + `
FROM notes n
LEFT JOIN note_tags nt ON nt.note_id = n.id AND nt.owner_user_id = n.owner_user_id
WHERE n.owner_user_id = $1 AND n.id = $2
GROUP BY n.id;`

const listRecentNotesSQL = `
SELECT ` + noteColumns + `
FROM notes n
LEFT JOIN note_tags nt ON nt.note_id = n.id AND nt.owner_user_id = n.owner_user_id
WHERE n.owner_user_id = $1
GROUP BY n.id
ORDER BY n.updated_at DESC
LIMIT $2 OFFSET $3;`

const searchNotesSQL = `
SELECT ` + searchNoteColumns + `
FROM notes n
LEFT JOIN note_tags nt ON nt.note_id = n.id AND nt.owner_user_id = n.owner_user_id
WHERE n.owner_user_id = $1
  AND ($2 = '' OR n.title ILIKE '%' || $2 || '%' OR n.body_markdown ILIKE '%' || $2 || '%')
  AND ($3 = '' OR n.category = $3)
  AND (cardinality($4::text[]) = 0 OR NOT EXISTS (
      SELECT 1 FROM unnest($4::text[]) requested_tag
      WHERE NOT EXISTS (
          SELECT 1 FROM note_tags matched_tag
          WHERE matched_tag.note_id = n.id
            AND matched_tag.owner_user_id = n.owner_user_id
            AND matched_tag.tag = requested_tag
      )
  ))
GROUP BY n.id
ORDER BY n.updated_at DESC
LIMIT $5 OFFSET $6;`

const updateNoteSQL = `
UPDATE notes
SET title = $3, body_markdown = $4, category = $5, status = $6, updated_at = clock_timestamp()
WHERE owner_user_id = $1 AND id = $2
RETURNING id;`

const deleteTagsSQL = `DELETE FROM note_tags WHERE owner_user_id = $1 AND note_id = $2;`

// note_tags rows are removed by the ON DELETE CASCADE foreign key.
const deleteNoteSQL = `DELETE FROM notes WHERE owner_user_id = $1 AND id = $2;`

const countTagsAfterAddSQL = `
SELECT COUNT(*) FROM (
    SELECT tag FROM note_tags WHERE owner_user_id = $1 AND note_id = $2
    UNION
    SELECT unnest($3::text[])
) resulting_tags;`

const touchNoteSQL = `
UPDATE notes SET updated_at = clock_timestamp()
WHERE owner_user_id = $1 AND id = $2
RETURNING id;`
