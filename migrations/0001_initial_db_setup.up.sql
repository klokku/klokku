CREATE SCHEMA IF NOT EXISTS klokku;

SET search_path TO klokku, public;

CREATE TABLE users
(
    id                                INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uid                               TEXT    NOT NULL DEFAULT '',
    username                          TEXT    NOT NULL UNIQUE,
    display_name                      TEXT    NOT NULL,
    photo_url                         TEXT,
    timezone                          TEXT    NOT NULL,
    week_first_day                    INTEGER NOT NULL,
    event_calendar_type               TEXT    NOT NULL,
    event_calendar_google_calendar_id TEXT
);
CREATE UNIQUE INDEX idx_user_uid ON users (uid);

CREATE TABLE budget_plan
(
    id      INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name    TEXT    NOT NULL,
    created TIMESTAMPTZ DEFAULT NOW(),
    user_id INTEGER NOT NULL
);
CREATE INDEX budget_plan_user_id_idx ON budget_plan (user_id);

CREATE TABLE budget_plan_current
(
    id             INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    budget_plan_id INTEGER NOT NULL REFERENCES budget_plan (id),
    user_id        INTEGER NOT NULL
);
CREATE UNIQUE INDEX budget_plan_current_user_id_idx ON budget_plan_current (user_id);

CREATE TABLE budget_item
(
    id                  INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    budget_plan_id      INTEGER NOT NULL REFERENCES budget_plan (id) ON DELETE CASCADE,
    name                TEXT    NOT NULL,
    weekly_duration_sec INTEGER NOT NULL,
    weekly_occurrences  INTEGER,
    icon                TEXT,
    color               TEXT,
    position            INTEGER NOT NULL,
    user_id             INTEGER NOT NULL
);
CREATE INDEX budget_item_budget_plan_id_idx ON budget_item (budget_plan_id);

CREATE TABLE weekly_plan_item
(
    id                  INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    budget_item_id      INTEGER NOT NULL,
    budget_plan_id      INTEGER NOT NULL,
    week_number         TEXT    NOT NULL, -- ISO 8601 week number, e.g. "2025-W03"
    name                TEXT    NOT NULL,
    weekly_duration_sec INTEGER NOT NULL,
    weekly_occurrences  INTEGER,
    icon                TEXT,
    color               TEXT,
    notes               TEXT    NOT NULL DEFAULT '',
    position            INTEGER NOT NULL,
    user_id             INTEGER NOT NULL DEFAULT 1
);
CREATE INDEX weekly_plan_item_user_id_week_number_idx ON weekly_plan_item (user_id, week_number);
CREATE UNIQUE INDEX weekly_plan_item_budget_item_week_number_idx ON weekly_plan_item (budget_item_id, week_number);

CREATE TABLE calendar_event
(
    id             INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uid            TEXT        NOT NULL,
    summary        TEXT        NOT NULL,
    start_time     TIMESTAMPTZ NOT NULL,
    end_time       TIMESTAMPTZ NOT NULL,
    budget_item_id INTEGER     NOT NULL,
    user_id        INTEGER     NOT NULL
);
CREATE INDEX calendar_user_id_event_start_end_idx ON calendar_event (user_id, start_time, end_time);

CREATE TABLE clickup_auth
(
    user_id       INTEGER PRIMARY KEY,
    access_token  TEXT,
    refresh_token TEXT,
    expiry        TIMESTAMPTZ,
    nonce         TEXT
);

CREATE TABLE clickup_config
(
    id                       INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id                  INTEGER NOT NULL,
    budget_plan_id           INTEGER NOT NULL,
    workspace_id             TEXT    NOT NULL,
    space_id                 TEXT    NOT NULL,
    folder_id                TEXT,
    only_tasks_with_priority BOOLEAN NOT NULL
);
CREATE INDEX clickup_config_budget_plan_id_idx ON clickup_config (budget_plan_id);
CREATE UNIQUE INDEX clickup_config_user_id_budget_plan_id_idx ON clickup_config (user_id, budget_plan_id);

CREATE TABLE clickup_tag_mapping
(
    id                INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    clickup_config_id INTEGER NOT NULL,
    clickup_space_id  TEXT    NOT NULL,
    clickup_tag_name  TEXT    NOT NULL,
    budget_item_id    INTEGER NOT NULL,
    position          INTEGER NOT NULL,
    user_id           INTEGER NOT NULL,
    FOREIGN KEY (clickup_config_id) REFERENCES clickup_config (id) ON DELETE CASCADE
);
CREATE INDEX clickup_tag_mapping_budget_item_id_idx ON clickup_tag_mapping (budget_item_id);
CREATE INDEX clickup_tag_mapping_clickup_config_id_idx ON clickup_tag_mapping (clickup_config_id);
CREATE UNIQUE INDEX clickup_tag_mapping_user_id_budget_item_id_idx ON clickup_tag_mapping (user_id, budget_item_id);
CREATE UNIQUE INDEX clickup_tag_mapping_user_id_config_id_tag_name_idx ON clickup_tag_mapping (user_id, clickup_config_id, clickup_tag_name);

CREATE TABLE current_event
(
    id                            INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    budget_item_id                INTEGER     NOT NULL REFERENCES budget_item (id),
    budget_item_name              text,
    plan_item_weekly_duration_sec INTEGER     NOT NULL,
    start_time                    TIMESTAMPTZ NOT NULL,
    user_id                       INTEGER     NOT NULL
);
CREATE UNIQUE INDEX current_event_user_id_idx ON current_event (user_id);

CREATE TABLE google_calendar_auth
(
    user_id       INTEGER PRIMARY KEY,
    access_token  TEXT,
    refresh_token TEXT,
    expiry        TIMESTAMPTZ,
    nonce         TEXT
);
CREATE INDEX google_calendar_auth_nonce_idx ON google_calendar_auth (nonce);
