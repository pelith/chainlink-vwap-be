CREATE TABLE IF NOT EXISTS trades (
    trade_id VARCHAR(66) PRIMARY KEY,
    maker VARCHAR(42) NOT NULL,
    taker VARCHAR(42) NOT NULL,
    maker_is_sell_eth BOOLEAN NOT NULL,
    maker_amount_in VARCHAR(78) NOT NULL,
    taker_deposit VARCHAR(78) NOT NULL,
    delta_bps INT NOT NULL,
    start_time BIGINT NOT NULL,
    end_time BIGINT NOT NULL,
    status VARCHAR(20) NOT NULL,
    settlement_price VARCHAR(78),
    maker_payout VARCHAR(78),
    taker_payout VARCHAR(78),
    maker_refund VARCHAR(78),
    taker_refund VARCHAR(78),
    created_at TIMESTAMP NOT NULL,
    settled_at TIMESTAMP,
    refunded_at TIMESTAMP
);

CREATE INDEX idx_trades_maker ON trades(maker);
CREATE INDEX idx_trades_taker ON trades(taker);
CREATE INDEX idx_trades_status ON trades(status);
