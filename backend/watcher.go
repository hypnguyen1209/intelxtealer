package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// LogWatcher watches the log directory for new files and processes them
type LogWatcher struct {
	watcher        *fsnotify.Watcher
	logDir         string
	processedFiles map[string]bool
	mu             sync.Mutex
	ctx            context.Context
	cancel         context.CancelFunc
}

// NewLogWatcher creates a new log watcher for the specified directory
func NewLogWatcher(logDir string) (*LogWatcher, error) {
	// Create a new fsnotify watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	// Create a context with cancel for proper shutdown
	ctx, cancel := context.WithCancel(context.Background())

	// Create the watcher
	w := &LogWatcher{
		watcher:        watcher,
		logDir:         logDir,
		processedFiles: make(map[string]bool),
		ctx:            ctx,
		cancel:         cancel,
	}

	// Load processed files from database
	if err := w.loadProcessedFiles(); err != nil {
		log.Printf("Warning: Failed to load processed files from database: %v", err)
	}

	return w, nil
}

// Start begins watching the log directory
func (w *LogWatcher) Start() error {
	log.Printf("Starting log watcher for directory: %s", w.logDir)

	// First, make sure the log directory exists
	if _, err := os.Stat(w.logDir); os.IsNotExist(err) {
		if err := os.MkdirAll(w.logDir, 0755); err != nil {
			return err
		}
		log.Printf("Created log directory: %s", w.logDir)
	}

	// Start watching the directory
	if err := w.watcher.Add(w.logDir); err != nil {
		return err
	}

	// Process existing files first
	w.processExistingFiles()

	// Start the watcher goroutine
	go w.watch()

	return nil
}

// processExistingFiles checks any log files that already exist in the directory
// against the database of processed files
func (w *LogWatcher) processExistingFiles() {
	files, err := filepath.Glob(filepath.Join(w.logDir, "*.txt"))
	if err != nil {
		log.Printf("Error reading existing log files: %v", err)
		return
	}

	if len(files) > 0 {
		log.Printf("Found %d existing log files", len(files))

		// Check which files haven't been processed by comparing with our loaded database records
		var unprocessedFiles []string
		w.mu.Lock()
		for _, file := range files {
			fileName := filepath.Base(file)
			if !w.processedFiles[fileName] {
				unprocessedFiles = append(unprocessedFiles, file)
			}
		}
		w.mu.Unlock()

		if len(unprocessedFiles) > 0 {
			log.Printf("Found %d unprocessed files", len(unprocessedFiles))
		} else {
			log.Printf("All existing files have been previously processed")
		}
	}
}

// watch is the main loop that watches for file system events
func (w *LogWatcher) watch() {
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// We're only interested in create events for .txt files
			if event.Op&fsnotify.Create == fsnotify.Create && filepath.Ext(event.Name) == ".txt" {
				w.handleNewFile(event.Name)
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)

		case <-w.ctx.Done():
			return
		}
	}
}

// handleNewFile processes a newly added log file
func (w *LogWatcher) handleNewFile(filePath string) {
	fileName := filepath.Base(filePath)

	w.mu.Lock()
	// Check if we've already processed this file
	if w.processedFiles[fileName] {
		w.mu.Unlock()
		return
	}

	// Mark as processed in our memory map
	w.processedFiles[fileName] = true
	w.mu.Unlock()

	// Wait a brief moment to make sure the file is fully written
	// This helps avoid processing a file that's still being copied
	time.Sleep(2 * time.Second)

	log.Printf("Processing new log file: %s", fileName)

	// Create a context with timeout for processing the file
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Process the file
	count, err := processLogFile(ctx, filePath)
	if err != nil {
		log.Printf("Error processing new log file %s: %v", filePath, err)
		return
	}

	// Record the processed file in the database
	_, err = dbPool.Exec(ctx,
		"INSERT INTO processed_log_files (filename, entries_added) VALUES ($1, $2) "+
			"ON CONFLICT (filename) DO UPDATE SET processed_at = NOW(), entries_added = $2",
		fileName, count)

	if err != nil {
		log.Printf("Failed to record processed file in database: %v", err)
	}

	log.Printf("Successfully processed new log file %s: %d entries added", fileName, count)
}

// ProcessFile manually processes a specific log file
func (w *LogWatcher) ProcessFile(filePath string) (int, error) {
	fileName := filepath.Base(filePath)

	w.mu.Lock()
	// If file was already processed, we could check the database to see how many entries were added previously
	if w.processedFiles[fileName] {
		var entriesAdded int
		err := dbPool.QueryRow(context.Background(),
			"SELECT entries_added FROM processed_log_files WHERE filename = $1",
			fileName).Scan(&entriesAdded)
		if err == nil {
			w.mu.Unlock()
			log.Printf("File %s was already processed with %d entries", fileName, entriesAdded)
			return entriesAdded, nil
		}
	}

	// Mark as processed in memory
	w.processedFiles[fileName] = true
	w.mu.Unlock()

	// Create a context with timeout for processing the file
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Process the file
	log.Printf("Manually processing log file: %s", fileName)
	count, err := processLogFile(ctx, filePath)

	if err != nil {
		return 0, err
	}

	// Record the processed file in the database
	_, dbErr := dbPool.Exec(ctx,
		"INSERT INTO processed_log_files (filename, entries_added) VALUES ($1, $2) "+
			"ON CONFLICT (filename) DO UPDATE SET processed_at = NOW(), entries_added = $2",
		fileName, count)

	if dbErr != nil {
		log.Printf("Warning: Failed to record processed file in database: %v", dbErr)
	}

	return count, nil
}

// Stop stops the watcher
func (w *LogWatcher) Stop() error {
	w.cancel()
	return w.watcher.Close()
}

// loadProcessedFiles loads the list of previously processed files from the database
func (w *LogWatcher) loadProcessedFiles() error {
	// Query the database for previously processed files
	rows, err := dbPool.Query(context.Background(),
		"SELECT filename FROM processed_log_files")
	if err != nil {
		return fmt.Errorf("failed to query processed files: %w", err)
	}
	defer rows.Close()

	// Add each file to the processed files map
	w.mu.Lock()
	defer w.mu.Unlock()

	count := 0
	for rows.Next() {
		var filename string
		if err := rows.Scan(&filename); err != nil {
			return fmt.Errorf("failed to scan filename: %w", err)
		}
		w.processedFiles[filename] = true
		count++
	}

	if count > 0 {
		log.Printf("Loaded %d processed files from database", count)
	}

	return rows.Err()
}
