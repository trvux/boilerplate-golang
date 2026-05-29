package app

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tranvux/boilerplate_golang/pkg/logger"
	"github.com/tranvux/boilerplate_golang/pkg/response"
	"go.uber.org/zap"
)

const HeaderXRequestID = "X-Request-ID"

func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader(HeaderXRequestID)
		if reqID == "" {
			reqID = uuid.New().String()
		}
		c.Set(HeaderXRequestID, reqID)
		c.Header(HeaderXRequestID, reqID)
		c.Next()
	}
}

func ZapLoggerMiddleware(log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		reqID, _ := c.Get(HeaderXRequestID)

		fields := []zap.Field{
			zap.Int("status", status),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.Duration("latency", latency),
			zap.String("user_agent", c.Request.UserAgent()),
		}

		if reqID != nil {
			fields = append(fields, zap.String("request_id", reqID.(string)))
		}

		if status >= 500 {
			log.Error("HTTP Server Internal Error", fields...)
		} else if status >= 400 {
			log.Warn("HTTP Request Warning Client Error", fields...)
		} else {
			log.Info("HTTP Request Processed", fields...)
		}
	}
}

func RecoveryMiddleware(log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Stack trace listing
				stack := debug.Stack()
				reqID, _ := c.Get(HeaderXRequestID)

				fields := []zap.Field{
					zap.Any("panic_error", err),
					zap.String("stack", string(stack)),
				}
				if reqID != nil {
					fields = append(fields, zap.String("request_id", reqID.(string)))
				}

				log.Error("Server Panic Recovered", fields...)

				// Centralized error response
				c.AbortWithStatusJSON(http.StatusInternalServerError, response.ErrorResponse{
					Success: false,
					Error: response.ErrorValue{
						Code:    "PANIC_RECOVERED",
						Message: fmt.Sprintf("A critical server panic was intercepted: %v", err),
					},
				})
			}
		}()
		c.Next()
	}
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Request-ID")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
