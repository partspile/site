package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/parts-pile/site/logger"
	"github.com/pgvector/pgvector-go"
)

var (
	db   *sqlx.DB
	once sync.Once
)

// Init initializes the database connection
func Init(databaseURL string) error {
	var err error
	once.Do(func() {
		db, err = sqlx.Open("pgx", databaseURL)
		if err != nil {
			logger.Error("Failed to open database",
				"error", err, "databaseURL", databaseURL)
			return
		}

		// Test the connection
		if err = db.Ping(); err != nil {
			logger.Error("Failed to ping database",
				"error", err, "databaseURL", databaseURL)
			return
		}

		logger.Info("Database initialized successfully",
			"databaseURL", databaseURL)
	})
	return err
}

// prettyPrintSQL formats a SQL query string for better readability
func prettyPrintSQL(query string) string {
	// Remove extra whitespace
	query = strings.TrimSpace(query)

	// Add newlines after major SQL keywords
	// Order matters: longer keywords first to avoid partial matches
	keywords := []string{
		"LEFT JOIN", "RIGHT JOIN", "INNER JOIN", "FULL OUTER JOIN",
		"GROUP BY", "ORDER BY", "INSERT INTO", "DELETE FROM",
		"SELECT", "FROM", "JOIN", "WHERE", "HAVING", "UNION",
		"UPDATE", "SET", "VALUES", "ON", "AND", "OR",
	}

	formatted := query
	for _, keyword := range keywords {
		// Case-insensitive replacement with newline
		re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(keyword) + `\b`)
		formatted = re.ReplaceAllStringFunc(formatted, func(match string) string {
			return "\n" + match
		})
	}

	// Clean up multiple newlines and add indentation
	lines := strings.Split(formatted, "\n")
	var result []string
	indentLevel := 0

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Decrease indent before certain keywords
		if strings.HasPrefix(strings.ToUpper(line), "WHERE") ||
			strings.HasPrefix(strings.ToUpper(line), "ORDER BY") ||
			strings.HasPrefix(strings.ToUpper(line), "GROUP BY") ||
			strings.HasPrefix(strings.ToUpper(line), "HAVING") {
			indentLevel = 0
		}

		// Add indentation
		indented := strings.Repeat("  ", indentLevel) + line
		result = append(result, indented)

		// Increase indent after certain keywords
		if strings.HasPrefix(strings.ToUpper(line), "SELECT") ||
			strings.HasPrefix(strings.ToUpper(line), "FROM") ||
			strings.HasPrefix(strings.ToUpper(line), "JOIN") ||
			strings.HasPrefix(strings.ToUpper(line), "WHERE") {
			indentLevel++
		}

		// Reset indent level for new major clauses
		if i < len(lines)-1 {
			nextLine := strings.ToUpper(strings.TrimSpace(lines[i+1]))
			if strings.HasPrefix(nextLine, "WHERE") ||
				strings.HasPrefix(nextLine, "ORDER BY") ||
				strings.HasPrefix(nextLine, "GROUP BY") ||
				strings.HasPrefix(nextLine, "HAVING") {
				indentLevel = 0
			}
		}
	}

	return strings.Join(result, "\n")
}

