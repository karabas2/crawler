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

## 🤖 Agentic System Design

Bu proje sadece kod üretmek için değil, **otonom bir agent hiyerarşisinin** karmaşık bir mühendislik problemini nasıl çözdüğünü kanıtlamak için tasarlanmıştır. Sistem, her biri kendi uzmanlık alanına sahip 5 farklı AI ajanının işbirliğiyle geliştirilmiştir:

-  **Strategic Planner:** Sistem mimarisini (Shared State Architecture) kurguladı ve `crawler`, `indexer`, `search` modülleri arasındaki veri akışını (Channel Pipeline) yönetti.
-  **Concurrency Specialist (Crawler Agent):** BFS algoritmasını, Go'nun yerel eşzamanlılık (concurrency) araçlarını (WaitGroup, Ticker) kullanarak "Back-Pressure" mekanizmasıyla birlikte inşa etti.
-  **Data Architect (Indexing Agent):** İndeksleme sürecini otonom hale getirerek Unicode destekli tokenization ve gerçek zamanlı (live-indexing) veri yazımını sağladı.
-  **UX & Product Agent (Searcher Agent):** Arama sonuçlarının ödev gereksinimlerine uygun olarak `triple` (URL, Origin, Depth) formatında dönmesini ve aramanın indeksleme sırasında asla kilitlenmemesini (RWMutex stratejisi) sağladı.
-  **QA & Safety Supervisor (Reviewer Agent):** Kodun "Race Condition" analizlerini yaptı ve sistemin aşırı yük altında (`HIGH` back-pressure) nasıl davranması gerektiğini dikte etti.

> Detaylı ajan karar logları ve tasarım felsefesi için [AGENT_SYSTEM_DESIGN.md](./agents/AGENT_SYSTEM_DESIGN.md) dosyasını inceleyebilirsiniz.

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
3. Start the Agentic Server:
   ```bash
   go run main/main.go
   ```
4. Access the Dashboard:
   Open your browser to [http://localhost:8888](http://localhost:8888)

### API Endpoints
- **POST `/crawl`**: Start a new task with `{"origin_url": "...", "max_depth": 2, "workers": 5}`.
- **GET `/search?q=query`**: Retrieve ranked triples.
- **GET `/status`**: View live metrics (Queue depth, Back-pressure).

## 📊 Concurrent Search During Indexing

The system is uniquely designed to stay responsive during high-volume crawls. While the **Indexer Agent's** code is busy acquiring `Mu.Lock()` to update keywords, the **Search Agent** utilizes `Mu.RLock()` to fulfill user queries. This ensures that the search engine is never offline while the crawler is running.

## ⚖️ License
MIT
