CREATE TABLE IF NOT EXISTS checkpoint (
    id VARCHAR(50) PRIMARY KEY,
    last_processed_block BIGINT NOT NULL,
    last_processed_tx_index INT NOT NULL
);
