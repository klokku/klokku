CREATE TABLE calendar_event
(
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    uid         TEXT    NOT NULL,
    summary     TEXT    NOT NULL,
    start_time  INTEGER NOT NULL,
    end_time    INTEGER NOT NULL,
    budget_id   INTEGER NOT NULL,
    user_id     INTEGER NOT NULL
);