package middlewares

import (
	"log/slog"
	"net/http"
	"payloop/internal/application/lib/logger"
	"payloop/internal/lib"

	"github.com/gin-gonic/gin"
)

// DatabaseTrx middleware for transactions support for database
type DatabaseTrx struct {
	handler lib.RequestHandler
	logger  logger.Logger
	db      lib.Database
}

// statusInList function checks if context writer status is in provided list
func statusInList(status int, statusList []int) bool {
	for _, i := range statusList {
		if i == status {
			return true
		}
	}
	return false
}

// NewDatabaseTrx creates new database transactions middleware
func NewDatabaseTrx(
	handler lib.RequestHandler,
	logger logger.Logger,
	db lib.Database,
) DatabaseTrx {
	return DatabaseTrx{
		handler: handler,
		logger:  logger,
		db:      db,
	}
}

// Setup sets up database transaction middleware
func (m DatabaseTrx) Setup() {
	m.logger.Debug("setting up database transaction middleware")

	m.handler.Gin.Use(func(c *gin.Context) {
		txHandle, err := m.db.Begin(c.Request.Context())
		if err != nil {
			m.logger.Error("error beginning database transaction", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			c.Abort()
			return
		}
		m.logger.Debug("beginning database transaction")

		defer func() {
			if r := recover(); r != nil {
				m.logger.Error("recover(), rolling back..", r)
				_ = txHandle.Rollback(c.Request.Context())
			}
		}()

		c.Set(lib.DBTransaction, txHandle)
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
