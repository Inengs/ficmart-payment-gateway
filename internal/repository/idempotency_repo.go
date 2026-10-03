package repository

import (
	"database/sql"
	"errors"

	"github.com/Inengs/ficmart-payment-gateway/internal/domain"
	"github.com/jmoiron/sqlx"
)

type IdempotencyRepository struct{ db *sqlx.DB } // repository struct

// constructor
func NewIdempotencyRepository(db *sqlx.DB) *IdempotencyRepository {
	return &IdempotencyRepository{db: db}
}

// Begin returns created=true if this caller now owns the key, or returns the existing record if the key is already owned by another caller. It also clears abandoned in-progress rows older than 2 minutes.
func (r *IdempotencyRepository) Begin(key, path, hash string) (*domain.IdempotencyRecord, bool, error) {
	// clear in-progress rows abandoned by a crash
	if _, err := r.db.Exec(`DELETE FROM idempotency_keys
		WHERE idempotency_key=$1 AND request_path=$2
		AND response_status IS NULL AND created_at < now() - interval '2 minutes'`, key, path); err != nil {
		return nil, false, err
	} // delete abandoned in-progress rows, why? because if a request is in progress and the server crashes, we want to allow a new request with the same idempotency key to proceed after a timeout.

	// try to insert a new row for this key
	res, err := r.db.Exec(`INSERT INTO idempotency_keys (idempotency_key, request_path, request_hash)
		VALUES ($1,$2,$3) ON CONFLICT (idempotency_key, request_path) DO NOTHING`, key, path, hash)
	if err != nil {
		return nil, false, err // return error if insert fails
	}
	if n, _ := res.RowsAffected(); n == 1 {
		return nil, true, nil // if insert succeeded, this caller now owns the key, what is the caller? the caller is the request handler that is trying to process a request with this idempotency key.
	}

	var record domain.IdempotencyRecord // if insert failed, the key already exists, so we need to check if the existing record matches the request hash
	err = r.db.Get(&record, `SELECT request_hash, response_status, response_body
		FROM idempotency_keys WHERE idempotency_key=$1 AND request_path=$2`, key, path) // get the existing record for this key
	if err != nil {
		return nil, false, err // return error if select fails
	}
	if errors.Is(err, sql.ErrNoRows) { // released between insert and select, try again
		return r.Begin(key, path, hash) // if the record was deleted between the insert and select, try again
	}
	return &record, false, err // return the existing record and indicate that this caller does not own the key
}

// Complete marks the idempotency key as completed with the given response status and body.
func (r *IdempotencyRepository) Complete(key, path string, status int, body []byte) error {
	_, err := r.db.Exec(`UPDATE idempotency_keys SET response_status=$1, response_body=$2::jsonb
		WHERE idempotency_key=$3 AND request_path=$4`, status, string(body), key, path) // update the existing record with the response status and body
	return err // return error if update fails
}

// Release deletes the idempotency key if it is still in progress (response_status is NULL). This allows a new request with the same key to proceed.
func (r *IdempotencyRepository) Release(key, path string) error {
	_, err := r.db.Exec(`DELETE FROM idempotency_keys
		WHERE idempotency_key=$1 AND request_path=$2 AND response_status IS NULL`, key, path) // delete the existing record if it is still in progress
	return err // return error if delete fails
}
