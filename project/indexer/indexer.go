// Package indexer provides live indexing of crawled pages into an inverted index.
package indexer

import (
	"log"
	"strings"
	"unicode"

	"webcrawler/storage"
)

// Indexer reads crawled pages from a channel and builds a searchable inverted index.
type Indexer struct {
	store *storage.Storage
}

// New creates a new Indexer instance.
func New(store *storage.Storage) *Indexer {
	return &Indexer{store: store}
}

// Start begins the indexing loop. It reads pages from pageCh and indexes them.
// This method blocks until pageCh is closed (i.e., crawling is done).
// It should be run in a goroutine.
func (idx *Indexer) Start(pageCh <-chan *storage.PageData) {
	log.Println("[Indexer] Started. Waiting for pages...")

	for page := range pageCh {
		idx.IndexPage(page)
	}

	log.Println("[Indexer] Page channel closed. Indexing complete.")
}

// IndexPage tokenizes a page's title and body and adds it to the inverted index.
// It is exported so it can be called directly during persistence restore.
func (idx *Indexer) IndexPage(page *storage.PageData) {
	// Combine title and body for tokenization
	combined := page.Title + " " + page.Body
	tokens := Tokenize(combined)

	// Deduplicate tokens — we only need to index each unique token once per page
	unique := uniqueTokens(tokens)

	idx.store.AddToIndex(unique, page)

	log.Printf("[Indexer] Indexed %s (%d unique tokens)", page.URL, len(unique))
}

// Tokenize splits text into lowercase alphabetic tokens.
// It filters out very short tokens (len < 2) and common stop words.
func Tokenize(text string) []string {
	text = strings.ToLower(text)

	words := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})

	var tokens []string
	for _, w := range words {
		if len(w) < 2 {
			continue
		}
		if isStopWord(w) {
			continue
		}
		tokens = append(tokens, w)
	}
	return tokens
}

// uniqueTokens returns a deduplicated slice of tokens.
func uniqueTokens(tokens []string) []string {
	seen := make(map[string]bool, len(tokens))
	var result []string
	for _, t := range tokens {
		if !seen[t] {
			seen[t] = true
			result = append(result, t)
		}
	}
	return result
}

// isStopWord returns true if the word is a common English stop word.
func isStopWord(w string) bool {
	stopWords := map[string]bool{
		"the": true, "is": true, "at": true, "which": true,
		"on": true, "a": true, "an": true, "and": true,
		"or": true, "but": true, "in": true, "with": true,
		"to": true, "for": true, "of": true, "by": true,
		"from": true, "as": true, "it": true, "that": true,
		"this": true, "was": true, "are": true, "be": true,
		"has": true, "had": true, "have": true, "not": true,
		"no": true, "do": true, "does": true, "did": true,
		"will": true, "would": true, "could": true, "should": true,
		"may": true, "might": true, "can": true, "if": true,
		"then": true, "than": true, "so": true, "we": true,
		"you": true, "he": true, "she": true, "they": true,
		"its": true, "his": true, "her": true, "our": true,
		"your": true, "their": true, "been": true, "being": true,
	}
	return stopWords[w]
}
