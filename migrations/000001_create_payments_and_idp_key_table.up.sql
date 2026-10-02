-- migrations/001_initial_table.sql
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id TEXT not null,
    customer_id TEXT not null,
    amount INTEGER not null,
    currency CHAR(3),
    status TEXT not null,
    card_last_four CHAR(4),
    bank_auth_id TEXT,
    bank_capture_id TEXT,
    bank_void_id TEXT,
    bank_refund_id TEXT,
    failure_reason TEXT,
    failed_at TIMESTAMPTZ,
    authorized_at TIMESTAMP,
    captured_at TIMESTAMP,
    voided_at TIMESTAMP,
    refunded_at TIMESTAMP,
    created_at TIMESTAMP not null DEFAULT now(),
    updated_at TIMESTAMP not null DEFAULT now(),
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