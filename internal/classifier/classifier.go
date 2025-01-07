package classifier

import (
	"strings"
)

type Classifier interface {
	ClassifyContent(content string) []string
}

type SimpleClassifier struct {
	minConfidence float64
	maxTags      int
}

func NewSimpleClassifier(minConfidence float64, maxTags int) *SimpleClassifier {
	return &SimpleClassifier{
		minConfidence: minConfidence,
		maxTags:      maxTags,
	}
}

// Simple implementation that extracts hashtags and common keywords
func (c *SimpleClassifier) ClassifyContent(content string) []string {
	words := strings.Fields(content)
	tags := make(map[string]struct{})
	
	// Extract hashtags
	for _, word := range words {
		if strings.HasPrefix(word, "#") {
			tag := strings.ToLower(strings.TrimPrefix(word, "#"))
			if tag != "" {
				tags[tag] = struct{}{}
			}
		}
	}

	// Extract common categories based on keywords
	categories := map[string][]string{
		"work":      {"project", "meeting", "deadline", "task", "report"},
		"personal":  {"family", "friend", "home", "birthday", "holiday"},
		"shopping":  {"buy", "purchase", "store", "shop", "price"},
		"education": {"study", "learn", "course", "book", "homework"},
		"travel":    {"trip", "flight", "hotel", "vacation", "booking"},
	}

	content = strings.ToLower(content)
	for category, keywords := range categories {
		for _, keyword := range keywords {
			if strings.Contains(content, keyword) {
				tags[category] = struct{}{}
				break
			}
		}
	}

	// Convert tags map to slice
	result := make([]string, 0, len(tags))
	for tag := range tags {
		result = append(result, tag)
	}

	// Limit the number of tags
	if len(result) > c.maxTags {
		result = result[:c.maxTags]
	}

	return result
} 