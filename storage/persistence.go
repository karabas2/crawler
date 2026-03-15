// Package storage provides persistence support for saving and loading
// crawled pages to/from disk, enabling crawl resumption after interruption.
package storage

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// PersistenceManager handles saving and loading crawl state to/from disk.
type PersistenceManager struct {
	store    *Storage
	dataDir  string
	mu       sync.Mutex
	stopCh   chan struct{}
	interval time.Duration
}

// persistedState is the on-disk format for the crawl state.
type persistedState struct {
	Pages       []*PageData       `json:"pages"`
	VisitedURLs []string          `json:"visited_urls"`
	SavedAt     string            `json:"saved_at"`
	Stats       Stats             `json:"stats"`
}

// NewPersistenceManager creates a manager that auto-saves state periodically.
func NewPersistenceManager(store *Storage, dataDir string, interval time.Duration) *PersistenceManager {
	if interval <= 0 {
		interval = 10 * time.Second
	}
	return &PersistenceManager{
		store:    store,
		dataDir:  dataDir,
		stopCh:   make(chan struct{}),
		interval: interval,
	}
}

// Start begins periodic auto-saving in a background goroutine.
func (pm *PersistenceManager) Start() {
	// Ensure data directory exists
	if err := os.MkdirAll(pm.dataDir, 0755); err != nil {
		log.Printf("[Persistence] Warning: could not create data dir %s: %v", pm.dataDir, err)
		return
	}

	go func() {
		ticker := time.NewTicker(pm.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := pm.Save(); err != nil {
					log.Printf("[Persistence] Auto-save error: %v", err)
				}
			case <-pm.stopCh:
				return
			}
		}
	}()

	log.Printf("[Persistence] Auto-save started (every %s) → %s", pm.interval, pm.dataDir)
}

// Stop halts the auto-save loop and performs a final save.
func (pm *PersistenceManager) Stop() {
	close(pm.stopCh)
	if err := pm.Save(); err != nil {
		log.Printf("[Persistence] Final save error: %v", err)
	} else {
		log.Println("[Persistence] Final state saved successfully.")
	}
}

// Save writes the current crawl state to disk atomically.
func (pm *PersistenceManager) Save() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pages := pm.store.GetAllPages()
	stats := pm.store.GetStats()

	state := persistedState{
		Pages:   pages,
		Stats:   stats,
		SavedAt: time.Now().Format(time.RFC3339),
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}

	// Atomic write: write to temp file, then rename
	filePath := filepath.Join(pm.dataDir, "crawl_state.json")
	tmpPath := filePath + ".tmp"

	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}

	if err := os.Rename(tmpPath, filePath); err != nil {
		return fmt.Errorf("renaming temp file: %w", err)
	}

	log.Printf("[Persistence] Saved %d pages to %s", len(pages), filePath)
	return nil
}

// Load reads a previously saved crawl state from disk.
// Returns the loaded pages and visited URLs, or nil if no state file exists.
func (pm *PersistenceManager) Load() ([]*PageData, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	filePath := filepath.Join(pm.dataDir, "crawl_state.json")

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No previous state — fresh start
		}
		return nil, fmt.Errorf("reading state file: %w", err)
	}

	var state persistedState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("unmarshaling state: %w", err)
	}

	log.Printf("[Persistence] Loaded %d pages from %s (saved at %s)",
		len(state.Pages), filePath, state.SavedAt)

	return state.Pages, nil
}

// Restore loads saved state into storage and returns the visited URLs
// so the crawler can skip already-crawled pages.
func (pm *PersistenceManager) Restore() ([]string, error) {
	pages, err := pm.Load()
	if err != nil {
		return nil, err
	}
	if pages == nil {
		log.Println("[Persistence] No previous state found. Starting fresh crawl.")
		return nil, nil
	}

	var visitedURLs []string
	for _, page := range pages {
		pm.store.StorePage(page)
		visitedURLs = append(visitedURLs, page.URL)
	}

	log.Printf("[Persistence] Restored %d pages into storage.", len(pages))
	return visitedURLs, nil
}
