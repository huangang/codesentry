package response

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response is the unified API response format.
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// AppError represents a structured application error with HTTP status and error code.
type AppError struct {
	HTTPStatus int    // HTTP status code (e.g. 400, 404, 500)
	Code       int    // Application-level error code
	Message    string // Human-readable error message
}

func (e *AppError) Error() string {
	return e.Message
}

// Pre-defined error constructors

func NewBadRequest(msg string) *AppError {
	return &AppError{HTTPStatus: http.StatusBadRequest, Code: 400, Message: msg}
}

func NewUnauthorized(msg string) *AppError {
	return &AppError{HTTPStatus: http.StatusUnauthorized, Code: 401, Message: msg}
}

func NewForbidden(msg string) *AppError {
	return &AppError{HTTPStatus: http.StatusForbidden, Code: 403, Message: msg}
}

func NewNotFound(msg string) *AppError {
	return &AppError{HTTPStatus: http.StatusNotFound, Code: 404, Message: msg}
}

func NewConflict(msg string) *AppError {
	return &AppError{HTTPStatus: http.StatusConflict, Code: 409, Message: msg}
}

func NewServerError(msg string) *AppError {
	return &AppError{HTTPStatus: http.StatusInternalServerError, Code: 500, Message: msg}
}

// --- Gin response helpers ---

// Success sends a 200 OK response with data.
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "ok",
		Data:    data,
	})
}

// Created sends a 201 Created response with data.
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{
		Code:    0,
		Message: "created",
		Data:    data,
	})
}

// Error sends an error response. If err is an *AppError, its code and status
// are used; otherwise a generic 500 internal server error is returned.
func Error(c *gin.Context, err error) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		c.JSON(appErr.HTTPStatus, Response{
			Code:    appErr.Code,
			Message: appErr.Message,
		})
		return
	}
	c.JSON(http.StatusInternalServerError, Response{
		Code:    500,
		Message: err.Error(),
	})
}

// Convenience error response functions

func BadRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, Response{Code: 400, Message: msg})
}

func Unauthorized(c *gin.Context, msg string) {
	c.JSON(http.StatusUnauthorized, Response{Code: 401, Message: msg})
}

func Forbidden(c *gin.Context, msg string) {
	c.JSON(http.StatusForbidden, Response{Code: 403, Message: msg})
}

func NotFound(c *gin.Context, msg string) {
	c.JSON(http.StatusNotFound, Response{Code: 404, Message: msg})
}

func ServerError(c *gin.Context, msg string) {
	c.JSON(http.StatusInternalServerError, Response{Code: 500, Message: msg})
}
