package kernel

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorResponse is the standard JSON error body.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Details string `json:"details,omitempty"`
}

// RespondError writes a structured error response based on the error type.
func RespondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error(), Code: "not_found"})
	case errors.Is(err, ErrUnauthorized):
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: err.Error(), Code: "unauthorized"})
	case errors.Is(err, ErrForbidden):
		c.JSON(http.StatusForbidden, ErrorResponse{Error: err.Error(), Code: "forbidden"})
	case errors.Is(err, ErrBadRequest):
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error(), Code: "bad_request"})
	case errors.Is(err, ErrConflict):
		c.JSON(http.StatusConflict, ErrorResponse{Error: err.Error(), Code: "conflict"})
	case errors.Is(err, ErrTooManyRequests):
		c.JSON(http.StatusTooManyRequests, ErrorResponse{Error: err.Error(), Code: "rate_limited"})
	default:
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "internal error", Code: "internal", Details: err.Error()})
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
