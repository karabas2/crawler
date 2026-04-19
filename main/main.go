package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"webcrawler/crawler"
	"webcrawler/indexer"
	"webcrawler/ranking"
	"webcrawler/search"
	"webcrawler/storage"
)

//go:embed dashboard.html
var dashboardHTML []byte

type App struct {
	store  *storage.Storage
	engine *search.Engine
	idx    *indexer.Indexer
	
	mu            sync.Mutex
	activeCrawler *crawler.Crawler
}

func main() {
	store := storage.New()
	ranker := ranking.New()
	engine := search.New(store, ranker)
	idx := indexer.New(store)

	app := &App{
		store:  store,
		engine: engine,
		idx:    idx,
	}

	// Handlers
	http.HandleFunc("/", app.serveDashboard)
	http.HandleFunc("/crawl", app.handleCrawl)
	http.HandleFunc("/search", app.handleSearch)
	http.HandleFunc("/status", app.handleStatus)
	http.HandleFunc("/clear", app.handleClear)

	// Shutdown handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Shutting down...")
		os.Exit(0)
	}()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8888"
	}

	log.Printf("Agentic Server running at http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func (a *App) serveDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write(dashboardHTML)
}

func (a *App) handleCrawl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		OriginURL     string  `json:"origin_url"`
		MaxDepth      int     `json:"max_depth"`
		Workers       int     `json:"workers"`
		HitRate       float64 `json:"hit_rate"`
		QueueCapacity int     `json:"queue_capacity"`
		MaxURLs       int     `json:"max_urls"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.activeCrawler = crawler.New(a.store, req.MaxDepth, req.Workers)
	
	// Pipeline
	go a.idx.Start(a.activeCrawler.PageCh)
	go a.activeCrawler.Start(req.OriginURL)

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "started"})
}

func (a *App) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	results := a.engine.Search(q)
	
	response := map[string]interface{}{
		"results": results,
		"count":   len(results),
		"query":   q,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (a *App) handleStatus(w http.ResponseWriter, r *http.Request) {
	stats := a.store.GetStats()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (a *App) handleClear(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	a.store = storage.New()
	a.engine = search.New(a.store, ranking.New())
	a.idx = indexer.New(a.store)
	a.mu.Unlock()
	fmt.Fprintln(w, "System state cleared")
}
