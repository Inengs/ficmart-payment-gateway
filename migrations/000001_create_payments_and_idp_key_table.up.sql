-- migrations/001_initial_table.sql
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id TEXT,
    customer_id TEXT,
    amount DECIMAL,
    currency CHAR(3),
    status TEXT,
    card_last_four CHAR(4),
    bank_auth_id TEXT,
    bank_capture_id TEXT,
    bank_void_id TEXT,
    authorized_at TIMESTAMP,
    captured_at TIMESTAMP,
    voided_at TIMESTAMP,
    refunded_at TIMESTAMP,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);

CREATE TABLE idempotency_keys(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    idempotency_key TEXT UNIQUE,
    request_path TEXT,
    request_hash TEXT,
    response_status INTEGER,
    response_body JSONB,
    created_at TIMESTAMP,
    payment_id UUID
);