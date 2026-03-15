// Package crawler implements a concurrent BFS web crawler with depth tracking.
package crawler

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"webcrawler/storage"

	"golang.org/x/net/html"
)

// CrawlTask represents a single unit of work for the crawler.
type CrawlTask struct {
	URL       string // The URL to crawl
	OriginURL string // The URL from which this was discovered
	Depth     int    // Current crawl depth
}

// Crawler orchestrates concurrent web crawling with a worker pool.
type Crawler struct {
	store      *storage.Storage
	maxDepth   int
	maxWorkers int
	maxURLs    int
	userAgent  string

	visited   map[string]bool
	visitedMu sync.Mutex

	taskCh chan CrawlTask
	PageCh chan *storage.PageData

	wg     sync.WaitGroup
	client *http.Client
	done   chan struct{}

	// DoneCh is closed when all crawl workers finish.
	DoneCh chan struct{}

	// Rate limiting
	rateTicker *time.Ticker

	// URL count tracking for MaxURLs
	urlCount   int
	urlCountMu sync.Mutex
}

// Config holds the crawler configuration parameters.
type Config struct {
	MaxDepth      int
	MaxWorkers    int
	MaxURLs       int     // 0 = unlimited
	HitRate       float64 // requests per second, 0 = unlimited
	QueueCapacity int     // 0 = default (workers*10)
	UserAgent     string
}

// New creates a new Crawler instance.
func New(store *storage.Storage, cfg Config) *Crawler {
	if cfg.MaxWorkers <= 0 {
		cfg.MaxWorkers = 5
	}
	if cfg.MaxDepth <= 0 {
		cfg.MaxDepth = 2
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = "GoCrawler/1.0"
	}
	queueCap := cfg.QueueCapacity
	if queueCap <= 0 {
		queueCap = 10000
	}

	c := &Crawler{
		store:      store,
		maxDepth:   cfg.MaxDepth,
		maxWorkers: cfg.MaxWorkers,
		maxURLs:    cfg.MaxURLs,
		userAgent:  cfg.UserAgent,
		visited:    make(map[string]bool),
		taskCh:     make(chan CrawlTask, queueCap),
		PageCh:     make(chan *storage.PageData, queueCap),
		client: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 3 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
		done:   make(chan struct{}),
		DoneCh: make(chan struct{}),
	}

	if cfg.HitRate > 0 {
		c.rateTicker = time.NewTicker(time.Duration(float64(time.Second) / cfg.HitRate))
	}

	return c
}

// Start begins the crawl from the given seed URL.
// It launches worker goroutines and returns immediately.
// The crawler runs in the background until Stop() is called or all tasks are exhausted.
func (c *Crawler) Start(seedURL string) {
	log.Printf("[Crawler] Starting with seed=%s, depth=%d, workers=%d",
		seedURL, c.maxDepth, c.maxWorkers)

	// Launch worker pool
	for i := 0; i < c.maxWorkers; i++ {
		c.wg.Add(1)
		go c.worker(i)
	}

	// Enqueue the seed URL
	c.enqueue(CrawlTask{
		URL:       seedURL,
		OriginURL: "",
		Depth:     0,
	})

	// Monitor: close taskCh when all work is done
	go func() {
		c.wg.Wait()
		close(c.PageCh)
		close(c.DoneCh)
		log.Println("[Crawler] All workers finished. Crawl complete.")
	}()
}

// Stop signals the crawler to stop processing new tasks.
func (c *Crawler) Stop() {
	close(c.done)
	if c.rateTicker != nil {
		c.rateTicker.Stop()
	}
}

// IsRunning returns true if the crawler is still actively processing.
func (c *Crawler) IsRunning() bool {
	select {
	case <-c.DoneCh:
		return false
	default:
		return true
	}
}

// MarkVisited marks a list of URLs as already visited.
// Used during persistence restore to skip already-crawled pages.
func (c *Crawler) MarkVisited(urls []string) {
	c.visitedMu.Lock()
	defer c.visitedMu.Unlock()
	for _, u := range urls {
		c.visited[u] = true
	}
	log.Printf("[Crawler] Marked %d URLs as already visited (restored from persistence)", len(urls))
}

// ResumeFrom takes restored pages and pre-fills the task channel with their
// unvisited child links so the crawler continues from where it left off.
// Must be called BEFORE Start() since workers haven't launched yet.
func (c *Crawler) ResumeFrom(pages []*storage.PageData) {
	enqueuedCount := 0
	capacity := cap(c.taskCh)

	c.visitedMu.Lock()
	for _, page := range pages {
		if page.Depth < c.maxDepth {
			for _, link := range page.Links {
				if c.visited[link] {
					continue
				}
				c.visited[link] = true

				// Stop if we'd exceed channel capacity (workers will discover more)
				if enqueuedCount >= capacity {
					break
				}

				c.taskCh <- CrawlTask{
					URL:       link,
					OriginURL: page.URL,
					Depth:     page.Depth + 1,
				}
				c.store.IncrURLsQueued()
				c.wg.Add(1)
				enqueuedCount++
			}
		}
		if enqueuedCount >= capacity {
			break
		}
	}
	c.visitedMu.Unlock()

	log.Printf("[Crawler] Resumed: enqueued %d new URLs from restored pages", enqueuedCount)
}

