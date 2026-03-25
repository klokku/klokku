CREATE TABLE weekly_plan
(
    id             INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id        INTEGER NOT NULL,
    budget_plan_id INTEGER NOT NULL,
    week_number    TEXT    NOT NULL,
    is_off_week    BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE UNIQUE INDEX weekly_plan_user_id_week_number_idx ON weekly_plan (user_id, week_number);
