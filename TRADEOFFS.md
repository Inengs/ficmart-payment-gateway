I added a FAILED status, for a transaction that is an irreversible failure like Insufficient funds. 

Failure handling
- I also added a 503 status code for transient failure, so with this response the row is left pending. this shows when the bank client fails in a transient way. This is after the 3 attempts using backoff with jitter method. this is the best status code when the bank client is unable to handle the request, so we save it as pending.