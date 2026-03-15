# Product Requirements Document (PRD)

## Concurrent Web Crawler & Search Engine

---

## 1. Overview

This document defines the product requirements for a minimal concurrent web crawler and search engine system. The system crawls web pages starting from a seed URL, indexes their content in real-time, and exposes a search API that can be queried while crawling is still in progress.

---

## 2. Goals

- **Concurrent Crawling**: Crawl multiple web pages simultaneously using a configurable worker pool.
- **Live Indexing**: Index pages as they are crawled, making them immediately searchable.
- **Search API**: Provide an HTTP endpoint that returns ranked search results as (relevant_url, origin_url, depth) triples.
- **Modular Architecture**: Separate concerns into distinct, testable components.
- **Thread Safety**: Ensure all shared data structures are safe for concurrent access.

---

## 3. User Stories

| ID | Story | Priority |
|----|-------|----------|
| US-1 | As a user, I can start the crawler with a seed URL and it begins fetching pages concurrently. | Must-have |
| US-2 | As a user, I can search for keywords via the HTTP API while the crawler is still running. | Must-have |
| US-3 | As a user, I receive search results ranked by relevancy with URL, origin, and depth info. | Must-have |
| US-4 | As a user, I can check the crawl status (pages crawled, pages indexed) via the API. | Should-have |
| US-5 | As a user, I can configure crawl depth, worker count, and server port via CLI flags. | Should-have |
| US-6 | As a user, I can gracefully stop the system with Ctrl+C. | Should-have |

---

## 4. Functional Requirements

### 4.1 Crawler
- FR-1: Accept a configurable seed URL as the starting point.
- FR-2: Perform BFS traversal up to a configurable maximum depth.
- FR-3: Track the origin URL (parent) and depth for every discovered page.
- FR-4: Use a configurable number of worker goroutines for concurrent fetching.
- FR-5: Avoid re-crawling already-visited URLs.
- FR-6: Parse HTML to extract page title, body text, and outbound links.
- FR-7: Resolve relative URLs to absolute URLs.
- FR-8: Filter out non-HTTP/HTTPS links, fragments, and javascript: URLs.

### 4.2 Indexer
- FR-9: Receive crawled pages in real-time via a channel.
- FR-10: Tokenize page title and body text into lowercase terms.
- FR-11: Remove common English stop words.
- FR-12: Build an inverted index mapping tokens to pages.
- FR-13: Deduplicate tokens per page before indexing.

### 4.3 Search Engine
- FR-14: Accept a query string and tokenize it.
- FR-15: Look up each query token in the inverted index.
- FR-16: Merge and deduplicate results across tokens.
- FR-17: Score results using the ranking heuristic.
- FR-18: Return results sorted by score descending.
- FR-19: Return results as (relevant_url, origin_url, depth, score) tuples.

### 4.4 Ranking
- FR-20: Score = 2 × (title match count) + 1 × (body keyword frequency).
- FR-21: Title match = number of query tokens appearing in the page title.
- FR-22: Body frequency = total occurrences of query tokens in page body.

### 4.5 HTTP API
- FR-23: `GET /search?q=<query>` returns JSON search results.
- FR-24: `GET /status` returns crawl statistics as JSON.
- FR-25: Return appropriate HTTP error codes (400 for missing query).

---

## 5. Non-Functional Requirements

| ID | Requirement | Target |
|----|-------------|--------|
| NFR-1 | **Concurrency Safety** | Zero data races under concurrent crawl + search. |
| NFR-2 | **Response Time** | Search queries respond in < 100ms for indexes up to 10k pages. |
| NFR-3 | **Memory** | Limit page body to 1MB to prevent memory exhaustion. |
| NFR-4 | **HTTP Timeout** | 10-second timeout per crawl request. |
| NFR-5 | **Graceful Shutdown** | Stop crawling and shut down the server on SIGINT/SIGTERM. |
| NFR-6 | **Modularity** | Each component in its own package with clear interfaces. |

---

## 6. Architecture

```
Seed URL → Crawler (N workers) → [Channel] → Indexer → Storage (Inverted Index)
                                                              ↕
                                              HTTP API → Search Engine → Ranker
```

### Component Interactions
1. **Crawler → Indexer**: Pages flow via a buffered Go channel.
2. **Indexer → Storage**: Tokens are written to the inverted index (write-locked).
3. **Search → Storage**: Queries read the inverted index (read-locked).
4. **Search → Ranker**: Each candidate page is scored by the ranker.

---

## 7. Data Models

### PageData
```go
type PageData struct {
    URL       string
    Title     string
    Body      string
    Links     []string
    OriginURL string
    Depth     int
}
```

### SearchResult
```go
type Result struct {
    RelevantURL string
    OriginURL   string
    Depth       int
    Score       float64
}
```

---

## 8. Acceptance Criteria

- [ ] System starts crawling from the seed URL and discovers linked pages.
- [ ] Search API returns results while crawling is actively running.
- [ ] Results include relevant_url, origin_url, and depth for each match.
- [ ] Results are ranked by score (higher = more relevant).
- [ ] No data races when crawling and searching concurrently.
- [ ] System shuts down gracefully on Ctrl+C.
