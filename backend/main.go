package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Response struct for our API
type Response struct {
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Status    string    `json:"status"`
}

// Entry represents a row in our data table
type Entry struct {
	ID      int    `json:"id"`
	URL     string `json:"url"`
	User    string `json:"user"`
	Pass    string `json:"pass"`
	Created string `json:"created"`
}

// PaginationResponse wraps data with pagination metadata
type PaginationResponse struct {
	Items       []Entry `json:"items"`
	Total       int     `json:"total"`
	Page        int     `json:"page"`
	PageSize    int     `json:"pageSize"`
	TotalPages  int     `json:"totalPages"`
	HasNext     bool    `json:"hasNext"`
	HasPrevious bool    `json:"hasPrevious"`
	NextPage    int     `json:"nextPage"`
	PrevPage    int     `json:"prevPage"`
	Offset      int     `json:"offset"`
}

// Database connection pool
var dbPool *pgxpool.Pool
var connString string

// Global instance of the log watcher
var logWatcher *LogWatcher

// initDB initializes the PostgreSQL connection pool
func initDB() error {
	// Get PostgreSQL connection string from environment variable or use default
	connString = os.Getenv("DATABASE_URL")
	if connString == "" {
		// Use default connection string if not provided - connect to default postgres database
		connString = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	}

	// Create a connection pool
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return fmt.Errorf("failed to parse DB config: %w", err)
	}

	// Set up connection pool configurations
	config.MaxConns = 10

	// Create the connection pool
	dbPool, err = pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test the connection
	if err := dbPool.Ping(context.Background()); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Create the entries table if it doesn't exist
	_, err = dbPool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS entries (
			id SERIAL PRIMARY KEY,
			url TEXT NOT NULL,
			username TEXT NOT NULL,
			password TEXT NOT NULL,
			created TEXT NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Create the processed_log_files table to track already processed files
	_, err = dbPool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS processed_log_files (
			id SERIAL PRIMARY KEY,
			filename TEXT NOT NULL UNIQUE,
			processed_at TIMESTAMP NOT NULL DEFAULT NOW(),
			entries_added INT NOT NULL DEFAULT 0
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create processed_log_files table: %w", err)
	}

	// Seed the database with initial data if it's empty
	var count int
	err = dbPool.QueryRow(context.Background(), "SELECT COUNT(*) FROM entries").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to count entries: %w", err)
	}

	if count == 0 {
		sampleEntries := []Entry{
			{URL: "https://github.com", User: "gituser123", Pass: "github123", Created: "2025-05-18"},
			{URL: "https://google.com", User: "googleuser", Pass: "google456", Created: "2025-05-19"},
			{URL: "https://microsoft.com", User: "msftuser", Pass: "ms789", Created: "2025-05-20"},
			{URL: "https://apple.com", User: "appleuser", Pass: "apple101", Created: "2025-05-20"},
			{URL: "https://amazon.com", User: "amazonuser", Pass: "amazon202", Created: "2025-05-20"},
		}

		for _, entry := range sampleEntries {
			_, err = dbPool.Exec(
				context.Background(),
				"INSERT INTO entries (url, username, password, created) VALUES ($1, $2, $3, $4)",
				entry.URL, entry.User, entry.Pass, entry.Created,
			)
			if err != nil {
				return fmt.Errorf("failed to seed database: %w", err)
			}
		}
	}

	log.Println("Database connected and initialized successfully")
	return nil
}

