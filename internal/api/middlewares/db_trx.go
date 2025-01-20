package middlewares

import (
	"github.com/dipeshdulal/clean-gin/constants"
	"net/http"
	"payloop/internal/db"
	"payloop/internal/lib"

	"github.com/gin-gonic/gin"
)

// DatabaseTrx middleware for transactions support for database
type DatabaseTrx struct {
	handler lib.RequestHandler
	logger  lib.Logger
	db      db.Database
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
	db db.Database,
) DatabaseTrx {
	return DatabaseTrx{
		handler: handler,
		logger:  logger,
		db:      db,
	}
}

// Setup sets up database transaction middleware
func (m DatabaseTrx) Setup() {
	m.logger.Info("setting up database transaction middleware")

	m.handler.Gin.Use(func(c *gin.Context) {
		txHandle, _ := m.db.Begin(c.Request.Context())
		m.logger.Info("beginning database transaction")

		defer func() {
			if r := recover(); r != nil {
				_ = txHandle.Rollback(c.Request.Context())
			}
		}()

		c.Set(constants.DBTransaction, txHandle)
		c.Next()

		// commit transaction on success status
		if statusInList(c.Writer.Status(), []int{http.StatusOK, http.StatusCreated, http.StatusNoContent}) {
			m.logger.Info("committing transactions")
			if err := txHandle.Commit(c.Request.Context()).Error; err != nil {
				m.logger.Error("trx commit error: ", err)
			}
		} else {
			m.logger.Info("rolling back transaction due to status code: 500")
			_ = txHandle.Rollback(c.Request.Context())
		}
	})
}
