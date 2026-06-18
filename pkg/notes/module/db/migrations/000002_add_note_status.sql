-- migrate:up

ALTER TABLE notes
    ADD COLUMN status SMALLINT NOT NULL DEFAULT 2,
    ADD CONSTRAINT notes_status_valid CHECK (status BETWEEN 1 AND 4);

-- migrate:down

ALTER TABLE notes DROP COLUMN status;
