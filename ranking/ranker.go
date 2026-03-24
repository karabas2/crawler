// Package ranking implements a simple relevancy scoring heuristic for search results.
package ranking

import (
	"strings"

	"webcrawler/storage"
)

// Ranker computes relevancy scores for pages against search queries.
type Ranker struct {
	// TitleWeight is the multiplier for title matches.
	TitleWeight float64
	// BodyWeight is the multiplier for body keyword frequency.
	BodyWeight float64
}

// New creates a Ranker with default weights.
func New() *Ranker {
	return &Ranker{
		TitleWeight: 2.0,
		BodyWeight:  1.0,
	}
}

// Score computes the relevancy score for a page given query tokens.
//
// Formula: score = TitleWeight * titleMatchCount + BodyWeight * bodyKeywordFrequency
//
// Where:
//   - titleMatchCount: number of query tokens found in the page title
//   - bodyKeywordFrequency: total number of query token occurrences in the page body
func (r *Ranker) Score(page *storage.PageData, queryTokens []string) float64 {
	var totalFrequency float64
	bodyLower := strings.ToLower(page.Body)
	titleLower := strings.ToLower(page.Title)

	for _, token := range queryTokens {
		// Count occurrences in body and title
		totalFrequency += float64(strings.Count(bodyLower, token))
		totalFrequency += float64(strings.Count(titleLower, token))
	}

	if totalFrequency == 0 {
		return 0
	}

	// Formula: score = (frequency x 10) + 1000 (exact match bonus) - (depth x 5)
	score := (totalFrequency * 10) + 1000 - float64(page.Depth*5)
	return score
}
