package fetcher

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/wanliqun/web-fetcher/parser"
	"github.com/wanliqun/web-fetcher/store"
	"github.com/wanliqun/web-fetcher/types"
)

// FetcherConfig modifies fetcher behaviors.
type FetcherConfig struct {
	// Async turns on asynchronous HTTP requesting.
	Async bool
	// Further configurations such as HTTP configurations eg., user agent, proxy,
	// and timeout may be considered in future enhancements.
}

// FetcherOption builder option on a fetcher.
type FetcherOption func(*Fetcher)

// Fetcher provides scraper instances for fetching web pages.
type Fetcher struct {
	*FetcherConfig

	client    *ThrottleClient
	callbacks []FetchedCallback
	wg        *sync.WaitGroup
}

// NewFetcher creates a fetcher instance with builder options.
func NewFetcher(options ...FetcherOption) *Fetcher {
	f := &Fetcher{
		FetcherConfig: &FetcherConfig{},
		client:        NewThrottleClient(0),
		wg:            &sync.WaitGroup{},
	}

	for _, option := range options {
		option(f)
	}

	return f
}

// Async turns on asynchronous HTTP requesting.
func Async(a ...bool) FetcherOption {
	return func(f *Fetcher) {
		if len(a) > 0 {
			f.Async = a[0]
		} else {
			f.Async = true
		}
	}
}

// Fetch starts scraping by HTTP requesting to the specified URL.
// Fetching result will be notified by callback functions if registered.
func (f *Fetcher) Fetch(url string) error {
	f.wg.Add(1)
	if f.Async {
		go f.scrape(url)
		return nil
	}

	return f.scrape(url)
}

// Wait blocks until all scraping jobs are done.
func (f *Fetcher) Wait() {
	f.wg.Wait()
}

func (f *Fetcher) scrape(strURL string) error {
	result := &types.FetchResult{URL: strURL}

	defer func() {
		f.handleOnFetched(result)
		f.wg.Done()
	}()

	urlObj, err := url.Parse(strURL)
	if err != nil {
		result.Err = errors.WithMessage(err, "invalid web URL")
		return result.Err
	}

	req, err := http.NewRequest(http.MethodGet, urlObj.String(), nil)
	if err != nil {
		result.Err = errors.WithMessage(err, "failed to create HTTP request")
		return result.Err
	}

	result.Response, err = f.client.Do(context.Background(), req)
	if err != nil {
		result.Err = errors.WithMessage(err, "failed to do HTTP request")
		return result.Err
	}
	defer result.Response.Body.Close()

	// Check for successful status codes (2xx range).
	// Redirection status code 301 and 302 may be supported for future enhancement.
	if statusCode := result.Response.StatusCode; statusCode < 200 || statusCode > 299 {
		result.Err = errors.Errorf("bad HTTP status code: %d", statusCode)
		return result.Err
	}

	result.Metadata, err = f.process(result.Response)
	if err != nil {
		result.Err = errors.WithMessage(err, "failed to process HTML response")
		return result.Err
	}

	return nil
}

func (f *Fetcher) process(resp *http.Response) (*types.Metadata, error) {
	if !strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "html") {
		return nil, errors.New("response content type is not HTML")
	}

	buf := bytes.NewBuffer(nil)
	teeReader := io.TeeReader(resp.Body, buf)

	// Prepare HTML DOM parser.
	domParser, err := parser.NewParser(teeReader)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to new DOM parser")
	}

	// Prepare file store.
	urlBaseName := constructURLBaseName(resp.Request.URL)
	fileStore, err := store.NewFileStore(urlBaseName)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to new file store")
	}

	// Extract and merge metadata.
	oldMetadata, err := fileStore.LoadMetadata()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to load metadata")
	}

	metadata := domParser.ExtractMetadata()
	metadata.FetchedAt = time.Now()
	if oldMetadata != nil {
		metadata.LastFetchedAt = &oldMetadata.FetchedAt
	}

	// Save metadata file.
	if err := fileStore.SaveMetadata(metadata); err != nil {
		return nil, errors.WithMessage(err, "failed to save metadata file")
	}

	// Save HTML doc file
	if err := fileStore.SaveDoc(domParser.Document); err != nil {
		return nil, errors.WithMessage(err, "failed to save HTML document")
	}

	return metadata, nil
}

// FetchedCallback is a type alias for the callback function executed upon
// task completion.
type FetchedCallback func(result *types.FetchResult)

// OnFetched registers a callback function to be invoked after each task finishes.
// NB this function is not thread safe.
func (f *Fetcher) OnFetched(cb FetchedCallback) {
	f.callbacks = append(f.callbacks, cb)
}

func (f *Fetcher) handleOnFetched(result *types.FetchResult) {
	for _, cb := range f.callbacks {
		cb(result)
	}
}

// constructURLBaseName creates the base file name from a URL.
func constructURLBaseName(docURL *url.URL) string {
	docName := docURL.Host + docURL.Path
	// Incorporate the query parameters to generate a unique file name,
	// as distinct query parameters can represent different web pages.
	if len(docURL.RawQuery) > 0 {
		docName += "+" + docURL.RawQuery
	}

	return docName
}
