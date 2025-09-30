CREATE TABLE budget
(
    id                 INTEGER PRIMARY KEY AUTOINCREMENT,
    name               TEXT               NOT NULL,
    weekly_time        INTEGER            NOT NULL,
    weekly_occurrences INTEGER default 0  NOT NULL,
    position           INTEGER            NOT NULL,
    status             TEXT               NOT NULL,
    user_id            INTEGER            NOT NULL,
    icon                       DEFAULT '' NOT NULL
);

CREATE TABLE budget_override
(
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    budget_id   INTEGER           NOT NULL,
    start_date  TEXT              NOT NULL,
    weekly_time INTEGER           NOT NULL,
    notes       TEXT,
    user_id     INTEGER default 1 NOT NULL
);

CREATE TABLE event
(
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    budget_id  INTEGER NOT NULL REFERENCES budget (id),
    start_time INTEGER NOT NULL,
    end_time   INTEGER,
    user_id    INTEGER NOT NULL
);

CREATE TABLE google_calendar_auth
(
    user_id       INTEGER NOT NULL UNIQUE,
    access_token  TEXT,
    refresh_token TEXT,
    expiry        INTEGER,
    nonce         TEXT
);

CREATE TABLE user
(
    id                                INTEGER PRIMARY KEY AUTOINCREMENT,
    username                          TEXT NOT NULL UNIQUE,
    display_name                      TEXT NOT NULL,
    photo_url                         TEXT,
    timezone                          TEXT NOT NULL,
    week_first_day                    INT  NOT NULL,
    event_calendar_type               TEXT NOT NULL,
    event_calendar_google_calendar_id TEXT
);

