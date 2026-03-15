# Production Roadmap & Recommendations

## Scaling the Web Crawler & Search Engine to Production

This document outlines a roadmap for evolving this minimal concurrent web crawler into a production-grade, distributed system.

---

## Phase 1: Enhanced Single-Node (1–2 weeks)

### Persistent Storage
- **Current**: In-memory maps lost on restart.
- **Recommendation**: Add a persistence layer using BoltDB or BadgerDB (embedded key-value stores).
- **Benefit**: Crawl state survives restarts; enables incremental crawling.

### Robots.txt Compliance
- **Current**: No robots.txt checking.
- **Recommendation**: Fetch and parse `robots.txt` for each domain before crawling. Respect `Crawl-delay` directives.
- **Benefit**: Ethical crawling; avoids getting blocked.

### Rate Limiting
- **Current**: No per-domain throttling.
- **Recommendation**: Implement per-domain rate limiters using `golang.org/x/time/rate`. Limit to 1–2 requests/second per domain.
- **Benefit**: Prevents overwhelming target servers; avoids IP bans.

### Improved Ranking
- **Current**: Simple title match + keyword frequency.
- **Recommendation**: Add TF-IDF scoring, link-based authority signals, and URL path relevancy.
- **Benefit**: Significantly better search quality.

---

## Phase 2: Distributed Crawling (2–4 weeks)

### Message Queue Architecture
- **Current**: In-process channel between crawler and indexer.
- **Recommendation**: Replace with Apache Kafka or RabbitMQ.
  ```
  Crawler Workers → [Kafka: crawl-results] → Indexer Workers
                   → [Kafka: crawl-tasks]  → Crawler Workers
  ```
- **Benefit**: Decouples components; enables independent scaling; provides durability and replay.

### Distributed Crawler Fleet
- **Current**: Single-process, N goroutines.
- **Recommendation**: Deploy multiple crawler instances, each pulling tasks from a shared queue. Use consistent hashing to assign URL domains to specific crawlers.
- **Benefit**: Horizontal scaling; fault isolation per crawler instance.

### URL Frontier
- **Current**: In-memory visited set.
- **Recommendation**: Use Redis or a distributed bloom filter for the URL frontier, tracking visited URLs and crawl priorities.
- **Benefit**: Shared state across crawler instances; memory-efficient dedup.

---

## Phase 3: Production Search (4–8 weeks)

### Elasticsearch Integration
- **Current**: Custom in-memory inverted index.
- **Recommendation**: Replace with Elasticsearch or OpenSearch for the search backend.
- **Benefit**: Battle-tested full-text search, built-in ranking (BM25), faceting, pagination, highlighting, near-real-time indexing.

### Search Features
- **Phrase search**: `"exact phrase"` matching.
- **Filters**: By domain, depth, crawl date.
- **Pagination**: Cursor-based navigation for large result sets.
- **Auto-complete / Suggestions**: Using Elasticsearch suggesters.

### Caching Layer
- **Recommendation**: Add Redis caching for frequent search queries.
- **TTL**: 5–30 minutes depending on crawl freshness requirements.
- **Benefit**: Reduces search latency; protects the index from query storms.

---

## Phase 4: Operational Excellence (Ongoing)

### Monitoring & Observability
- **Metrics**: Prometheus + Grafana for crawl rate, index size, search latency, error rates.
- **Logging**: Structured logging (JSON) with ELK stack or Loki.
- **Tracing**: Distributed tracing with OpenTelemetry for cross-service request flows.

### Containerization & Orchestration
- **Docker**: Containerize each component (crawler, indexer, API).
- **Kubernetes**: Deploy with Helm charts; use HPA for auto-scaling crawlers based on queue depth.

### Deduplication & Quality
- **Content hashing**: SimHash or MinHash to detect near-duplicate pages.
- **Language detection**: Filter to target languages only.
- **Content quality signals**: Penalize thin content, boilerplate, and spam pages.

---

## Architecture Evolution

### Current (Minimal)
```
Single Process
├── Goroutine Pool (Crawlers)
├── Channel
├── Goroutine (Indexer)
├── In-Memory Storage
└── HTTP Server
```

### Target (Production)
```
┌──────────────┐     ┌───────┐     ┌──────────────┐
│ Crawler Fleet│────▶│ Kafka │────▶│ Indexer Fleet │
│ (K8s pods)   │     │       │     │ (K8s pods)    │
└──────────────┘     └───────┘     └──────┬───────┘
       │                                   │
       ▼                                   ▼
┌──────────────┐                  ┌──────────────┐
│    Redis     │                  │Elasticsearch │
│  (Frontier)  │                  │  (Index)     │
└──────────────┘                  └──────┬───────┘
                                         │
                                  ┌──────▼───────┐
                                  │  API Gateway │
                                  │   + Cache    │
                                  └──────────────┘
```

---

## Technology Recommendations

| Component | Current | Recommended |
|-----------|---------|-------------|
| Crawl Storage | In-memory map | Redis / PostgreSQL |
| URL Frontier | In-memory set | Redis + Bloom filter |
| Message Queue | Go channel | Apache Kafka / RabbitMQ |
| Search Index | Custom inverted index | Elasticsearch / OpenSearch |
| Caching | None | Redis |
| Monitoring | Log statements | Prometheus + Grafana |
| Deployment | Single binary | Docker + Kubernetes |
| Rate Limiting | None | Token bucket per domain |

---

## Key Takeaways

1. **The current architecture is a solid foundation** — clean separation of concerns makes each component independently replaceable.
2. **Go channels map naturally to message queues** — the channel-based design translates directly to Kafka/RabbitMQ topics.
3. **The inverted index interface maps to Elasticsearch** — the `SearchIndex(token)` abstraction can be swapped for ES queries.
4. **Horizontal scaling is the primary lever** — adding more crawler and indexer pods is the most impactful scaling strategy.
