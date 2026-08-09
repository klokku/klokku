ALTER TABLE calendar_event
    ADD CONSTRAINT calendar_event_notes_length_check CHECK (char_length(notes) <= 10000);

ALTER TABLE current_event
    ADD CONSTRAINT current_event_notes_length_check CHECK (char_length(notes) <= 10000);
