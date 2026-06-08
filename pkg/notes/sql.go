package notes

const noteColumns = `
n.id, n.owner_user_id, n.title, n.body_markdown, n.category,
COALESCE(array_agg(nt.tag ORDER BY nt.tag) FILTER (WHERE nt.tag IS NOT NULL), ARRAY[]::text[]) AS tags,
n.created_at, n.updated_at`

const createNoteSQL = `
INSERT INTO notes (owner_user_id, title, body_markdown, category)
VALUES ($1, $2, $3, $4)
RETURNING id, owner_user_id, title, body_markdown, category, created_at, updated_at;`

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
LIMIT $2;`

const searchNotesSQL = `
SELECT ` + noteColumns + `
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
LIMIT $5;`

const updateNoteSQL = `
UPDATE notes
SET title = $3, body_markdown = $4, category = $5, updated_at = clock_timestamp()
WHERE owner_user_id = $1 AND id = $2
RETURNING id;`

const deleteTagsSQL = `DELETE FROM note_tags WHERE owner_user_id = $1 AND note_id = $2;`

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
