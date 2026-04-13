package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"

	"payloop/internal/core/port"
	"payloop/internal/lib"
)

// DatabaseTrx wraps every request in a database transaction.
type DatabaseTrx struct {
	handler   lib.RequestHandler
	logger    port.Logger
	primaryDb lib.Database
}

// statusInList checks if a status is in the provided list.
func statusInList(status int, statusList []int) bool {
	for _, i := range statusList {
		if i == status {
			return true
		}
	}
	return false
}

// NewDatabaseTrx creates a new DatabaseTrx middleware.
func NewDatabaseTrx(
	primaryDb lib.Database,
	handler lib.RequestHandler,
	logger port.Logger,
) DatabaseTrx {
	return DatabaseTrx{
		handler:   handler,
		logger:    logger,
		primaryDb: primaryDb,
	}
}

// Setup registers the database transaction middleware on the gin engine.
func (m DatabaseTrx) Setup() {
	m.logger.Debug("setting up database transaction middleware")

	m.handler.Gin.Use(func(c *gin.Context) {
		txHandle, err := m.primaryDb.Begin(c.Request.Context())
		if err != nil {
			m.logger.Error("error beginning database transaction", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			c.Abort()
			return
		}

		defer func() {
			if r := recover(); r != nil {
				m.logger.Error("recover(), rolling back..",
					"err", r,
					"stack", string(debug.Stack()),
					"url", c.Request.URL.String())
				_ = txHandle.Rollback(c.Request.Context())
			}
		}()

		reqCtx := context.WithValue(c.Request.Context(), lib.DBTransaction, txHandle)
		c.Request = c.Request.WithContext(reqCtx)
		c.Next()

		// commit transaction on success status
		if statusInList(c.Writer.Status(), []int{http.StatusOK, http.StatusCreated, http.StatusNoContent}) {
			m.logger.Debug("committing transaction")
			if err := txHandle.Commit(c.Request.Context()); err != nil {
				m.logger.Error("trx commit error: ", err)
			}
		} else {
			m.logger.Debug("rolling back transaction due to status code:", slog.Int("err", c.Writer.Status()))
			_ = txHandle.Rollback(c.Request.Context())
		}
	})
}
