# Multi-Agent Web Crawler & Search Engine 

A high-performance search engine and web crawler implemented in Go, designed through a collaborative **Multi-Agent AI Workflow**. This system demonstrates complex concurrency patterns, real-time incremental indexing, and thread-safe search operations.

## 🏗️ Technical Architecture

This project is built from the ground up to handle large-scale crawling on a single machine using standard library primitives.

- **Concurrency Model:** Uses a Worker Pool pattern with `N` concurrent goroutines.
- **Task Distribution:** Managed via buffered channels to ensure efficient resource utilization.
- **Real-time Pipeline:** The crawler uses a dedicated `PageCh` to stream results to the Indexer as they are discovered, enabling zero-latency search.
- **Thread-Safety:** Implements a central `Storage` hub protected by `sync.RWMutex`, allowing multiple searchers to read while a single indexer writes.

## 🚀 Key Features

- **Recursive BFS Crawling:** Explores the web starting from a seed URL up to a maximum depth `k`.
- **Intelligent Deduplication:** Never visits the same URL twice using a mutex-protected lookup map.
- **Dynamic Back-Pressure:** Monitors task queue depth and adjusts indexing speed to prevent memory exhaustion (States: NORMAL, MODERATE, HIGH).
- **Inverted Indexing:** Maps lowercase keywords to URL locations, optimized for retrieval.
- **Triple-Metadata Search:** Returns results with `(relevant_url, origin_url, depth)` as required by assignment specs.

## 🤖 Multi-Agent Development Workflow

This system's design was mapped into a multi-agent development lifecycle:

1.  **Planner Agent:** Designed the package structure and inter-service communication.
2.  **Crawler Agent:** Developed the BFS core and worker pool concurrency logic.
3.  **Indexing Agent:** Built the tokenization pipeline and inverted index storage.
4.  **Search Agent:** Implemented the relevance scoring and read-locked query processor.
5.  **Reviewer Agent:** Audited code for race conditions and verified back-pressure integrity.

> See [multi_agent_workflow.md](./multi_agent_workflow.md) for the full agentic reasoning log.

## 📁 Repository Structure

```bash
.
├── agents/             # Markdown definitions of AI development roles
├── crawler/            # Concurrent BFS engine (Worker Pool & Fetcher)
├── indexer/            # Tokenization & Real-time Indexing logic
├── search/             # Query parsing & Relevancy Ranking
├── storage/            # Memory-efficient Storage with RWMutex
├── main/               # CLI entry point
├── multi_agent_workflow.md  # Detailed documentation of AI collaboration
├── product_prd.md      # Full Product Requirements Document
└── recommendation.md   # Distributed Scaling Strategy
```

## 🛠️ Installation & Execution

### Prerequisites
- Go 1.18 or higher
- External dependency: `golang.org/x/net/html`

### Steps
1. Clone the repository
2. Install dependencies:
   ```bash
   go mod tidy
   ```
3. Run a crawl:
   ```bash
   go run main/main.go --url https://example.com --depth 3 --workers 20
   ```

## 📊 Concurrent Search During Indexing

The system is uniquely designed to stay responsive during high-volume crawls. While the **Indexer Agent's** code is busy acquiring `Mu.Lock()` to update keywords, the **Search Agent** utilizes `Mu.RLock()` to fulfill user queries. This ensures that the search engine is never offline while the crawler is running.

## ⚖️ License
MIT
