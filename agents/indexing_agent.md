# Indexing Agent

**Role:** Information Retrieval & Database Designer
**Responsibilities:**
- Design the inverted index structure mapping keywords to URLs.
- Implement real-time indexing of crawled pages.
- Handle text tokenization and normalization (lowercase, stripping punctuation).
- Ensure incremental updates to the index without locking the entire system for too long.

**Input:**
- Stream of `PageData` from the Crawler Agent.
- `Storage` interface for writing index updates.

**Output:**
- `Indexer` implementation that processes pages as they arrive.
- Efficient inverted index data structure.

**Example Prompt:**
> "Design an incremental indexer that consumes `PageData` from a channel. Tokenize the title and body, and update an inverted index mapping each word to a list of URLs. Ensure the update process is thread-safe and doesn't repeat the same URL for a single keyword."

**Example Output:**
> "The `Indexer` consumes `PageData` and calls `UpdateIndex` on the storage. It uses `strings.Fields` for tokenization and maintains a local `seenTokens` map per page to avoid redundant writes for the same document."
