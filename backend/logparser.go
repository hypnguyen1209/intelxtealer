package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

// ParseLogDirectory parses all log files in the specified directory
// and adds their contents to the database, skipping already processed files
func ParseLogDirectory(logDir string) error {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Get all files in the log directory
	files, err := filepath.Glob(filepath.Join(logDir, "*.txt"))
	if err != nil {
		return fmt.Errorf("failed to read log directory: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no log files found in directory: %s", logDir)
	}

	log.Printf("Found %d log files to check", len(files))

	// Get list of already processed files from the database
	processedFiles := make(map[string]bool)
	rows, err := dbPool.Query(ctx, "SELECT filename FROM processed_log_files")
	if err != nil {
		log.Printf("Warning: Failed to query processed files: %v", err)
	} else {
		defer rows.Close()

		for rows.Next() {
			var filename string
			if err := rows.Scan(&filename); err != nil {
				log.Printf("Warning: Failed to scan filename: %v", err)
				continue
			}
			processedFiles[filename] = true
		}

		if err := rows.Err(); err != nil {
			log.Printf("Warning: Error iterating processed files: %v", err)
		}

		log.Printf("Found %d previously processed files in database", len(processedFiles))
	}

	// Filter out already processed files
	var filesToProcess []string
	for _, file := range files {
		fileName := filepath.Base(file)
		if !processedFiles[fileName] {
			filesToProcess = append(filesToProcess, file)
		}
	}

	if len(filesToProcess) == 0 {
		log.Printf("All files have been processed already")
		return nil
	}

	log.Printf("Processing %d new log files", len(filesToProcess))
	totalEntries := 0

	// Process each new file
	for _, file := range filesToProcess {
		fileName := filepath.Base(file)
		count, err := processLogFile(ctx, file)
		if err != nil {
			log.Printf("Error processing file %s: %v", fileName, err)
			continue
		}

		// Record the processed file in the database
		_, dbErr := dbPool.Exec(ctx,
			"INSERT INTO processed_log_files (filename, entries_added) VALUES ($1, $2) "+
				"ON CONFLICT (filename) DO UPDATE SET processed_at = NOW(), entries_added = $2",
			fileName, count)

		if dbErr != nil {
			log.Printf("Warning: Failed to record processed file in database: %v", dbErr)
		}

		totalEntries += count
		log.Printf("Processed %s: %d entries added", fileName, count)
	}

	log.Printf("Total entries added to database: %d", totalEntries)
	return nil
}

// processLogFile reads a single log file and processes each line
func processLogFile(ctx context.Context, filePath string) (int, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	log.Printf("Processing file: %s", filepath.Base(filePath))

	// Create a prepared statement for better performance
	const insertSQL = "INSERT INTO entries (url, username, password, created) VALUES ($1, $2, $3, $4)"

	// Acquire a connection from the pool for this operation
	conn, err := dbPool.Acquire(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to acquire database connection: %w", err)
	}
	defer conn.Release()
	// Prepare the statement - the name "insert_entry" is used in batch.Queue later
	_, err = conn.Conn().Prepare(ctx, "insert_entry", insertSQL)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare statement: %w", err)
	}

	// Create a scanner to read the file line by line
	scanner := bufio.NewScanner(file)

	// Buffer size optimization for large files
	const maxCapacity = 512 * 1024 // 512KB
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	// Use a custom split function that can handle problematic bytes
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		// Skip null bytes and try to find the next newline
		start := 0
		for start < len(data) && data[start] == 0 {
			start++
		}

		if start == len(data) {
			if atEOF {
				return 0, nil, nil
			}
			// Request more data
			return 0, nil, nil
		}

		// Look for newline after skipping null bytes
		if i := bytes.IndexByte(data[start:], '\n'); i >= 0 {
			// We have a full line
			return start + i + 1, dropInvalidUtf8(data[start : start+i]), nil
		}

		// If we're at EOF, return the remaining data
		if atEOF {
			return len(data), dropInvalidUtf8(data[start:]), nil
		}

		// Request more data
		return 0, nil, nil
	})

	// Create a batch
	batch := &pgx.Batch{}

	entryCount := 0
	batchSize := 0
	maxBatchSize := 1000 // Process in batches of 1000 entries
	currentTime := time.Now().Format("2006-01-02")

	// For each line in the file
	for scanner.Scan() {
		// Get line and sanitize it to handle invalid UTF-8 characters
		line := sanitizeString(scanner.Text())

		// Skip empty lines
		if len(strings.TrimSpace(line)) == 0 {
			continue
		}

		// Parse the line
		parts := splitLogLine(line)
		if len(parts) < 3 {
			// Skip invalid lines without logging to avoid spam
			continue
		}

		// Sanitize each part to ensure no invalid UTF-8 characters
		url := sanitizeString(parts[0])
		username := sanitizeString(parts[1])
		password := sanitizeString(parts[2])

		// Queue the prepared statement in the batch
		batch.Queue("insert_entry", url, username, password, currentTime)
		batchSize++
		entryCount++

		// Execute batch when it reaches the maximum size
		if batchSize >= maxBatchSize {
			// Send batch and get results
			br := conn.Conn().SendBatch(ctx, batch)
			_, err := br.Exec()
			if err != nil {
				_ = br.Close()
				return entryCount - batchSize, fmt.Errorf("batch execution failed: %w", err)
			}
			_ = br.Close() // Close the batch results

			if entryCount%10000 == 0 {
				log.Printf("Processed %d entries so far", entryCount)
			}

			// Create a new batch
			batch = &pgx.Batch{}
			batchSize = 0
		}
	}

	// Execute remaining entries in the final batch
	if batchSize > 0 {
		br := conn.Conn().SendBatch(ctx, batch)
		_, err := br.Exec()
		if err != nil {
			_ = br.Close()
			return entryCount - batchSize, fmt.Errorf("final batch execution failed: %w", err)
		}
		_ = br.Close()
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		return entryCount, fmt.Errorf("error reading file: %w", err)
	}

	return entryCount, nil
}

