GW=localhost:8080
K=$(date +%s)
H='Content-Type: application/json'

pay() { # $1 = suffix, $2 = extra json
  curl -s -X POST $GW/payments/authorize -H "$H" -H "Idempotency-Key: $K-a-$1" \
    -d "{\"order_id\":\"o-$K-$1\",\"customer_id\":\"c1\",\"amount\":10000,\"currency\":\"USD\",\"card_number\":\"4111111111111111\",\"cvv\":\"123\",\"expiry_month\":12,\"expiry_year\":2030$2}"
}
call() { curl -s -w "\n  -> HTTP %{http_code}\n" -X POST "$@"; }

echo "=== 1. capture with no body (expect 200, captured_amount 10000) ==="
ID=$(pay 1 | jq -r .payment_id); echo "payment: $ID"
call $GW/payments/$ID/capture -H "Idempotency-Key: $K-c1"

echo; echo "=== 2. capture with amount 6000 (expect 4xx from bank amount_mismatch) ==="
ID=$(pay 2 | jq -r .payment_id)
call $GW/payments/$ID/capture -H "$H" -H "Idempotency-Key: $K-c2" -d '{"amount":6000}'

echo; echo "=== 3. capture with amount 99999 (expect 4xx from gateway, never reaches bank) ==="
ID=$(pay 3 | jq -r .payment_id)
call $GW/payments/$ID/capture -H "$H" -H "Idempotency-Key: $K-c3" -d '{"amount":99999}'

echo; echo "=== 4. JPY authorize (expect 4xx not supported) ==="
call $GW/payments/authorize -H "$H" -H "Idempotency-Key: $K-a-jpy" \
  -d "{\"order_id\":\"o-$K-jpy\",\"customer_id\":\"c1\",\"amount\":10000,\"currency\":\"JPY\",\"card_number\":\"4111111111111111\",\"cvv\":\"123\",\"expiry_month\":12,\"expiry_year\":2030}"

echo; echo "=== 5. full refund with no body (expect 200, status REFUNDED) ==="
ID=$(pay 5 | jq -r .payment_id)
call $GW/payments/$ID/capture -H "Idempotency-Key: $K-c5" > /dev/null
call $GW/payments/$ID/refund -H "Idempotency-Key: $K-r5"
