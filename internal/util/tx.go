// internal/util/tx.go
package util

import (
	"log"

	"github.com/jmoiron/sqlx"
)

// SafeRollback выполняет Rollback и логирует ошибку, если нужно
func SafeRollback(tx *sqlx.Tx) {
	if err := tx.Rollback(); err != nil && err.Error() != "sql: transaction has already been committed or rolled back" {
		log.Printf("⚠️  rollback error: %v", err)
	}
}
