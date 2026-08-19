package kernel

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorResponse is the standard JSON error body.
//
// `details` is intentionally omitted in 5xx responses — the real error is
// logged server-side so we don't leak Go internals / library messages /
// hostnames / paths to API clients. If you need the internal reason for
// debugging, read the api-gateway log.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Details string `json:"details,omitempty"`
}

// User-facing error messages. Keep them short and actionable — callers
// shouldn't see Go error strings or stack info.
const (
	msgConfigInvalid    = "Configuration is incomplete. Update the connection settings and try again."
	msgUpstreamFailed   = "Unable to reach the remote host. Check the host address and authentication."
	msgInternalGeneric  = "Internal error. See server logs."
	msgBadRequestFields = "One or more required fields are missing or invalid."
)

// RespondError writes a structured error response based on the error type.
//
// 4xx → user-facing message with optional sanitized details.
// 5xx → generic message; real error logged via log.Printf so it shows up
// in api-gateway.log but never reaches the client.
func RespondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error(), Code: "not_found"})
	case errors.Is(err, ErrUnauthorized):
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: err.Error(), Code: "unauthorized"})
	case errors.Is(err, ErrForbidden):
		c.JSON(http.StatusForbidden, ErrorResponse{Error: err.Error(), Code: "forbidden"})
	case errors.Is(err, ErrBadRequest):
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: msgBadRequestFields, Code: "bad_request", Details: err.Error()})
	case errors.Is(err, ErrConflict):
		c.JSON(http.StatusConflict, ErrorResponse{Error: err.Error(), Code: "conflict"})
	case errors.Is(err, ErrTooManyRequests):
		c.JSON(http.StatusTooManyRequests, ErrorResponse{Error: err.Error(), Code: "rate_limited"})
	case errors.Is(err, ErrConfigInvalid):
		c.JSON(http.StatusUnprocessableEntity, ErrorResponse{Error: msgConfigInvalid, Code: "config_invalid", Details: err.Error()})
	case errors.Is(err, ErrUpstream):
		c.JSON(http.StatusBadGateway, ErrorResponse{Error: msgUpstreamFailed, Code: "upstream_failed", Details: err.Error()})
	default:
		// Unknown / 500-class — log server-side, return generic.
		method, path := "<unknown>", "<unknown>"
		if c.Request != nil {
			method = c.Request.Method
			path = c.Request.URL.Path
		}
		log.Printf("api 500 %s %s: %v", method, path, err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: msgInternalGeneric, Code: "internal"})
	}
}

// RespondOK writes a JSON body with 200.
func RespondOK(c *gin.Context, body any) {
	c.JSON(http.StatusOK, body)
}

// RespondCreated writes a JSON body with 201.
func RespondCreated(c *gin.Context, body any) {
	c.JSON(http.StatusCreated, body)
}
