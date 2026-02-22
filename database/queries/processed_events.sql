-- name: InsertProcessedEvent :exec
INSERT INTO processed_events (event_id) VALUES ($1);

-- name: ProcessedEventExists :one
SELECT EXISTS(SELECT 1 FROM processed_events WHERE event_id = $1);
