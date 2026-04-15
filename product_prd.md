# Product Requirements Document (PRD) - Multi-Agent Web Crawler & Search

## 1. Project Overview
This project is a high-performance, single-machine web crawler and search engine implemented in Go. It allows users to crawl the web starting from a seed URL up to a depth of `k` hops, index the content in real-time, and perform concurrent searches on the indexed data. 

The development of this system follows a **Multi-Agent AI Workflow**, where distinct AI agents collaborate to design, implement, and review the technical architecture.

## 2. Target Audience
- Academic evaluators for Project 2.
- Developers looking for a concurrent search engine reference implementation.

## 3. Core Features

### 3.1 Concurrent BFS Crawling
- **Recursive Depth (k):** Limits the crawl to `k` steps from the seed URL.
- **URL Deduplication:** Uses a thread-safe map to ensure each unique URL is only crawled once.
- **Worker Pool:** Utilizes a pool of concurrent goroutines to fetch pages.
- **Back-Pressure:** Monitors the task queue and adjusts processing speed (NORMAL, MODERATE, HIGH states) to manage system load.

### 3.2 Real-Time Indexing
- **Inverted Index:** Maps keywords found in the title and body ofpages to their corresponding URLs.
- **Tokenization:** Normalizes text (lowercase, whitespace splitting) for efficient matching.
- **Pipeline:** Crawler sends pages to a dedicated Indexer via Go channels.

### 3.3 Information Retrieval (Search)
- **Keyword Search:** Finds relevant pages based on query tokens.
- **Relevance Scoring:** Ranks results based on occurrences of query terms.
- **Triple Output:** Every search result includes `(relevant_url, origin_url, depth)`.
- **Concurrent Access:** Searching is fully functional while the indexer is actively writing to the index.

### 3.4 Persistence & Stats
- **Thread-Safe Storage:** Uses `sync.RWMutex` to manage shared state across Crawler, Indexer, and Search components.
- **Metric Tracking:** Tracks URLs processed, failed, and currently queued.

## 4. Technical Constraints
- **Language:** Go 1.18+ (Standard Library preferred).
- **Concurrency:** Must use native Go concurrency primitives (Channels, Mutexes, Goroutines).
- **Architecture:** Single-machine execution.

## 5. Multi-Agent Development Workflow
The system design was achieved through the interaction of:
1. **Planner Agent:** Architectural strategy and constraints.
2. **Crawler Agent:** Concurrency and BFS implementation.
3. **Indexing Agent:** Inverted index and tokenization design.
4. **Search Agent:** Query processing and ranking.
5. **Reviewer Agent:** Race condition detection and design critique.

## 6. Success Criteria
- [x] Complete crawl up to depth `k` without infinite loops.
- [x] Search remains responsive during heavy crawling.
- [x] Search results include the required metadata triples.
- [x] Documented agentic collaboration process.
