# AI Prompt - Technical Specification (PRD for AI)

## Objective
Build a high-concurrency modular web crawler and real-time search engine in Go. The system follows a "Live-Indexing" approach where pages are tokenized and added to an inverted index while the crawl is ongoing.

## Architecture Guidelines
- **Language**: Go 1.21+
- **Pattern**: Modular architecture with separate packages (`crawler`, `indexer`, `search`, `ranking`, `storage`).
- **Communication**: Go Channels for worker-to-indexer flow.
- **Thread Safety**: Use `sync.RWMutex` for storage and index access; `sync.Mutex` for counters/visited sets.
- **Frontend**: Single-page HTML/JS/CSS dashboard using native browser fetch API.

## Requirements Checklist for Implementation

### 1. Crawler Engine (`crawler/`)
- [ ] Concurrent BFS implementation using worker pool.
- [ ] Track `(URL, OriginURL, Depth)` for each page.
- [ ] Global `Visited` set to prevent cycles.
- [ ] Rate limiting (hit rate per second) across all workers.
- [ ] Back-pressure signal if the internal queue is full.

### 2. Live Indexer (`indexer/`)
- [ ] Read `PageData` from the `crawler` via channel.
- [ ] Tokenize title and body (lowercase, alphanumeric).
- [ ] Remove common stop words (a, an, the, and, or, but).
- [ ] Update the global inverted index in real-time.

### 3. Search & Ranking (`search/`, `ranking/`)
- [ ] Handle multi-word search queries.
- [ ] Intersect/Union results from the inverted index.
- [ ] Heuristic: `2 * (title_matches) + 1 * (body_keyword_freq)`.
- [ ] Deduplicate results and return TOP N by score.

### 4. Storage & Persistence (`storage/`)
- [ ] In-memory `map[string]*PageData` for storage.
- [ ] In-memory `map[string][]string` for inverted index.
- [ ] Persistent state: Background goroutine saves state to `data/crawl_state.json` periodically.
- [ ] Restore state on startup if the file exists.

### 5. HTTP API & Orchestrator (`main/`)
- [ ] `GET /dashboard`: Serve embedded `dashboard.html`.
- [ ] `POST /crawl`: Start a crawler with config (URL, Workers, Depth).
- [ ] `GET /search`: Return JSON array of ranked results.
- [ ] `GET /status`: Return live metrics (Pages Crawled, Queue Depth, Unique Tokens).
- [ ] `POST /clear`: Wipe in-memory and disk state.

## Core Data Structures

```go
type PageData struct {
    URL       string
    Title     string
    Body      string
    Links     []string
    OriginURL string
    Depth     int
}

type Result struct {
    URL       string
    OriginURL string
    Depth     int
    Score     float64
}
```

## Non-Functional Constraints
- No external heavy databases (use Go maps + RWMutex).
- Clean shutdown using `os/signal` to save state one last time.
- Standard Library approach preferred, minimize dependencies.
