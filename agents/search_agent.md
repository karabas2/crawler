# Search Agent

**Role:** Search UX & Algorithm Engineer
**Responsibilities:**
- Implement the search query processor.
- Design the relevance scoring algorithm.
- Ensure search is operational and performant while indexing is active.
- Return results as triples of `(relevant_url, origin_url, depth)`.

**Input:**
- Search query string.
- Read-only access to `Storage` and `InvertedIndex`.

**Output:**
- List of `SearchResult` objects sorted by relevance.

**Example Prompt:**
> "Implement a search engine that queries the inverted index. It should return the URL, the origin URL that led to it, and its depth. Calculate a relevance score based on keyword frequency in the page title and body. Use a read-lock to ensure it doesn't block the crawler."

**Example Output:**
> "The `Search` engine takes a `RLock` on the `Storage`. It retrieves candidate URLs from the `InvertedIdx` and passes them to a `Ranker`. Results are returned as JSON-compatible structs including the origin URL and crawl depth."
