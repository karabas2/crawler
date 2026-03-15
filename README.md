# Concurrent Web Crawler & Search Engine

A modular concurrent web crawler and search engine built in Go.  
Crawls web pages, indexes them in real-time, and serves search queries via HTTP — all while crawling continues in the background. Includes a real-time web dashboard for creating crawlers, monitoring progress, and searching indexed pages.

---

## System Architecture

```
┌─────────────┐     ┌──────────┐     ┌─────────┐
│   Crawler   │────▶│  Indexer  │────▶│ Storage │
│ (goroutines)│     │(goroutine)│     │(RWMutex)│
└─────────────┘     └──────────┘     └────┬────┘
                                          │
                                    ┌─────▼─────┐
                               ┌────│  Search    │
                               │    │  Engine    │
                               │    └─────┬─────┘
                               │          │
                               │    ┌─────▼─────┐
                               │    │  Ranker    │
                               │    └───────────┘
                               │
                         ┌─────▼─────┐
                         │ HTTP API  │
                         │ :8080     │
                         └───────────┘
```

### Components

| Component | Package | Responsibility |
|-----------|---------|---------------|
| **Crawler** | `crawler/` | Concurrent BFS crawl with worker pool, depth/origin tracking, rate limiting |
| **Indexer** | `indexer/` | Live tokenization and inverted index construction |
| **Search Engine** | `search/` | Query parsing, index lookup, result deduplication |
| **Ranker** | `ranking/` | Relevancy scoring (title match + keyword frequency) |
| **Storage** | `storage/` | Thread-safe in-memory page store, inverted index, and persistence |
| **Main/API** | `main/` | HTTP server, component orchestration, dashboard UI |

---

## Dashboard

The system includes a real-time web dashboard with three tabs:

- ** Create Crawler** — Configure and launch crawlers from the UI with Origin URL, Max Depth, Workers, Hit Rate, Queue Capacity, and Max URLs to Visit.
- ** Search** — Search indexed pages in real-time while crawling continues.
- ** Crawler Status** — Monitor live metrics including indexing progress, queue depth, back-pressure status, and crawler metadata.

A stats bar at the top shows URLs Visited, Words in DB, Active Crawlers, and Total Created with a Clear button.

---

## Concurrency Model

### Goroutines & Channels
- **Worker Pool**: The crawler launches N goroutines (configurable), each reading `CrawlTask` structs from a shared channel.
- **Page Channel**: Crawled pages flow from workers → indexer goroutine via a buffered `chan *PageData`.
- **Indexer Goroutine**: A single goroutine processes pages from the channel, building the inverted index.

### Thread Safety
- **`sync.RWMutex`**: Protects the pages map and inverted index in storage. Read locks allow concurrent search queries; write locks ensure safe page/index updates.
- **`sync.Mutex`**: Protects the crawler's visited-URL set and statistics counters.
- **No data races**: All shared state access is serialized through locks or channels.

### Back-Pressure
- Queue capacity is configurable (default: `workers × 10`).
- Three levels: **NORMAL** (< 50%), **MODERATE** (50-80%), **HIGH** (> 80%).
- Blocking channel send prevents unbounded queue growth.

### Rate Limiting
- **Hit Rate**: Configurable requests-per-second using a shared `time.Ticker`.
- All workers share the ticker, so Setting `hit_rate=2` means 2 total requests/second across all workers.

```
Time ──────────────────────────────────────────────▶

Worker 1:  [fetch]──[parse]──[send to indexer]──[fetch next]──...
Worker 2:  [fetch]──[parse]──[send to indexer]──[fetch next]──...
Worker N:  [fetch]──[parse]──[send to indexer]──[fetch next]──...

Indexer:   .........[tokenize]──[write index]──[tokenize]──...

HTTP API:  ...[search query]──[read index]──[respond]──...
```

---

## Project Structure

```
project/
├── crawler/
│   └── crawler.go         # Concurrent BFS crawler with worker pool & rate limiting
├── indexer/
│   └── indexer.go         # Live tokenizer and inverted index builder
├── search/
│   └── search.go          # Query engine with dedup and ranking
├── ranking/
│   └── ranker.go          # Relevancy scoring heuristic
├── storage/
│   ├── storage.go         # Thread-safe in-memory store + inverted index
│   └── persistence.go     # Save/load crawl state to disk for resume
├── main/
│   ├── main.go            # HTTP API server & orchestrator
│   └── dashboard.html     # Real-time web dashboard UI
├── data/                  # Persisted crawl state (auto-generated)
├── go.mod
├── README.md
├── product_prd.md
└── recommendation.md
```

---

## Getting Started

### Prerequisites
- Go 1.21+

### Build & Run

```bash
# Navigate to the project
cd project

# Download dependencies
go mod tidy

# Run the system (opens dashboard UI, waits for crawler creation via UI)
go run ./main/ --port=8080

# Or auto-start a crawler via CLI flags
go run ./main/ --seed=https://go.dev --depth=2 --workers=5 --port=8080
```

Then open **http://localhost:8080** to access the dashboard.

### Run with Docker

You can easily run the crawler using Docker. A `Dockerfile` and `docker-compose.yml` are provided.

```bash
# Build and start the container in detached mode
docker-compose up -d

# Check the logs
docker-compose logs -f

# Stop the container
docker-compose down
```
The persistence data is automatically mapped to the `./data` folder on your host machine to ensure your crawls are saved across container restarts.

### Command-Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--seed` | *(empty)* | Starting URL (empty = wait for UI) |
| `--depth` | `2` | Maximum crawl depth (BFS levels) |
| `--workers` | `5` | Number of concurrent crawler goroutines |
| `--port` | `8080` | HTTP server port |
| `--data-dir` | `./data` | Directory to persist crawl state |

---

## API Usage

### Create a Crawler (from API)

```bash
curl -X POST http://localhost:8080/crawl \
  -H "Content-Type: application/json" \
  -d '{"origin_url":"https://go.dev","max_depth":2,"workers":5,"hit_rate":2,"max_urls":100}'
```

### Search

```bash
curl "http://localhost:8080/search?q=go+programming"
```

**Response:**
```json
{
  "query": "go programming",
  "count": 3,
  "results": [
    {
      "relevant_url": "https://go.dev/learn/",
      "origin_url": "https://go.dev",
      "depth": 1,
      "score": 15.0
    }
  ]
}
```

### Status

```bash
curl "http://localhost:8080/status"
```

### Stop Active Crawler

```bash
curl -X POST http://localhost:8080/stop
```

### Clear All Data

```bash
curl -X POST http://localhost:8080/clear
```

---

## Persistence (Bonus)

The system automatically saves crawl state to disk every 10 seconds. On restart, it restores previously crawled pages, re-indexes them, and resumes crawling from unvisited child links.

- State file: `data/crawl_state.json`
- Atomic writes via temp file + rename to prevent corruption
- Graceful shutdown performs a final save on SIGINT/SIGTERM

---

## Ranking Formula

```
score = 2 × (title_match_count) + 1 × (body_keyword_frequency)
```

- **Title match count**: Number of query tokens appearing in the page title.
- **Body keyword frequency**: Total occurrences of all query tokens in the page body text.

The weight constants are configurable in `ranking/ranker.go`.

---

## License

MIT
