package models

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIResponse is the standard API response envelope.
type APIResponse struct {
	Data  interface{} `json:"data"`
	Error *APIError   `json:"error"`
}

// APIError represents an error in the standard API response.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// SuccessResponse sends a JSON response with the standard success format.
func SuccessResponse(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, APIResponse{
		Data:  data,
		Error: nil,
	})
}

// ErrorResponse sends a JSON response with the standard error format.
func ErrorResponse(c *gin.Context, statusCode int, code string, message string) {
	c.JSON(statusCode, APIResponse{
		Data: nil,
		Error: &APIError{
			Code:    code,
			Message: message,
		},
	})
}

// Common error response helpers.

// BadRequestError sends a 400 error response.
func BadRequestError(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusBadRequest, "BAD_REQUEST", message)
}

// NotFoundError sends a 404 error response.
func NotFoundError(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusNotFound, "NOT_FOUND", message)
}

// InternalError sends a 500 error response.
func InternalError(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", message)
}

// UnauthorizedError sends a 401 error response.
func UnauthorizedError(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", message)
}

// ForbiddenError sends a 403 error response.
func ForbiddenError(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusForbidden, "FORBIDDEN", message)
}

// ConflictError sends a 409 error response.
func ConflictError(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusConflict, "CONFLICT", message)
}

// TooManyRequestsError sends a 429 error response.
func TooManyRequestsError(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusTooManyRequests, "RATE_LIMITED", message)
}
