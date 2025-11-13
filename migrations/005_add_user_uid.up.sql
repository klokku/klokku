-- Add the column with the default empty string
ALTER TABLE user ADD uid TEXT NOT NULL DEFAULT '';

-- Populate UIDs for existing users using SQLite's built-in functions
UPDATE user
SET uid = PRINTF('%08x-%04x-%04x-%04x-%012x',
                 ABS(RANDOM()) % 4294967296,
                 ABS(RANDOM()) % 65536,
                 ABS(RANDOM()) % 65536,
                 ABS(RANDOM()) % 65536,
                 ABS(RANDOM()) % 281474976710656)
WHERE uid = '';