func main() {
	// Initialize database connection
	if err := initDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer dbPool.Close()
	// Initialize log watcher for the log directory
	var err error
	logWatcher, err = NewLogWatcher("./data")
	if err != nil {
		log.Printf("Warning: Failed to initialize log watcher: %v", err)
	} else {
		if err := logWatcher.Start(); err != nil {
			log.Printf("Warning: Failed to start log watcher: %v", err)
		} else {
			// Ensure the watcher stops when the application exits
			defer logWatcher.Stop()
			log.Println("Log watcher started successfully")
		}
	}

	// Initialize a new Fiber app
	app := fiber.New(fiber.Config{
		AppName: "Hello World Go Fiber API",
	})

	// Add middleware
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{"Origin, Content-Type, Accept"},
	}))

	// API routes
	api := app.Group("/api")
	// Define a route for the GET method on the '/api/hello' path
	api.Get("/hello", func(c fiber.Ctx) error {
		// Return a JSON response
		return c.JSON(Response{
			Message:   "Hello, World 👋 from Go Fiber API!",
			Timestamp: time.Now(),
			Status:    "success",
		})
	})

	// Get entries with pagination
	api.Get("/entries", func(c fiber.Ctx) error {
		ctx := context.Background()

		// Parse pagination parameters from query string
		pageStr := c.Query("page", "1") // Default to page 1
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			page = 1
		}

		pageSizeStr := c.Query("pageSize", "10") // Default to 10 items per page
		pageSize, err := strconv.Atoi(pageSizeStr)
		if err != nil || pageSize < 1 || pageSize > 200 {
			pageSize = 10 // Ensure reasonable limits
		}

		offset := (page - 1) * pageSize

		// Get total count for pagination metadata
		var totalCount int
		err = dbPool.QueryRow(ctx, "SELECT COUNT(*) FROM entries").Scan(&totalCount)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "Failed to count entries",
				"details": err.Error(),
			})
		}

		// Query entries with pagination
		entriesQuery := "SELECT id, url, username, password, created FROM entries ORDER BY id DESC LIMIT $1 OFFSET $2"
		rows, err := dbPool.Query(ctx, entriesQuery, pageSize, offset)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "Failed to query database",
				"details": err.Error(),
			})
		}
		defer rows.Close()

		// Process query results
		var results []Entry
		for rows.Next() {
			var entry Entry
			if err := rows.Scan(&entry.ID, &entry.URL, &entry.User, &entry.Pass, &entry.Created); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error":   "Failed to scan row",
					"details": err.Error(),
				})
			}
			results = append(results, entry)
		}

		// Calculate pagination metadata
		totalPages := (totalCount + pageSize - 1) / pageSize // Ceiling division
		hasNext := page < totalPages
		hasPrevious := page > 1
		nextPage := page + 1
		if !hasNext {
			nextPage = page
		}
		prevPage := page - 1
		if !hasPrevious {
			prevPage = page
		}

		// Create pagination response
		response := PaginationResponse{
			Items:       results,
			Total:       totalCount,
			Page:        page,
			PageSize:    pageSize,
			TotalPages:  totalPages,
			HasNext:     hasNext,
			HasPrevious: hasPrevious,
			NextPage:    nextPage,
			PrevPage:    prevPage,
			Offset:      offset,
		}

		return c.JSON(response)
	})

	// Search entries with pagination
	api.Get("/search", func(c fiber.Ctx) error {
		query := strings.ToLower(c.Query("q", ""))
		urlFilter := strings.ToLower(c.Query("url", ""))
		userFilter := strings.ToLower(c.Query("user", ""))
		passFilter := strings.ToLower(c.Query("pass", ""))

		// Parse pagination parameters
		pageStr := c.Query("page", "1")
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			page = 1
		}

		pageSizeStr := c.Query("pageSize", "10")
		pageSize, err := strconv.Atoi(pageSizeStr)
		if err != nil || pageSize < 1 || pageSize > 200 {
			pageSize = 10
		}

		offset := (page - 1) * pageSize

		// Base query with placeholders for search conditions
		baseSQL := "SELECT id, url, username, password, created FROM entries WHERE 1=1"
		var conditions []string
		var params []interface{}
		paramCount := 1

		// Add conditions based on provided filters
		if query != "" {
			conditions = append(conditions, fmt.Sprintf("(LOWER(url) LIKE $%d OR LOWER(username) LIKE $%d OR LOWER(password) LIKE $%d)",
				paramCount, paramCount+1, paramCount+2))
			params = append(params, "%"+query+"%", "%"+query+"%", "%"+query+"%")
			paramCount += 3
		}

		if urlFilter != "" {
			conditions = append(conditions, fmt.Sprintf("LOWER(url) LIKE $%d", paramCount))
			params = append(params, "%"+urlFilter+"%")
			paramCount++
		}

		if userFilter != "" {
			conditions = append(conditions, fmt.Sprintf("LOWER(username) LIKE $%d", paramCount))
			params = append(params, "%"+userFilter+"%")
			paramCount++
		}

		if passFilter != "" {
			conditions = append(conditions, fmt.Sprintf("LOWER(password) LIKE $%d", paramCount))
			params = append(params, "%"+passFilter+"%")
			paramCount++
		}

		// Build the complete SQL query with conditions
		finalSQL := baseSQL
		if len(conditions) > 0 {
			finalSQL += " AND " + strings.Join(conditions, " AND ")
		}

		// Get total count for pagination metadata
		countSQL := "SELECT COUNT(*) FROM entries"
		if len(conditions) > 0 {
			countSQL = "SELECT COUNT(*) FROM (" + finalSQL + ") as filtered_entries"
		}

		var totalCount int
		if len(conditions) > 0 {
			err = dbPool.QueryRow(context.Background(), countSQL, params...).Scan(&totalCount)
		} else {
			err = dbPool.QueryRow(context.Background(), countSQL).Scan(&totalCount)
		}

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "Failed to count filtered entries",
				"details": err.Error(),
			})
		}

		// Add ordering and pagination to the final query
		finalSQL += " ORDER BY id DESC LIMIT $" + strconv.Itoa(paramCount) + " OFFSET $" + strconv.Itoa(paramCount+1)
		params = append(params, pageSize, offset)

		ctx := context.Background()
		// PostgreSQL automatically caches execution plans for parameterized queries
		rows, err := dbPool.Query(ctx, finalSQL, params...)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "Failed to search database",
				"details": err.Error(),
			})
		}
		defer rows.Close()

		// Process query results
		var results []Entry
		for rows.Next() {
			var entry Entry
			if err := rows.Scan(&entry.ID, &entry.URL, &entry.User, &entry.Pass, &entry.Created); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error":   "Failed to scan search results",
					"details": err.Error(),
				})
			}
			results = append(results, entry)
		}

		// Calculate pagination metadata
		totalPages := (totalCount + pageSize - 1) / pageSize // Ceiling division
		hasNext := page < totalPages
		hasPrevious := page > 1
		nextPage := page + 1
		if !hasNext {
			nextPage = page
		}
		prevPage := page - 1
		if !hasPrevious {
			prevPage = page
		}

		// Create pagination response
		response := PaginationResponse{
			Items:       results,
			Total:       totalCount,
			Page:        page,
			PageSize:    pageSize,
			TotalPages:  totalPages,
			HasNext:     hasNext,
			HasPrevious: hasPrevious,
			NextPage:    nextPage,
			PrevPage:    prevPage,
			Offset:      offset,
		}

		return c.JSON(response)
	})

	// Import logs endpoint
	api.Post("/import-logs", func(c fiber.Ctx) error {
		// Get the log directory from the request or use default
		logDir := c.FormValue("logDir")
		if logDir == "" {
			// Use default log directory
			logDir = "./data"
		}

		// Make sure the directory exists
		if _, err := os.Stat(logDir); os.IsNotExist(err) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Log directory does not exist",
			})
		}

		// Start the import process in a goroutine to avoid blocking
		go func() {
			if err := ParseLogDirectory(logDir); err != nil {
				log.Printf("Error importing logs: %v", err)
			}
		}()
		return c.JSON(fiber.Map{
			"message": "Log import started in background",
			"status":  "success",
		})
	})
	// Process a specific log file endpoint
	api.Post("/process-file", func(c fiber.Ctx) error {
		// Get the file path from the request
		filePath := c.FormValue("filePath")
		if filePath == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "File path is required",
			})
		}

		// Make sure the file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "File does not exist",
			})
		}

		// Check if the logWatcher is available
		if logWatcher == nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Log watcher is not initialized",
			})
		}

		// Process the file using the LogWatcher (which tracks processed files)
		count, err := logWatcher.ProcessFile(filePath)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "Failed to process log file",
				"details": err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"message": fmt.Sprintf("Processed file %s successfully", filepath.Base(filePath)),
			"entries": count,
			"status":  "success",
		})
	})
	// Get log watcher status endpoint
	api.Get("/watcher-status", func(c fiber.Ctx) error {
		// Get the log directory
		logDir := "./data"

		// Get list of files in log directory
		files, err := filepath.Glob(filepath.Join(logDir, "*.txt"))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to read log directory",
			})
		}

		// Get info about processed files
		var processedCount int
		var processedFiles []string

		if logWatcher != nil {
			logWatcher.mu.Lock()
			processedCount = len(logWatcher.processedFiles)
			for file := range logWatcher.processedFiles {
				processedFiles = append(processedFiles, file)
			}
			logWatcher.mu.Unlock()
		}

		// Return the status information
		return c.JSON(fiber.Map{
			"watching":       logDir,
			"fileCount":      len(files),
			"files":          files,
			"watcherActive":  logWatcher != nil,
			"processed":      processedCount,
			"processedFiles": processedFiles,
			"status":         "success",
		})
	})
	// Get total count of records in the database
	api.Get("/stats", func(c fiber.Ctx) error {
		// Query the total count from the entries table
		var count int
		err := dbPool.QueryRow(context.Background(), "SELECT COALESCE(SUM(entries_added), 0) FROM processed_log_files").Scan(&count)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "Failed to count entries",
				"details": err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"totalRecords": count,
			"status":       "success",
		})
	})

	// Get processed files list from database
	api.Get("/processed-files", func(c fiber.Ctx) error {
		// Query processed files from database with their details
		rows, err := dbPool.Query(context.Background(),
			"SELECT filename, processed_at, entries_added FROM processed_log_files ORDER BY processed_at DESC")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "Failed to query processed files",
				"details": err.Error(),
			})
		}
		defer rows.Close()

		// Create response structure
		type ProcessedFile struct {
			Filename    string    `json:"filename"`
			ProcessedAt time.Time `json:"processedAt"`
			Entries     int       `json:"entriesAdded"`
		}

		var result []ProcessedFile
		for rows.Next() {
			var file ProcessedFile
			if err := rows.Scan(&file.Filename, &file.ProcessedAt, &file.Entries); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error":   "Failed to scan row",
					"details": err.Error(),
				})
			}
			result = append(result, file)
		}

		if err := rows.Err(); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "Error iterating results",
				"details": err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"processedFiles": result,
			"count":          len(result),
			"status":         "success",
		})
	})

	// Find or remove duplicate entries in the database
	api.Get("/duplicates", func(c fiber.Ctx) error {
		// Check if we should remove duplicates or just report them
		shouldRemove := c.Query("remove", "false") == "true"

		ctx := context.Background()
		var duplicates []Entry
		var removed int

		if shouldRemove {
			// Start a transaction to ensure consistency
			tx, err := dbPool.Begin(ctx)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error":   "Failed to start transaction",
					"details": err.Error(),
				})
			}
			defer tx.Rollback(ctx) // will be ignored if transaction is committed

			// First identify duplicates
			rows, err := tx.Query(ctx, `
				WITH duplicates AS (
					SELECT id, url, username, password, created,
						ROW_NUMBER() OVER(PARTITION BY url, username, password ORDER BY id) AS row_num
					FROM entries
				)
				SELECT id, url, username, password, created
				FROM duplicates
				WHERE row_num > 1
				ORDER BY url, username, password, id
			`)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error":   "Failed to identify duplicates",
					"details": err.Error(),
				})
			}

			// Process the duplicate rows
			var duplicateIDs []int
			for rows.Next() {
				var entry Entry
				if err := rows.Scan(&entry.ID, &entry.URL, &entry.User, &entry.Pass, &entry.Created); err != nil {
					rows.Close()
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
						"error":   "Failed to scan row",
						"details": err.Error(),
					})
				}
				duplicates = append(duplicates, entry)
				duplicateIDs = append(duplicateIDs, entry.ID)
			}
			rows.Close()

			// Delete the duplicates if found
			if len(duplicateIDs) > 0 {
				// Convert duplicate IDs to a string for the SQL IN clause
				idStrs := make([]string, len(duplicateIDs))
				for i, id := range duplicateIDs {
					idStrs[i] = strconv.Itoa(id)
				}
				idList := strings.Join(idStrs, ",")

				// Execute the delete query
				result, err := tx.Exec(ctx, "DELETE FROM entries WHERE id IN ("+idList+")")
				if err != nil {
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
						"error":   "Failed to remove duplicates",
						"details": err.Error(),
					})
				}

				// Get number of removed rows
				removed = int(result.RowsAffected())

				// Commit the transaction
				if err := tx.Commit(ctx); err != nil {
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
						"error":   "Failed to commit transaction",
						"details": err.Error(),
					})
				}
			}

			return c.JSON(fiber.Map{
				"duplicatesFound":   len(duplicates),
				"duplicatesRemoved": removed,
				"duplicates":        duplicates,
				"status":            "success",
			})
		} else {
			// Just query and return duplicates without removing them
			rows, err := dbPool.Query(ctx, `
				WITH duplicates AS (
					SELECT id, url, username, password, created,
						ROW_NUMBER() OVER(PARTITION BY url, username, password ORDER BY id) AS row_num
					FROM entries
				)
				SELECT id, url, username, password, created
				FROM duplicates
				WHERE row_num > 1
				ORDER BY url, username, password, id
			`)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error":   "Failed to identify duplicates",
					"details": err.Error(),
				})
			}
			defer rows.Close()

			// Process and return the duplicate rows
			for rows.Next() {
				var entry Entry
				if err := rows.Scan(&entry.ID, &entry.URL, &entry.User, &entry.Pass, &entry.Created); err != nil {
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
						"error":   "Failed to scan row",
						"details": err.Error(),
					})
				}
				duplicates = append(duplicates, entry)
			}

			if err := rows.Err(); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error":   "Error processing results",
					"details": err.Error(),
				})
			}

			return c.JSON(fiber.Map{
				"duplicatesFound": len(duplicates),
				"duplicates":      duplicates,
				"status":          "success",
			})
		}
	})

	// Start the server on port 3000
	log.Println("Starting Go Fiber server on http://localhost:3000")
	log.Fatal(app.Listen(":3000"))
}
