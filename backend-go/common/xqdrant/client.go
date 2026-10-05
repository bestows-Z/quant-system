package xqdrant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	Addr       string
	Collection string
	httpClient *http.Client
}

type Point struct {
	ID      uint64         `json:"id"`
	Vector  []float64      `json:"vector"`
	Payload map[string]any `json:"payload"`
}

type ScoredPoint struct {
	ID      any            `json:"id"`
	Score   float64        `json:"score"`
	Payload map[string]any `json:"payload"`
}

func New(addr, collection string) *Client {
	return &Client{
		Addr:       strings.TrimRight(addr, "/"),
		Collection: collection,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *Client) EnsureCollection(ctx context.Context, collection string, dimension int) error {
	path := "/collections/" + url.PathEscape(c.collection(collection))
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err == nil && resp.StatusCode == http.StatusOK {
		resp.Body.Close()
		return nil
	}
	if resp != nil {
		resp.Body.Close()
	}
	body := map[string]any{"vectors": map[string]any{"size": dimension, "distance": "Cosine"}}
	resp, err = c.do(ctx, http.MethodPut, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return responseError(resp)
}

func (c *Client) Upsert(ctx context.Context, collection string, point Point) error {
	path := "/collections/" + url.PathEscape(c.collection(collection)) + "/points?wait=true"
	resp, err := c.do(ctx, http.MethodPut, path, map[string]any{"points": []Point{point}})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return responseError(resp)
}

func (c *Client) Search(ctx context.Context, collection string, vector []float64, topK int) ([]ScoredPoint, error) {
	path := "/collections/" + url.PathEscape(c.collection(collection)) + "/points/search"
	body := map[string]any{"vector": vector, "limit": topK, "with_payload": true}
	resp, err := c.do(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := responseError(resp); err != nil {
		return nil, err
	}
	var payload struct {
		Result []ScoredPoint `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode qdrant search response: %w", err)
	}
	return payload.Result, nil
}

func (c *Client) DeletePoints(ctx context.Context, collection string, pointIDs []uint64) error {
	if len(pointIDs) == 0 {
		return nil
	}
	path := "/collections/" + url.PathEscape(c.collection(collection)) + "/points/delete?wait=true"
	resp, err := c.do(ctx, http.MethodPost, path, map[string]any{"points": pointIDs})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return responseError(resp)
}

func (c *Client) DeleteCollection(ctx context.Context, collection string) error {
	path := "/collections/" + url.PathEscape(c.collection(collection))
	resp, err := c.do(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	return responseError(resp)
}

func (c *Client) Health(ctx context.Context) error {
	resp, err := c.do(ctx, http.MethodGet, "/", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return responseError(resp)
}

func (c *Client) collection(value string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return c.Collection
}

func (c *Client) do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.Addr+path, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.httpClient.Do(req)
}

func responseError(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return fmt.Errorf("qdrant returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
}
