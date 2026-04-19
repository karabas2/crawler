package indexer

import (
	"log"
	"regexp"
	"strings"
	"webcrawler/storage"
)

var (
	stopWords = map[string]bool{
		"a": true, "an": true, "the": true, "and": true, "or": true, "but": true,
		"is": true, "are": true, "in": true, "on": true, "at": true, "to": true,
	}
	wordRegex = regexp.MustCompile(`[\p{L}0-9]+`)
)

// Indexer processes pages into an inverted index.
type Indexer struct {
	store *storage.Storage
}

// New creates a new Indexer.
func New(store *storage.Storage) *Indexer {
	return &Indexer{store: store}
}

// Start begins the indexing loop.
func (idx *Indexer) Start(pageCh <-chan *storage.PageData) {
	log.Println("[Indexer] Worker started. Monitoring page channel...")
	for page := range pageCh {
		idx.IndexPage(page)
	}
	log.Println("[Indexer] Worker finished.")
}

// IndexPage tokenizes and persists a page to the index.
func (idx *Indexer) IndexPage(page *storage.PageData) {
	combined := page.Title + " " + page.Body
	tokens := Tokenize(combined)
	unique := uniqueTokens(tokens)

	idx.store.AddPage(page)
	idx.store.AddToIndex(unique, page)
}

// Tokenize splits text into alphanumeric tokens, supporting Unicode.
func Tokenize(text string) []string {
	text = strings.ToLower(text)
	return wordRegex.FindAllString(text, -1)
}

func uniqueTokens(tokens []string) []string {
	set := make(map[string]bool)
	var res []string
	for _, t := range tokens {
		if !stopWords[t] && len(t) > 1 && !set[t] {
			set[t] = true
			res = append(res, t)
		}
	}
	return res
}