// enqueue adds a crawl task if the URL hasn't been visited yet.
func (c *Crawler) enqueue(task CrawlTask) {
	c.visitedMu.Lock()
	if c.visited[task.URL] {
		c.visitedMu.Unlock()
		return
	}
	c.visited[task.URL] = true
	c.visitedMu.Unlock()

	c.store.IncrURLsQueued()

	c.wg.Add(1)

	// Track queue depth and back-pressure
	queueLen := len(c.taskCh)
	queueCap := cap(c.taskCh)
	c.store.UpdateQueueDepth(queueLen)
	if queueLen >= queueCap*8/10 {
		c.store.UpdateBackPressure("HIGH — queue near capacity")
	} else if queueLen >= queueCap/2 {
		c.store.UpdateBackPressure("MODERATE — queue filling up")
	} else {
		c.store.UpdateBackPressure("NORMAL")
	}

	select {
	case c.taskCh <- task:
	case <-c.done:
		c.wg.Done()
	}
}

// worker is a goroutine that processes crawl tasks from the task channel.
func (c *Crawler) worker(id int) {
	defer c.wg.Done()

	for {
		select {
		case <-c.done:
			return
		case task, ok := <-c.taskCh:
			if !ok {
				return
			}
			c.processTask(id, task)
		default:
			// No tasks available, brief sleep to avoid busy-waiting
			time.Sleep(100 * time.Millisecond)
			// Check if there are still tasks or if we should exit
			select {
			case <-c.done:
				return
			case task, ok := <-c.taskCh:
				if !ok {
					return
				}
				c.processTask(id, task)
			case <-time.After(2 * time.Second):
				// Timeout waiting for tasks — worker exits
				return
			}
		}
	}
}

// processTask fetches a URL, parses its HTML, stores the page, and enqueues discovered links.
func (c *Crawler) processTask(workerID int, task CrawlTask) {
	defer c.wg.Done()

	// Max URL limit check
	if c.maxURLs > 0 {
		c.urlCountMu.Lock()
		if c.urlCount >= c.maxURLs {
			c.urlCountMu.Unlock()
			return
		}
		c.urlCount++
		c.urlCountMu.Unlock()
	}

	// Rate limiting
	if c.rateTicker != nil {
		select {
		case <-c.rateTicker.C:
		case <-c.done:
			return
		}
	}

	log.Printf("[Worker %d] Crawling depth=%d url=%s", workerID, task.Depth, task.URL)

	page, err := c.fetchAndParse(task)
	if err != nil {
		log.Printf("[Worker %d] Error crawling %s: %v", workerID, task.URL, err)
		c.store.IncrURLsFailed()
		c.store.IncrURLsProcessed()
		return
	}

	c.store.IncrURLsProcessed()

	// Store the page (thread-safe)
	c.store.StorePage(page)

	// Send to indexer for live indexing
	select {
	case c.PageCh <- page:
	case <-c.done:
		return
	}

	// Enqueue discovered links if we haven't reached max depth
	if task.Depth < c.maxDepth {
		for _, link := range page.Links {
			c.enqueue(CrawlTask{
				URL:       link,
				OriginURL: task.URL,
				Depth:     task.Depth + 1,
			})
		}
	}
}

// fetchAndParse downloads a page and extracts title, body text, and links.
func (c *Crawler) fetchAndParse(task CrawlTask) (*storage.PageData, error) {
	req, err := http.NewRequest("GET", task.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	// Only process HTML pages
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		return nil, fmt.Errorf("non-HTML content type: %s", ct)
	}

	// Limit body size to 1MB
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("reading body: %w", err)
	}

	title, text, links := parseHTML(string(body), task.URL)

	return &storage.PageData{
		URL:       task.URL,
		Title:     title,
		Body:      text,
		Links:     links,
		OriginURL: task.OriginURL,
		Depth:     task.Depth,
	}, nil
}

// parseHTML extracts the title, visible text, and absolute links from an HTML document.
func parseHTML(htmlContent string, baseURL string) (title string, text string, links []string) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return "", htmlContent, nil
	}

	base, _ := url.Parse(baseURL)

	var titleBuilder strings.Builder
	var textBuilder strings.Builder
	var foundLinks []string
	var inTitle bool

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "title":
				inTitle = true
			case "script", "style", "noscript":
				return // skip script/style content
			case "a":
				for _, attr := range n.Attr {
					if attr.Key == "href" {
						link := resolveURL(base, attr.Val)
						if link != "" {
							foundLinks = append(foundLinks, link)
						}
					}
				}
			}
		}

		if n.Type == html.TextNode {
			cleaned := strings.TrimSpace(n.Data)
			if cleaned != "" {
				if inTitle {
					titleBuilder.WriteString(cleaned)
				}
				textBuilder.WriteString(cleaned)
				textBuilder.WriteString(" ")
			}
		}

		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}

		if n.Type == html.ElementNode && n.Data == "title" {
			inTitle = false
		}
	}

	walk(doc)

	return titleBuilder.String(), textBuilder.String(), foundLinks
}

// resolveURL converts a potentially relative URL to an absolute URL.
// Returns empty string for non-HTTP(S) links, fragments, and javascript: URLs.
func resolveURL(base *url.URL, href string) string {
	href = strings.TrimSpace(href)
	if href == "" || strings.HasPrefix(href, "#") ||
		strings.HasPrefix(href, "javascript:") ||
		strings.HasPrefix(href, "mailto:") {
		return ""
	}

	parsed, err := url.Parse(href)
	if err != nil {
		return ""
	}

	resolved := base.ResolveReference(parsed)

	// Only keep HTTP/HTTPS links
	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return ""
	}

	// Strip fragment
	resolved.Fragment = ""

	return resolved.String()
}
