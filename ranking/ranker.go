package ranking

import (
	"strings"
	"webcrawler/storage"
)

// Ranker computes relevancy scores for pages against search queries.
type Ranker struct {
	TitleWeight float64
	BodyWeight  float64
}

// New creates a Ranker with default weights.
func New() *Ranker {
	return &Ranker{
		TitleWeight: 2.0,
		BodyWeight:  1.0,
	}
}

// Score computes the relevancy score for a page given query tokens.
func (r *Ranker) Score(page *storage.PageData, queryTokens []string) float64 {
	var totalFrequency float64
	bodyLower := strings.ToLower(page.Body)
	titleLower := strings.ToLower(page.Title)

	for _, token := range queryTokens {
		// Calculate frequency in both title and body
		totalFrequency += float64(strings.Count(titleLower, token)) * r.TitleWeight
		totalFrequency += float64(strings.Count(bodyLower, token)) * r.BodyWeight
	}

	if totalFrequency == 0 {
		return 0
	}

	// Advanced formula: (frequency bonus) + (exact match base) - (depth penalty)
	score := (totalFrequency * 10) + 1000 - float64(page.Depth*5)
	return score
}
