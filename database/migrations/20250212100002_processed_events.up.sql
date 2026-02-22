CREATE TABLE IF NOT EXISTS processed_events (
    event_id VARCHAR(130) PRIMARY KEY
);

CREATE INDEX idx_processed_events_event_id ON processed_events(event_id);
