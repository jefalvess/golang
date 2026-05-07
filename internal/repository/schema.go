package repository

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
)

// LoadSchema lê o schema de um arquivo para que produção e testes possam escolher a origem.
func LoadSchema(filePath string) (string, error) {
	schema, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("read schema %s: %w", filePath, err)
	}

	return string(schema), nil
}

// MigrateSchema aplica um schema SQL arbitrário ao banco, permitindo testes com schemas customizados.
func MigrateSchema(db *sql.DB, schema string) error {
	if strings.TrimSpace(schema) == "" {
		return fmt.Errorf("schema is empty")
	}

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("exec schema: %w", err)
	}

	return nil
}
