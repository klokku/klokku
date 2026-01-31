SET search_path TO klokku, public;

-- Remove webhook_token from budget_item table
ALTER TABLE budget_item DROP COLUMN IF EXISTS webhook_token;

-- Create webhooks table
CREATE TABLE webhooks (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    type TEXT NOT NULL,
    token TEXT NOT NULL,
    user_id INTEGER NOT NULL,
    data JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX webhooks_token_idx ON webhooks (token);
CREATE INDEX webhooks_user_id_idx ON webhooks (user_id);
CREATE INDEX webhooks_type_idx ON webhooks (type);
