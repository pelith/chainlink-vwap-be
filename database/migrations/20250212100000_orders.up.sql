CREATE TABLE IF NOT EXISTS orders (
    order_hash VARCHAR(66) PRIMARY KEY,
    maker VARCHAR(42) NOT NULL,
    maker_is_sell_eth BOOLEAN NOT NULL,
    amount_in VARCHAR(78) NOT NULL,
    min_amount_out VARCHAR(78) NOT NULL,
    delta_bps INT NOT NULL,
    salt VARCHAR(78) NOT NULL,
    deadline BIGINT NOT NULL,
    signature BYTEA NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    filled_at TIMESTAMP,
    cancelled_at TIMESTAMP,
    expired_at TIMESTAMP
);

CREATE INDEX idx_orders_maker ON orders(maker);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_deadline ON orders(deadline) WHERE status = 'active';
