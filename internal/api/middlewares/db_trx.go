package middlewares

import (
	"net/http"
	"payloop/internal/lib"

	"github.com/gin-gonic/gin"
)

// DatabaseTrx middleware for transactions support for database
type DatabaseTrx struct {
	handler lib.RequestHandler
	logger  lib.Logger
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
	logger lib.Logger,
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
		txHandle, _ := m.db.Begin(c.Request.Context())
		m.logger.Debug("beginning database transaction")

		defer func() {
			if r := recover(); r != nil {
				_ = txHandle.Rollback(c.Request.Context())
			}
		}()

		c.Set(lib.DBTransaction, txHandle)
		c.Next()

		// commit transaction on success status
		if statusInList(c.Writer.Status(), []int{http.StatusOK, http.StatusCreated, http.StatusNoContent}) {
			m.logger.Debug("committing transactions")
			if err := txHandle.Commit(c.Request.Context()).Error; err != nil {
				m.logger.Error("trx commit error: ", err)
			}
		} else {
			m.logger.Debug("rolling back transaction due to status code:", c.Writer.Status())
			_ = txHandle.Rollback(c.Request.Context())
		}
	})
}
