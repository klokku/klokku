SET search_path TO klokku, public;

ALTER TABLE users ADD COLUMN ignore_short_events BOOLEAN NOT NULL DEFAULT FALSE;