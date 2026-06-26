-- migration/001_create_initial_table.down.sql

DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS idempotency_keys;