// formatArgs formats query arguments for display, truncating slices and vectors
func formatArgs(args []interface{}) []interface{} {
	formatted := make([]interface{}, len(args))
	for i, arg := range args {
		if arg == nil {
			formatted[i] = arg
			continue
		}

		val := reflect.ValueOf(arg)
		typ := reflect.TypeOf(arg)

		// Check if arg is a pgvector.Vector (has Slice() method)
		if vec, ok := arg.(pgvector.Vector); ok {
			slice := vec.Slice()
			formatted[i] = formatSlice(slice, len(slice))
			continue
		}

		// Use reflection to check for pgvector.Vector type
		if typ != nil && typ.String() == "pgvector.Vector" {
			if val.Kind() == reflect.Struct {
				sliceMethod := val.MethodByName("Slice")
				if sliceMethod.IsValid() {
					results := sliceMethod.Call(nil)
					if len(results) > 0 {
						slice := results[0].Interface().([]float32)
						formatted[i] = formatSlice(slice, len(slice))
						continue
					}
				}
			}
		}

		// Check if arg is a slice or array
		if val.Kind() == reflect.Slice || val.Kind() == reflect.Array {
			length := val.Len()
			if length == 0 {
				formatted[i] = "[]"
				continue
			}

			// Extract first few elements
			previewCount := 2
			if length < previewCount {
				previewCount = length
			}

			preview := make([]interface{}, previewCount)
			for j := 0; j < previewCount; j++ {
				preview[j] = val.Index(j).Interface()
			}

			// Format based on element type
			if length > previewCount {
				formatted[i] = fmt.Sprintf("[%v, %v, ...] (%d elements)", preview[0], preview[1], length)
			} else if length == 2 {
				formatted[i] = fmt.Sprintf("[%v, %v]", preview[0], preview[1])
			} else {
				formatted[i] = fmt.Sprintf("[%v]", preview[0])
			}
			continue
		}

		formatted[i] = arg
	}
	return formatted
}

// formatSlice formats a float32 slice for display
func formatSlice(slice []float32, length int) string {
	if length > 2 {
		return fmt.Sprintf("[%g, %g, ...] (%d values)", slice[0], slice[1], length)
	} else if length == 2 {
		return fmt.Sprintf("[%g, %g]", slice[0], slice[1])
	} else if length == 1 {
		return fmt.Sprintf("[%g]", slice[0])
	}
	return "[]"
}

// Query executes a query that returns rows
func Query(query string, args ...interface{}) (*sql.Rows, error) {
	/*
		fmt.Println("")
		fmt.Println("=== SQL Query ===")
		fmt.Println(prettyPrintSQL(query))
		fmt.Println("Args:", formatArgs(args))
		fmt.Println("================")
		fmt.Println("")
	*/
	return db.Query(query, args...)
}

// QueryRow executes a query that returns a single row
func QueryRow(query string, args ...interface{}) *sql.Row {
	/*
		fmt.Println("")
		fmt.Println("=== SQL QueryRow ===")
		fmt.Println(prettyPrintSQL(query))
		fmt.Println("Args:", formatArgs(args))
		fmt.Println("===================")
		fmt.Println("")
	*/
	return db.QueryRow(query, args...)
}

// Exec executes a query that doesn't return rows
func Exec(query string, args ...interface{}) (sql.Result, error) {
	/*
		fmt.Println("")
		fmt.Println("=== SQL Exec ===")
		fmt.Println(prettyPrintSQL(query))
		fmt.Println("Args:", args)
		fmt.Println("===============")
		fmt.Println("")
	*/
	return db.Exec(query, args...)
}

// Begin starts a new transaction
func Begin() (*sql.Tx, error) {
	return db.Begin()
}

// Select executes a query and scans the results into dest (slice)
func Select(dest interface{}, query string, args ...interface{}) error {
	/*
		fmt.Println("")
		fmt.Println("=== SQL Select ===")
		fmt.Println(prettyPrintSQL(query))
		fmt.Println("Args:", formatArgs(args))
		fmt.Println("=================")
		fmt.Println("")
	*/
	return db.Select(dest, query, args...)
}

// Ping verifies the database connection is alive
func Ping() error {
	return db.Ping()
}

// QueryJSON executes a query that returns a single JSON result and unmarshals it into dst
func QueryJSON(dst interface{}, query string, args ...interface{}) error {
	var jsonResult []byte
	/*
		fmt.Println("")
		fmt.Println("=== SQL QueryJSON ===")
		fmt.Println(prettyPrintSQL(query))
		fmt.Println("Args:", formatArgs(args))
		fmt.Println("===============")
		fmt.Println("")
	*/
	err := db.QueryRow(query, args...).Scan(&jsonResult)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonResult, dst)
}
