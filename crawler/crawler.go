package crawler

import (
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
	URL       string
	OriginURL string
	Depth     int
}

// Crawler orchestrates concurrent web crawling with rate limiting and backpressure.
type Crawler struct {
	store      *storage.Storage
	maxDepth   int
	workers    int
	userAgent  string

	visited   map[string]bool
	visitedMu sync.Mutex

	taskCh chan CrawlTask
	PageCh chan *storage.PageData

	wg     sync.WaitGroup
	DoneCh chan struct{}
	client *http.Client

	// Rate limiting
	rateTicker *time.Ticker
}

// New creates a new Crawler instance.
func New(store *storage.Storage, maxDepth, workers int) *Crawler {
	return &Crawler{
		store:      store,
		maxDepth:   maxDepth,
		workers:    workers,
		userAgent:  "AgenticCrawler/2.0 (HighPerformance Refined Version)",
		visited:    make(map[string]bool),
		taskCh:     make(chan CrawlTask, 10000),
		PageCh:     make(chan *storage.PageData, 100),
		DoneCh:     make(chan struct{}),
		client:     &http.Client{Timeout: 10 * time.Second},
		rateTicker: time.NewTicker(time.Second / 10), // Default 10 requests per second
	}
}

// Start begins the crawl from a seed URL.
func (c *Crawler) Start(seedURL string) {
	log.Printf("[Crawler] Starting crawl at %s (Depth: %d, Workers: %d)", seedURL, c.maxDepth, c.workers)

	// Start workers
	for i := 0; i < c.workers; i++ {
		go c.worker()
	}

	// Queue seed
	c.addTask(CrawlTask{URL: seedURL, OriginURL: "seed", Depth: 0})

	// Monitor completion
	go func() {
		c.wg.Wait()
		c.rateTicker.Stop()
		close(c.taskCh)
		close(c.PageCh)
		close(c.DoneCh)
		log.Println("[Crawler] Completed all tasks.")
	}()
}

func (c *Crawler) addTask(t CrawlTask) {
	c.visitedMu.Lock()
	if c.visited[t.URL] {
		c.visitedMu.Unlock()
		return
	}
	c.visited[t.URL] = true
	c.visitedMu.Unlock()

	c.wg.Add(1)
	c.taskCh <- t

	// Update storage stats
	c.store.UpdateStats(func(s *storage.Stats) {
		s.URLsQueued++
		s.QueueDepth = len(c.taskCh)
	})
}

func (c *Crawler) worker() {
	for task := range c.taskCh {
		<-c.rateTicker.C // Rate limiting

		c.process(task)
		
		c.wg.Done()
		c.store.UpdateStats(func(s *storage.Stats) {
			s.URLsProcessed++
			s.QueueDepth = len(c.taskCh)
		})
	}
}

func (c *Crawler) process(t CrawlTask) {
	if t.Depth > c.maxDepth {
		return
	}

	req, _ := http.NewRequest("GET", t.URL, nil)
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		c.store.UpdateStats(func(s *storage.Stats) { s.URLsFailed++ })
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}

	title, body, links := parseHTML(resp.Body, t.URL)

	page := &storage.PageData{
		URL:       t.URL,
		Title:     title,
		Body:      body,
		Links:     links,
		OriginURL: t.OriginURL,
		Depth:     t.Depth,
	}

	c.PageCh <- page

	// Queue child links
	if t.Depth < c.maxDepth {
		for _, link := range links {
			c.addTask(CrawlTask{URL: link, OriginURL: t.URL, Depth: t.Depth + 1})
		}
	}
}

func parseHTML(r io.Reader, base string) (string, string, []string) {
	var (
		title string
		body  strings.Builder
		links []string
	)

	doc, err := html.Parse(r)
	if err != nil {
		return "", "", nil
	}

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "title" && n.FirstChild != nil {
			title = n.FirstChild.Data
		}
		if n.Type == html.TextNode {
			p := n.Parent
			if p.Data != "script" && p.Data != "style" {
				body.WriteString(n.Data + " ")
			}
		}
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					if l := resolveURL(base, a.Val); l != "" {
						links = append(links, l)
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return title, strings.TrimSpace(body.String()), links
}

func resolveURL(base, href string) string {
	b, _ := url.Parse(base)
	r, err := url.Parse(href)
	if err != nil {
		return ""
	}
	res := b.ResolveReference(r)
	if res.Scheme != "http" && res.Scheme != "https" {
		return ""
	}
	res.Fragment = ""
	return res.String()
}
