package fetcher

import (
	"context"
	"net/http"
	"time"
)

// ThrottleClient is a throttled HTTP client that limits the number of concurrent requests to
// avoid resource overload and rate limiting issues.
type ThrottleClient struct {
	// Parallelism is the number of max allowed concurrent requests.
	// Default 0 with unlimited concurrencies.
	Parallelism int

	client *http.Client
	ch     chan struct{}
}

func NewThrottleClient(parallelism int) *ThrottleClient {
	c := &ThrottleClient{
		Parallelism: parallelism,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}

	if parallelism > 0 {
		c.ch = make(chan struct{}, parallelism)
	}

	return c
}

func (c *ThrottleClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	if c.Parallelism > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case c.ch <- struct{}{}:
		}

		defer func() {
			<-c.ch
		}()
	}

	return c.client.Do(req)
}
