ALTER TABLE payments
    ADD COLUMN captured_amount BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN refunded_amount BIGINT NOT NULL DEFAULT 0;

UPDATE payments SET captured_amount = amount WHERE status IN ('CAPTURED', 'REFUNDED');
UPDATE payments SET refunded_amount = amount WHERE status = 'REFUNDED';

ALTER TABLE payments 
    ADD CONSTRAINT chk_captured_le_amount CHECK (captured_amount <= amount),
    ADD CONSTRAINT chk_refunded_le_captured CHECK (refunded_amount <= captured_amount);