package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// maxSources membatasi jumlah sumber rujukan yang dipakai.
const maxSources = 5

// SearchClient adalah client HTTP ke Search Engine.
type SearchClient struct {
	endpoint string
	http     *http.Client
}

// NewSearchClient membuat SearchClient untuk endpoint tertentu.
func NewSearchClient(endpoint string) *SearchClient {
	return &SearchClient{endpoint: endpoint, http: &http.Client{Timeout: 20 * time.Second}}
}

type searchPayload struct {
	Results []Source `json:"results"`
	Data    struct {
		Results []Source `json:"results"`
	} `json:"data"`
}

// Search melakukan penelusuran fakta terkini untuk sebuah topik.
func (c *SearchClient) Search(ctx context.Context, query string) ([]Source, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+"?q="+url.QueryEscape(query), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search engine returned status %d", resp.StatusCode)
	}
	var payload searchPayload
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	results := payload.Results
	if len(results) == 0 {
		results = payload.Data.Results
	}
	if len(results) > maxSources {
		results = results[:maxSources]
	}
	return results, nil
}
