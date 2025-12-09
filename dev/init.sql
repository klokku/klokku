-- Create additional schema
CREATE SCHEMA IF NOT EXISTS klokku;

-- Grant permissions
GRANT USAGE ON SCHEMA klokku TO PUBLIC;
GRANT CREATE ON SCHEMA klokku TO PUBLIC;

-- Set search_path to include the new schema
ALTER DATABASE klokku SET search_path TO klokku, public;
