# Crawler Agent

**Role:** Concurrency Specialist & Web Engineer
**Responsibilities:**
- Implement the BFS crawling logic with depth limit `k`.
- Manage the worker pool and task queue.
- Implement URL deduplication to prevent infinite loops.
- Handle back-pressure and rate limiting to avoid overwhelming the system or target servers.

**Input:**
- Seed URL and max depth `k` from the Planner.
- `Storage` interface for visited URL checks.

**Output:**
- `Crawler` implementation with worker pool.
- Stream of `PageData` objects sent to the Indexer.

**Example Prompt:**
> "Implement a concurrent BFS crawler in Go. Use a worker pool of 20 goroutines. Ensure no URL is crawled more than once by using a thread-safe map. Implement back-pressure by monitoring the task queue length and slowing down enqueueing if the queue exceeds 10,000 items."

**Example Output:**
> "Implemented `Crawler` struct with a `taskCh` channel of capacity 10,000. Workers fetch URLs using `http.Client` and parse links using `golang.org/x/net/html`. A `visitedMu sync.Mutex` protects the `visited` map."
