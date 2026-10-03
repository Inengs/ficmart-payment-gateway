package domain

type IdempotencyRecord struct {
	RequestHash    string `db:"request_hash"`
	ResponseStatus *int   `db:"response_status"`
	ResponseBody   []byte `db:"response_body"`
}
