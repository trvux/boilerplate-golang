package response

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tranvux/boilerplate_golang/pkg/apperr"
	"github.com/tranvux/boilerplate_golang/pkg/logger"
	"go.uber.org/zap"
)

type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
}

type ErrorResponse struct {
	Success bool       `json:"success"`
	Error   ErrorValue `json:"error"`
}

type ErrorValue struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

func JSON(c *gin.Context, httpStatus int, data interface{}) {
	c.JSON(httpStatus, SuccessResponse{
		Success: true,
		Data:    data,
	})
}

func OK(c *gin.Context, data interface{}) {
	JSON(c, http.StatusOK, data)
}

func Created(c *gin.Context, data interface{}) {
	JSON(c, http.StatusCreated, data)
}

func Error(c *gin.Context, err error) {
	if err == nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorValue{
				Code:    "UNKNOWN_ERROR",
				Message: "An unexpected error occurred",
			},
		})
		return
	}

	var appErr *apperr.AppError
	if errors.As(err, &appErr) {
		httpStatus := mapErrorTypeToHTTPStatus(appErr.Type)

		// Log warning for validation issues, errors for internal issues
		if appErr.Type == apperr.TypeInternal {
			logger.Error("Internal server error occurred", zap.Error(appErr.Err))
		} else {
			logger.Warn("Business domain error returned",
				zap.String("type", string(appErr.Type)),
				zap.String("code", appErr.Code),
				zap.String("msg", appErr.Message),
				zap.Error(appErr.Err),
			)
		}

		c.JSON(httpStatus, ErrorResponse{
			Success: false,
			Error: ErrorValue{
				Code:    appErr.Code,
				Message: appErr.Message,
				Details: getErrorDetails(appErr.Err),
			},
		})
		return
	}

	// External/raw unmapped errors are handled as internal server errors
	logger.Error("Unhandled raw error occurred", zap.Error(err))

	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Success: false,
		Error: ErrorValue{
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "A generic system error occurred",
		},
	})
}

func mapErrorTypeToHTTPStatus(errType apperr.ErrorType) int {
	switch errType {
	case apperr.TypeValidation:
		return http.StatusBadRequest
	case apperr.TypeNotFound:
		return http.StatusNotFound
	case apperr.TypeConflict:
		return http.StatusConflict
	case apperr.TypeUnauthorized:
		return http.StatusUnauthorized
	case apperr.TypeForbidden:
		return http.StatusForbidden
	case apperr.TypeInternal:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

func getErrorDetails(err error) interface{} {
	if err == nil {
		return nil
	}
	// In production, we don't return nested internal technical error details to prevent leaks.
	// But in development, we can expose the underlying error string for easier debugging.
	return err.Error()
}