// splitLogLine splits a log line into its components
// Handles format like "https://auralia.cloud/login:Bengalar:Robert2024!"
func splitLogLine(line string) []string { // Handle Android scheme URLs (e.g., android://base64@com.app/:username:password)
	reAndroid := regexp.MustCompile(`^(android://[^@]+@[^/:]+(?:/[^:]*)?):([^:]+):(.+)$`)

	matchesAndroid := reAndroid.FindStringSubmatch(line)
	if len(matchesAndroid) >= 4 {
		return []string{
			matchesAndroid[1], // Android URL (group 1)
			matchesAndroid[2], // Username (group 2)
			matchesAndroid[3], // Password (group 3)
		}
	}

	// Special case for URLs with port numbers (e.g., https://example.com:8080:username:password)
	reWithPort := regexp.MustCompile(`^(https?://[^/:]+(?::\d+)?(?:/[^:]*)?):([^:]+):(.+)$`)

	matches := reWithPort.FindStringSubmatch(line)
	if len(matches) >= 4 {
		//log.Printf("Debug - URL: %s, Username: %s, Password: %s", matches[1], matches[2], matches[3])
		// We have a match with the expected format
		return []string{
			matches[1], // URL (group 1)
			matches[2], // Username (group 2)
			matches[3], // Password (group 3)
		}
	}

	// Try a more flexible pattern that handles cases where the URL may contain colons
	// This pattern assumes the URL is everything between http(s):// and the last slash before username
	reAlternate := regexp.MustCompile(`^(https?://[^/]+(?:/[^:]*)?):([^:]+):(.+)$`)

	matches = reAlternate.FindStringSubmatch(line)
	if len(matches) >= 4 {
		return []string{
			matches[1], // URL with path (group 1)
			matches[2], // Username (group 2)
			matches[3], // Password (group 3)
		}
	}
	// If regex attempts fail, fall back to simpler methods

	// Try to detect http/https/android URLs with colons and split appropriately
	if strings.HasPrefix(strings.ToLower(line), "http") || strings.HasPrefix(strings.ToLower(line), "android") {
		// Find position of "://" which indicates protocol separator
		protoIdx := strings.Index(line, "://")
		if protoIdx > 0 {
			// Find the first colon after the protocol separator
			protocol := line[:protoIdx+3] // includes "://"
			remainder := line[protoIdx+3:]

			// For android URLs, handle the @ symbol
			if strings.HasPrefix(strings.ToLower(line), "android") {
				atIdx := strings.Index(remainder, "@")
				if atIdx > 0 {
					baseToken := remainder[:atIdx+1] // includes "@"
					remainder = remainder[atIdx+1:]
					protocol = protocol + baseToken
				}
			}

			// Find the next two colons which should separate URL from username and password
			parts := strings.SplitN(remainder, ":", 3)

			if len(parts) >= 3 {
				url := protocol + parts[0]
				username := parts[1]
				password := parts[2]

				return []string{url, username, password}
			}
		}
	}

	// Fallback to simple colon-separated splitting
	parts := strings.Split(line, ":")
	if len(parts) >= 3 {
		// Simple case with exactly 3 parts
		return []string{
			strings.TrimSpace(parts[0]),  // URL
			strings.TrimSpace(parts[1]),  // Username
			strings.Join(parts[2:], ":"), // Password (may contain colons)
		}
	}

	// Try simple space-based splitting as a last resort
	parts = strings.Fields(line)
	if len(parts) >= 3 {
		return []string{
			parts[0],                     // URL
			parts[1],                     // Username
			strings.Join(parts[2:], " "), // Password (may contain spaces)
		}
	}

	// Return whatever we have
	return parts
}

// sanitizeString removes null bytes and ensures valid UTF-8 characters
func sanitizeString(input string) string {
	// Check for null bytes
	if strings.Contains(input, "\x00") {
		// Log that we found null bytes (only once per execution to avoid spam)
		log.Printf("Found and removed null bytes in input")
	}

	// Remove null bytes which cause PostgreSQL UTF-8 encoding errors
	sanitized := strings.ReplaceAll(input, "\x00", "")

	// Handle other invalid UTF-8 sequences by replacing them
	// This creates a new string with invalid UTF-8 sequences replaced
	if !utf8.ValidString(sanitized) {
		// Replace invalid UTF-8 sequences with the Unicode replacement character
		var result []rune
		for _, r := range sanitized {
			if r == utf8.RuneError {
				// Skip invalid runes
				continue
			}
			result = append(result, r)
		}
		sanitized = string(result)
	}

	return sanitized
}

// dropInvalidUtf8 removes invalid UTF-8 sequences from a byte slice
func dropInvalidUtf8(data []byte) []byte {
	if utf8.Valid(data) {
		return data
	}

	result := make([]byte, 0, len(data))
	for len(data) > 0 {
		r, size := utf8.DecodeRune(data)
		if r != utf8.RuneError {
			result = append(result, data[:size]...)
		}
		data = data[size:]
	}
	return result
}
