-- migrate:up

-- go-mcp-markdown-notes: initial notes schema

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Notes are always owned by an internal application user ID supplied by the
-- authentication layer. BIGINT matches go-cloud-k8s-auth alternate_app_id.
CREATE TABLE notes (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    owner_user_id   BIGINT NOT NULL,
    title           TEXT NOT NULL,
    body_markdown   TEXT NOT NULL,
    category        TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT notes_owner_user_id_positive CHECK (owner_user_id > 0),
    CONSTRAINT notes_title_not_blank CHECK (length(btrim(title)) > 0),
    CONSTRAINT notes_title_max_length CHECK (char_length(title) <= 200),
    CONSTRAINT notes_body_max_length CHECK (char_length(body_markdown) <= 200000),
    CONSTRAINT notes_id_owner_unique UNIQUE (id, owner_user_id)
);

CREATE TABLE note_tags (
    note_id         UUID NOT NULL,
    owner_user_id   BIGINT NOT NULL,
    tag             TEXT NOT NULL,

    CONSTRAINT note_tags_pkey PRIMARY KEY (note_id, tag),
    CONSTRAINT note_tags_note_owner_fkey
        FOREIGN KEY (note_id, owner_user_id)
        REFERENCES notes (id, owner_user_id)
        ON DELETE CASCADE,
    CONSTRAINT note_tags_owner_user_id_positive CHECK (owner_user_id > 0),
    CONSTRAINT note_tags_tag_not_blank CHECK (length(btrim(tag)) > 0),
    CONSTRAINT note_tags_tag_max_length CHECK (char_length(tag) <= 50),
    CONSTRAINT note_tags_tag_normalized CHECK (tag = lower(btrim(tag)))
);

CREATE INDEX idx_notes_owner_updated
    ON notes (owner_user_id, updated_at DESC);

CREATE INDEX idx_note_tags_owner_tag
    ON note_tags (owner_user_id, tag);

-- Keep updated_at correct for writes outside the current service code as well.
-- migrate:statementbegin
CREATE FUNCTION set_notes_updated_at()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = clock_timestamp();
    RETURN NEW;
END;
$$;
-- migrate:statementend

CREATE TRIGGER notes_set_updated_at
BEFORE UPDATE ON notes
FOR EACH ROW
EXECUTE FUNCTION set_notes_updated_at();

-- migrate:down

-- migrate:statementbegin
DROP TRIGGER IF EXISTS notes_set_updated_at ON notes;
DROP FUNCTION IF EXISTS set_notes_updated_at();
-- migrate:statementend
DROP TABLE IF EXISTS note_tags;
DROP TABLE IF EXISTS notes;

-- uuid-ossp may be shared by other applications in the database, so this
-- migration deliberately leaves the extension installed.
