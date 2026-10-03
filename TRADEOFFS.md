I added a FAILED status, for a transaction that is an irreversible failure like Insufficient funds. 

Failure handling
- I also added a 503 status code for transient failure, so with this response the row is left pending. this shows when the bank client fails in a transient way. This is after the 3 attempts using backoff with jitter method. this is the best status code when the bank client is unable to handle the request, so we save it as pending.

- Saved vs released: 200 and 402 are final answers, so they replay. A 503 is released so the client can retry with the same key. A 400 is released so the client can fix the body and reuse the key.
- Stale rows: a crash mid-request leaves a row with no response. After 2 minutes Begin clears it, otherwise that key would return "in progress" forever.
- Concurrent duplicates: the unique constraint plus ON CONFLICT DO NOTHING means only one request wins the insert, with no race.
- Hash: it's over the raw bytes, so the same JSON with different whitespace or key order counts as a different body. Fine for now, and worth a line in TRADEOFFS.md.