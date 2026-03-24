// Package main provides the HTTP API server and orchestrates
// the crawler, indexer, and search engine.
package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"webcrawler/crawler"
	"webcrawler/indexer"
	"webcrawler/ranking"
	"webcrawler/search"
	"webcrawler/storage"
)

//go:embed dashboard.html
var dashboardHTML []byte

// App manages the application state including active crawlers.
type App struct {
	store        *storage.Storage
	searchEngine *search.Engine
	ranker       *ranking.Ranker
	pm           *storage.PersistenceManager

	mu             sync.Mutex
	activeCrawler  *crawler.Crawler
	totalCreated   int
	defaultWorkers int
	dataDir        string

	// Crawler metadata for the UI
	crawlerOrigin     string
	crawlerDepth      int
	crawlerHitRate    float64
	crawlerQueueCap   int
	crawlerStartedAt  time.Time
	crawlerLastUpdate time.Time
}

// CrawlRequest represents a request to create a new crawler via API.
type CrawlRequest struct {
	OriginURL     string  `json:"origin_url"`
	MaxDepth      int     `json:"max_depth"`
	Workers       int     `json:"workers"`
	HitRate       float64 `json:"hit_rate"`
	QueueCapacity int     `json:"queue_capacity"`
	MaxURLs       int     `json:"max_urls"`
}

func main() {
	seed := flag.String("seed", "", "Seed URL (empty = wait for UI)")
	depth := flag.Int("depth", 2, "Maximum crawl depth")
	workers := flag.Int("workers", 5, "Number of concurrent crawler workers")
	port := flag.Int("port", 8080, "HTTP server port")
	dataDir := flag.String("data-dir", "./data", "Directory to persist crawl state")
	flag.Parse()

	log.Println("=== Concurrent Web Crawler & Search Engine ===")
	log.Printf("Port     : %d", *port)
	log.Printf("Data Dir : %s", *dataDir)

	// ---- Initialize Components ----
	store := storage.New()
	ranker := ranking.New()
	searchEngine := search.New(store, ranker)

	app := &App{
		store:          store,
		searchEngine:   searchEngine,
		ranker:         ranker,
		dataDir:        *dataDir,
		defaultWorkers: *workers,
	}

	// Persistence manager
	app.pm = storage.NewPersistenceManager(store, *dataDir, 10*time.Second)

	// Restore previous state
	visitedURLs, err := app.pm.Restore()
	if err != nil {
		log.Printf("[Persistence] Warning: could not restore state: %v", err)
	}
	if len(visitedURLs) > 0 {
		idx := indexer.New(store)
		pages := store.GetAllPages()
		for _, page := range pages {
			idx.IndexPage(page)
		}
		log.Printf("[Persistence] Re-indexed %d restored pages.", len(pages))
	}

	app.pm.Start()

	// Auto-start if seed URL provided via CLI
	if *seed != "" {
		app.startCrawler(CrawlRequest{
			OriginURL: *seed,
			MaxDepth:  *depth,
			Workers:   *workers,
		})
	}

	// ---- HTTP API ----
	mux := http.NewServeMux()

	// GET / — Dashboard UI
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(dashboardHTML)
	})

	// GET /search?q=<query>
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		if query == "" {
			query = r.URL.Query().Get("q")
		}
		if query == "" {
			http.Error(w, `{"error":"missing query parameter 'query' or 'q'"}`, http.StatusBadRequest)
			return
		}
		results := searchEngine.Search(query)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"query":   query,
			"count":   len(results),
			"results": results,
		})
	})

	// GET /status — Enhanced status with crawler metadata
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		stats := store.GetStats()

		app.mu.Lock()
		active := 0
		if app.activeCrawler != nil && app.activeCrawler.IsRunning() {
			active = 1
		}
		resp := map[string]interface{}{
			"pages_crawled":    stats.PagesCrawled,
			"pages_indexed":    stats.PagesIndexed,
			"unique_tokens":    stats.UniqueTokens,
			"urls_queued":      stats.URLsQueued,
			"urls_processed":   stats.URLsProcessed,
			"urls_failed":      stats.URLsFailed,
			"queue_depth":      stats.QueueDepth,
			"back_pressure":    stats.BackPressure,
			"active_crawlers":  active,
			"total_created":    app.totalCreated,
			"crawler_origin":   app.crawlerOrigin,
			"crawler_depth":    app.crawlerDepth,
			"crawler_hit_rate": app.crawlerHitRate,
			"queue_capacity":   app.crawlerQueueCap,
			"crawler_started":  "",
			"crawler_last_update": "",
		}
		if !app.crawlerStartedAt.IsZero() {
			resp["crawler_started"] = app.crawlerStartedAt.Format("1/2/2006, 3:04:05 PM")
		}
		if !app.crawlerLastUpdate.IsZero() {
			resp["crawler_last_update"] = app.crawlerLastUpdate.Format("1/2/2006, 3:04:05 PM")
		}
		app.crawlerLastUpdate = time.Now()
		app.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		json.NewEncoder(w).Encode(resp)
	})

	// POST /crawl — Create and start a new crawler
	mux.HandleFunc("/crawl", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		var req CrawlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}
		if req.OriginURL == "" {
			http.Error(w, `{"error":"origin_url is required"}`, http.StatusBadRequest)
			return
		}
		app.startCrawler(req)

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "started",
			"message": fmt.Sprintf("Crawler started for %s", req.OriginURL),
		})
	})

	// POST /stop — Stop the active crawler
	mux.HandleFunc("/stop", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		app.mu.Lock()
		if app.activeCrawler != nil && app.activeCrawler.IsRunning() {
			app.activeCrawler.Stop()
		}
		app.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "stopped"})
	})

	// POST /clear — Stop crawler and clear all data
	mux.HandleFunc("/clear", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		app.mu.Lock()
		if app.activeCrawler != nil && app.activeCrawler.IsRunning() {
			app.activeCrawler.Stop()
		}
		app.activeCrawler = nil
		app.crawlerOrigin = ""
		app.crawlerDepth = 0
		app.crawlerHitRate = 0
		app.crawlerQueueCap = 0
		app.crawlerStartedAt = time.Time{}
		app.crawlerLastUpdate = time.Time{}
		app.mu.Unlock()

		store.Clear()

		// Remove persistence file
		os.Remove(filepath.Join(app.dataDir, "crawl_state.json"))

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "cleared"})
	})

	// ---- Start Server ----
	addr := fmt.Sprintf(":%d", *port)
	server := &http.Server{Addr: addr, Handler: mux}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		log.Printf("Received signal %s. Shutting down...", sig)
		app.mu.Lock()
		if app.activeCrawler != nil {
			app.activeCrawler.Stop()
		}
		app.mu.Unlock()
		app.pm.Stop()
		server.Close()
	}()

	log.Printf("[Server] Listening on http://localhost%s", addr)
	log.Println("[Server] Endpoints:")
	log.Println("  GET  /                  — Dashboard UI")
	log.Println("  GET  /search?q=<query>  — Search indexed pages")
	log.Println("  GET  /status            — Crawl statistics")
	log.Println("  POST /crawl             — Start a new crawler")
	log.Println("  POST /stop              — Stop active crawler")
	log.Println("  POST /clear             — Clear all data")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
	log.Println("Server stopped.")
}

