# Multi-Agent Web Crawler & Search Engine

A high-performance search engine and web crawler implemented in Go, designed through a collaborative Multi-Agent AI Workflow. This system demonstrates complex concurrency patterns, real-time incremental indexing, and thread-safe search operations.

## Technical Architecture

This project is built from the ground up to handle large-scale crawling on a single machine using standard library primitives.

- **Concurrency Model:** Uses a Worker Pool pattern with `N` concurrent goroutines.
- **Task Distribution:** Managed via buffered channels to ensure efficient resource utilization.
- **Real-time Pipeline:** The crawler uses a dedicated `PageCh` to stream results to the Indexer as they are discovered, enabling zero-latency search.
- **Thread-Safety:** Implements a central `Storage` hub protected by `sync.RWMutex`, allowing multiple searchers to read while a single indexer writes.

## Key Features

- **Recursive BFS Crawling:** Explores the web starting from a seed URL up to a maximum depth k.
- **Intelligent Deduplication:** Never visits the same URL twice using a mutex-protected lookup map.
- **Dynamic Back-Pressure:** Monitors task queue depth and adjusts indexing speed to prevent memory exhaustion (States: NORMAL, MODERATE, HIGH).
- **Inverted Indexing:** Maps lowercase keywords to URL locations, optimized for retrieval.
- **Triple-Metadata Search:** Returns results with (relevant_url, origin_url, depth) as required by assignment specs.

## Agentic System Design

This project is not just about code generation; it is designed to demonstrate how an autonomous agent hierarchy can solve complex engineering problems. The system was developed through the collaboration of 5 distinct AI agents, each with its own area of expertise:

- **Strategic Planner:** Defined the system architecture (Shared State Architecture) and managed the data flow (Channel Pipeline) between crawler, indexer, and search modules.
- **Concurrency Specialist (Crawler Agent):** Implemented the BFS algorithm using Go-native concurrency primitives (WaitGroups, Tickers) along with the back-pressure mechanism.
- **Data Architect (Indexing Agent):** Automated the indexing process, ensuring Unicode-supported tokenizing and real-time (live-indexing) data commits.
- **UX & Product Agent (Searcher Agent):** Ensured that search results include the required metadata triples and that search remains functional during indexing (RWMutex strategy).
- **QA & Safety Supervisor (Reviewer Agent):** Conducted race condition analysis and dictated system behavior under high load (High Back-Pressure states).

For detailed agent decision logs and design philosophy, please refer to [multi_agent_workflow.md](./multi_agent_workflow.md).

## Repository Structure

```bash
.
├── agents/             # Markdown definitions of AI development roles
├── crawler/            # Concurrent BFS engine (Worker Pool & Fetcher)
├── indexer/            # Tokenization & Real-time Indexing logic
├── search/             # Query parsing & Relevancy Ranking
├── storage/            # Memory-efficient Storage with RWMutex
├── main/               # Application entry point and UI
├── multi_agent_workflow.md  # Detailed documentation of AI collaboration
├── product_prd.md      # Full Product Requirements Document
└── recommendation.md   # Distributed Scaling Strategy
```

## Installation & Execution

### Prerequisites
- Go 1.18 or higher
- External dependency: `golang.org/x/net/html`

### Steps
1. Clone the repository
2. Install dependencies:
   ```bash
   go mod tidy
   ```
3. Start the Agentic Server:
   ```bash
   go run main/main.go
   ```
4. Access the Dashboard:
   Open your browser to [http://localhost:8888](http://localhost:8888)

### API Endpoints
- **POST /crawl**: Start a new task with `{"origin_url": "...", "max_depth": 2, "workers": 5}`.
- **GET /search?q=query**: Retrieve ranked triples.
- **GET /status**: View live metrics (Queue depth, Back-pressure).

## Concurrent Search During Indexing

The system is uniquely designed to stay responsive during high-volume crawls. While the Indexer Agent's code is busy acquiring the write lock to update keywords, the Search Agent utilizes the read lock to fulfill user queries. This ensures that the search engine is never offline while the crawler is running.

## License
MIT
