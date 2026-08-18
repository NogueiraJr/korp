package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

var (
	ErrEstoqueUnavailable = errors.New("estoque service unavailable")
	ErrInsufficientStock  = errors.New("insufficient stock")
	ErrProductNotFound    = errors.New("product not found in estoque")
)

// breaker is a minimal circuit breaker: after consecutiveFailures failures it
// opens for resetTimeout, failing fast. After the timeout it allows one probe
// (half-open) and closes again on success.
type breaker struct {
	mu                  sync.Mutex
	failures            int
	openedAt            time.Time
	open                bool
	threshold           int
	resetTimeout        time.Duration
}

func newBreaker(threshold int, resetTimeout time.Duration) *breaker {
	return &breaker{threshold: threshold, resetTimeout: resetTimeout}
}

func (b *breaker) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.open {
		return true
	}
	if time.Since(b.openedAt) >= b.resetTimeout {
		b.open = false
		b.failures = 0
		return true
	}
	return false
}

func (b *breaker) recordFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures++
	if b.failures >= b.threshold {
		b.open = true
		b.openedAt = time.Now()
	}
}

func (b *breaker) recordSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures = 0
	b.open = false
}

type EstoqueClient struct {
	baseURL     string
	internalTok string
	http        *http.Client
	maxRetries  int
	breaker     *breaker
}

func NewEstoqueClient(baseURL, internalToken string) *EstoqueClient {
	return &EstoqueClient{
		baseURL:     baseURL,
		internalTok: internalToken,
		http:        &http.Client{Timeout: 10 * time.Second},
		maxRetries:  3,
		breaker:     newBreaker(3, 15*time.Second),
	}
}

type ConsumeRequest struct {
	Items []ConsumeItem `json:"items"`
}

type ConsumeItem struct {
	Code     string `json:"code"`
	Quantity int    `json:"quantity"`
}

// ConsumeStock calls the Estoque microservice to deduct stock atomically.
// It retries with exponential backoff on transient errors and uses a circuit
// breaker to fail fast while the dependency is down.
func (c *EstoqueClient) ConsumeStock(ctx context.Context, items []ConsumeItem) error {
	if !c.breaker.allow() {
		return fmt.Errorf("%w: circuit breaker open", ErrEstoqueUnavailable)
	}

	payload, err := json.Marshal(ConsumeRequest{Items: items})
	if err != nil {
		return err
	}

	var lastErr error
	backoff := 300 * time.Millisecond
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
			backoff *= 2
		}

		err := c.doConsume(ctx, payload)
		if err == nil {
			c.breaker.recordSuccess()
			return nil
		}

		lastErr = err
		if !isRetryable(err) {
			c.breaker.recordFailure()
			return err
		}
	}

	c.breaker.recordFailure()
	return fmt.Errorf("%w: %v", ErrEstoqueUnavailable, lastErr)
}

func (c *EstoqueClient) doConsume(ctx context.Context, payload []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/products/consume", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Token", c.internalTok)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrEstoqueUnavailable, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))

	switch {
	case resp.StatusCode == http.StatusOK:
		return nil
	case resp.StatusCode == http.StatusConflict:
		return ErrInsufficientStock
	case resp.StatusCode == http.StatusNotFound:
		return ErrProductNotFound
	case resp.StatusCode >= http.StatusInternalServerError:
		return fmt.Errorf("%w: estoque returned status %d: %s",
			ErrEstoqueUnavailable, resp.StatusCode, truncate(string(body), 200))
	default:
		return fmt.Errorf("estoque returned status %d: %s", resp.StatusCode, truncate(string(body), 200))
	}
}

func isRetryable(err error) bool {
	return errors.Is(err, ErrEstoqueUnavailable)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}