// startCrawler creates and launches a new crawler with the given config.
func (app *App) startCrawler(req CrawlRequest) {
	app.mu.Lock()
	defer app.mu.Unlock()

	// Stop existing crawler if running
	if app.activeCrawler != nil && app.activeCrawler.IsRunning() {
		app.activeCrawler.Stop()
		// Brief pause so goroutines clean up
		time.Sleep(200 * time.Millisecond)
	}

	if req.MaxDepth <= 0 {
		req.MaxDepth = 2
	}
	w := req.Workers
	if w <= 0 {
		w = app.defaultWorkers
	}

	c := crawler.New(app.store, crawler.Config{
		MaxDepth:      req.MaxDepth,
		MaxWorkers:    w,
		MaxURLs:       req.MaxURLs,
		HitRate:       req.HitRate,
		QueueCapacity: req.QueueCapacity,
	})

	idx := indexer.New(app.store)
	go idx.Start(c.PageCh)

	c.Start(req.OriginURL)

	app.activeCrawler = c
	app.totalCreated++
	app.crawlerOrigin = req.OriginURL
	app.crawlerDepth = req.MaxDepth
	app.crawlerHitRate = req.HitRate
	
	qCap := req.QueueCapacity
	if qCap <= 0 {
		qCap = 10000
	}
	app.crawlerQueueCap = qCap

	app.crawlerStartedAt = time.Now()
	app.crawlerLastUpdate = time.Now()

	log.Printf("[App] Crawler #%d started: origin=%s depth=%d workers=%d hitRate=%.1f maxURLs=%d",
		app.totalCreated, req.OriginURL, req.MaxDepth, w, req.HitRate, req.MaxURLs)
}
