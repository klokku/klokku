CREATE TABLE clickup_auth
(
    user_id       INTEGER PRIMARY KEY,
    access_token  TEXT,
    refresh_token TEXT,
    expiry        INTEGER NULL,
    nonce         TEXT
);

CREATE TABLE clickup_config
(
    user_id      INTEGER PRIMARY KEY,
    workspace_id INTEGER NOT NULL,
    space_id     INTEGER NOT NULL,
    folder_id    INTEGER NULL
);

CREATE TABLE clickup_tag_mapping
(
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id          INTEGER NOT NULL REFERENCES clickup_config (user_id),
    clickup_space_id INTEGER NOT NULL,
    clickup_tag_name TEXT    NOT NULL,
    budget_id        INTEGER NOT NULL,
    position         INTEGER NOT NULL
);