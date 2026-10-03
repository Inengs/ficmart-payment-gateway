// checks idempotency keys before processing requests
package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"strings"

	"github.com/Inengs/ficmart-payment-gateway/internal/domain"
	"github.com/Inengs/ficmart-payment-gateway/internal/repository"
	"github.com/gin-gonic/gin"
)

// bodyWriter is a custom ResponseWriter that captures the response body for later use.
type bodyWriter struct {
	gin.ResponseWriter               // embeds the original ResponseWriter, what is the original ResponseWriter? it is the interface that allows us to write the response to the client
	body               *bytes.Buffer // buffer to store the response body
}

// Write captures the response body and writes it to the original ResponseWriter.
func (w *bodyWriter) Write(b []byte) (int, error) {
	w.body.Write(b)                  // write the response body to the buffer
	return w.ResponseWriter.Write(b) // write the response body to the original ResponseWriter
}

// reject is a helper function to abort the request with a JSON error response.
func reject(c *gin.Context, status int, code domain.ErrorCode, msg string) {
	c.AbortWithStatusJSON(status, gin.H{"code": code, "message": msg}) // abort the request and return a JSON error response with the given status, code, and message
}

// a closure that returns a gin.HandlerFunc to handle idempotency for requests. It checks the idempotency key, request hash, and manages the lifecycle of the idempotency record in the database.
func Idempotency(repo *repository.IdempotencyRepository) gin.HandlerFunc {
	// return a gin.HandlerFunc that handles idempotency for requests
	return func(c *gin.Context) {
		key := strings.TrimSpace(c.GetHeader("Idempotency-Key")) // get the idempotency key from the request header and trim whitespace
		if key == "" || len(key) > 255 {
			reject(c, 400, domain.ErrValidation, "Idempotency-Key header is required (max 255 characters)") // reject the request if the idempotency key is missing or too long
			return
		}

		raw, err := io.ReadAll(c.Request.Body) // read the raw request body to compute the hash
		if err != nil {
			reject(c, 400, domain.ErrValidation, "could not read request body") // reject the request if the body cannot be read
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(raw)) // give the body back to the handler, so it can read it again

		sum := sha256.Sum256(raw)          // compute the SHA-256 hash of the request body to ensure idempotency for identical requests
		hash := hex.EncodeToString(sum[:]) // convert the hash to a hexadecimal string
		path := c.Request.URL.Path         // get the request path to use as part of the idempotency key

		rec, created, err := repo.Begin(key, path, hash) // check if the idempotency key is already in use or if this caller can create a new record
		if err != nil {
			log.Printf("idempotency begin: %v", err)             // log the error if there is an issue with the idempotency repository
			reject(c, 500, domain.ErrInternal, "internal error") // reject the request with a 500 Internal Server Error if there is an issue with the idempotency repository
			return
		}

		if !created { // if the idempotency key already exists, check the existing record
			switch {
			case rec.RequestHash != hash:
				reject(c, 409, domain.ErrIdempotencyConflict, "Idempotency-Key was already used with a different request") // reject the request with a 409 Conflict if the existing record has a different request hash, indicating that the same idempotency key was used for a different request
			case rec.ResponseStatus == nil:
				reject(c, 409, domain.ErrIdempotencyConflict, "a request with this key is still in progress") // reject the request with a 409 Conflict if the existing record is still in progress (response status is nil), indicating that the same idempotency key is being used for a request that has not yet completed
			default:
				c.Header("Idempotent-Replayed", "true")                           // set a header to indicate that this is a replay of a previous request with the same idempotency key
				c.Data(*rec.ResponseStatus, "application/json", rec.ResponseBody) // return the cached response from the existing record, allowing the client to receive the same response as the original request without reprocessing it
				c.Abort()                                                         // abort the request to prevent further processing, as the response has already been sent
			}
			return
		}

		c.Set("idempotency_key", key)                                      // store the idempotency key in the context for use by downstream handlers
		bw := &bodyWriter{ResponseWriter: c.Writer, body: &bytes.Buffer{}} // create a new bodyWriter to capture the response body for later use
		c.Writer = bw                                                      // replace the original ResponseWriter with the bodyWriter to capture the response body
		c.Next()                                                           // call the next handler in the chain to process the request

		// save definitive outcomes; release anything the client may safely retry
		if s := c.Writer.Status(); s == 200 || s == 201 || s == 402 {
			if err := repo.Complete(key, path, s, bw.body.Bytes()); err != nil {
				log.Printf("idempotency complete: %v", err) // log the error if there is an issue completing the idempotency record, but do not reject the request as the response has already been sent
			}
		} else if err := repo.Release(key, path); err != nil {
			log.Printf("idempotency release: %v", err) // log the error if there is an issue releasing the idempotency record, but do not reject the request as the response has already been sent
		}
	}
}
