#### payments/authorize

when the payments/authorize endpoint is clicked, it validates the request, then calls the authorize service to authorize it.

then a payment is created in PENDING State, then the api calls the bank api, updates the payment to AUTHORIZED, then returns payment details to Ficmart
