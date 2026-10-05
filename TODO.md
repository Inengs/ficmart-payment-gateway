1. use K6 to test the api and document the results
2. setup postman to test the endpoints
3. Issue 1: Order lookups can return the wrong row

Problem: a 503 followed by a successful retry leaves two rows for one order (an orphan PENDING and an AUTHORIZED one). GET /orders/{order_id}/payment could return the PENDING orphan, so FicMart would see the wrong state.
Fix: make the query deterministic: prefer non-PENDING rows, then the newest (ORDER BY (status = 'PENDING') ASC, created_at DESC LIMIT 1).
Where: payment_repo.go (GetByOrderID), then the service, handler and route.
When: while building the GET endpoints.
Extra: the customer history endpoint should return all rows, including PENDING ones.

4. Issue 2: Preventing orphan rows (find-or-create)

Problem: the service can't tell a retry from a new request, because the payments table doesn't store the client's Idempotency-Key. Every retry after a released 503 inserts a new row.
Fix: add a nullable idempotency_key column with a unique index on payments. At the start of Authorize, find or create the row by that key, then:
PENDING: reuse it and go straight to the bank call, with the same bank key.
AUTHORIZED: return it, with no bank call.
FAILED: return the stored failure.
Where: a new migration (or an edit to 000001 if it's still unapplied), payment_repo.go, payment_service.go.
When: after capture, void and refund work.
Limit: crash orphans where FicMart never retries still need the reconciler, and the reconciler can only look up rows that have a bank_auth_id.
For TRADEOFFS.md: explain in your own words that you accepted orphans first, made reads deterministic as a stopgap, then added find-or-create, with the reconciler as the backstop.