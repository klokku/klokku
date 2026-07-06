SET search_path TO klokku, public;

-- Backfill rows previously blanked by the profile update bug
UPDATE users SET event_calendar_type = 'klokku' WHERE event_calendar_type = '' OR event_calendar_type IS NULL;

-- Ensure new/updated rows default to the Klokku calendar
ALTER TABLE users ALTER COLUMN event_calendar_type SET DEFAULT 'klokku';

-- Reject empty or unknown values at the database level
ALTER TABLE users ADD CONSTRAINT users_event_calendar_type_check CHECK (event_calendar_type IN ('klokku', 'google'));
