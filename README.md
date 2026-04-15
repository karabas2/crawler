# Multi-Agent Web Crawler & Search Engine (Project 2)

A robust Go-based search engine designed and developed using a multi-agent AI workflow. This system demonstrates advanced concurrency patterns, real-time indexing, and high-performance information retrieval.

## 🚀 Key Features

- **Multi-Agent Design:** Built using a collaborative workflow between Planner, Crawler, Indexing, Search, and Reviewer agents.
- **Concurrent BFS Crawler:** High-speed exploration with URL deduplication and a depth limit of `k`.
- **Real-Time Search:** Search capabilities remain active while the crawler is running, supported by a thread-safe inverted index.
- **Back-Pressure Management:** Intelligent worker pool and queue depth monitoring to handle load.
- **Detailed Metadata:** Search results return the `(relevant_url, origin_url, depth)` triple.

## 📁 Project Structure

```bash
.
├── agents/             # Multi-agent role definitions & prompts
├── crawler/            # Concurrent BFS crawling logic
├── indexer/            # Real-time tokenization & index pipeline
├── search/             # Query processing & relevance scoring
├── storage/            # Thread-safe storage with RWMutex
├── main/               # Application entry point
├── multi_agent_workflow.md  # Detailed agent interaction log
└── product_prd.md      # Project requirements & features
```

## 🛠️ Installation & Usage

### Prerequisites
- Go 1.18 or higher

### Running the System
```bash
go run main/main.go --url https://example.com --depth 2 --workers 10
```

### Searching
You can perform searches even while the crawl is active. Use the provided CLI or API interface to query the system.

## 🤖 Multi-Agent Development Workflow

This project was developed by mapping the architecture into a multi-agent system. Each component was "designed" by a specialized AI agent:

- **Planner Agent:** Established the system backbone and sync strategies.
- **Crawler Agent:** Optimized the worker pool and deduplication logic.
- **Indexing Agent:** Designed the `keyword -> []URL` map and persistence.
- **Search Agent:** Implemented the ranking algorithm and concurrent search locks.
- **Reviewer Agent:** Performed stress-testing analysis and verified thread-safety.

For a full breakdown of agent prompts and decisions, see [multi_agent_workflow.md](./multi_agent_workflow.md).

## 📊 Design Decisions: Searching While Indexing

To ensure the search engine is operational during a crawl:
1. **Shared State:** All data is held in a central `Storage` struct.
2. **Locking Strategy:** We use `sync.RWMutex`. 
   - **Indexer:** Takes a `Lock()` (write) while updating terms.
   - **Searcher:** Takes a `RLock()` (read) while processing queries.
3. **Incremental Updates:** New pages are indexed as soon as they are fetched, making results available immediately.

## 📜 License
MIT
