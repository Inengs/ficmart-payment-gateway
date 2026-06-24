-- migrations/001_initial_table.sql
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid
    order_id TEXT
    customer_id TEXT
    amount DECIMAL
    currency CHAR(3)
    status TEXT
    card_last_four CHAR(4)
    bank_auth_id TEXT NULLABLE
    bank_capture_id TEXT NULLABLE
    bank_void_id TEXT NULLABLE
    authorized_at TIMESTAMP NULLABLE
    captured_at TIMESTAMP NULLABLE
    voided_at TIMESTAMP NULLABLE
    refunded_at TIMESTAMP NULLABLE
    created_at TIMESTAMP
    updated_at TIMESTAMP
)

CREATE TABLE idempotency_key(
    id UUID PRIMARY
    idempotency_key TEXT UNIQUE
    request_path TEXT
    request_hash TEXT
    response_status INTEGER
    response_body JSONB
    created_at TIMESTAMP
    payment_id UUID NULLABLE
)