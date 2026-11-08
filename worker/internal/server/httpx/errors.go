package httpx

import (
	"errors"
	"fmt"
	"log"
	"net/http"
)

const (
	notFoundMessage       = "The requested URL was not found on the server. If you entered the URL manually please check your spelling and try again."
	internalErrorMessage  = "The server encountered an internal error and was unable to complete your request. Either the server is overloaded or there is an error in the application."
	unauthorizedMessage   = "The server could not verify that you are authorized to access the URL requested."
	payloadTooLargeReason = "The data value transmitted exceeds the capacity limit."
)

type Error struct {
	Status  int
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%d %s", e.Status, e.Message)
}

func NewError(status int, message string) *Error {
	return &Error{Status: status, Message: message}
}

func BadRequest(message string) *Error {
	return NewError(http.StatusBadRequest, message)
}

func Unauthorized(message string) *Error {
	if message == "" {
		message = unauthorizedMessage
	}
	return NewError(http.StatusUnauthorized, message)
}

func Forbidden(message string) *Error {
	return NewError(http.StatusForbidden, message)
}

func NotFound(message string) *Error {
	if message == "" {
		message = notFoundMessage
	}
	return NewError(http.StatusNotFound, message)
}

func Conflict(message string) *Error {
	return NewError(http.StatusConflict, message)
}

func PayloadTooLarge() *Error {
	return NewError(http.StatusRequestEntityTooLarge, payloadTooLargeReason)
}

func writeError(w http.ResponseWriter, err error, logger *log.Logger) {
	var httpError *Error
	if !errors.As(err, &httpError) {
		logger.Printf("[ERROR] %v", err)
		httpError = NewError(http.StatusInternalServerError, internalErrorMessage)
	}
	WriteJSON(w, httpError.Status, map[string]any{"error": httpError.Message, "status": httpError.Status})
}
