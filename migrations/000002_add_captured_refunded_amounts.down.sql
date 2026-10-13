ALTER TABLE payments
    DROP CONSTRAINT chk_refunded_le_captured,
    DROP CONSTRAINT chk_captured_le_amount,
    DROP COLUMN refunded_amount,
    DROP COLUMN captured_amount;