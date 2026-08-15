package feed

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

//go:embed mockdata/sample_items.json
var sampleItemsJSON []byte

type mockFixture struct {
	ExternalID    string `json:"external_id"`
	SourceAccount string `json:"source_account"`
	Title         string `json:"title"`
	Content       string `json:"content"`
	URL           string `json:"url"`
}

// MockSource replays the bundled fixture a few items at a time on every Poll,
// simulating a live feed without hitting any network. Used by default
// (FEATURE_LIVE_FEED=false) so the rest of the pipeline can be built and
// tested before a real feed source is wired up.
type MockSource struct {
	mu        sync.Mutex
	items     []mockFixture
	cursor    int
	lap       int
	batchSize int
}

func NewMockSource() (*MockSource, error) {
	var items []mockFixture
	if err := json.Unmarshal(sampleItemsJSON, &items); err != nil {
		return nil, fmt.Errorf("parsing mock fixture: %w", err)
	}
	return &MockSource{items: items, batchSize: 2}, nil
}

func (m *MockSource) Name() string { return "mock" }

func (m *MockSource) Poll(_ context.Context) ([]Item, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.items) == 0 {
		return nil, nil
	}

	var out []Item
	now := time.Now()
	for i := 0; i < m.batchSize; i++ {
		if m.cursor >= len(m.items) {
			m.cursor = 0
			m.lap++
		}
		src := m.items[m.cursor]
		externalID := src.ExternalID
		if m.lap > 0 {
			// Re-surface the fixture on subsequent laps under a fresh id so the
			// unique-external_id dedup constraint doesn't just swallow it forever.
			externalID = fmt.Sprintf("%s-lap%d", src.ExternalID, m.lap)
		}
		out = append(out, Item{
			ExternalID:    externalID,
			SourceAccount: src.SourceAccount,
			Title:         src.Title,
			Content:       src.Content,
			URL:           src.URL,
			PublishedAt:   now,
		})
		m.cursor++
	}
	return out, nil
}
