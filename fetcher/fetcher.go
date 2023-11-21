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
func (f *Fetcher) Fetch(strURL string) error {
	urlObj, err := url.Parse(strURL)
	if err != nil {
		return errors.WithMessage(err, "invalid web URL")
	}

	f.wg.Add(1)
	if f.Async {
		go f.scrape(urlObj)
		return nil
	}

	return f.scrape(urlObj)
}

func (f *Fetcher) scrape(urlObj *url.URL) (err error) {
	var result *types.FetchResult

	defer func() {
		f.handleOnFetched(result, err)
		f.wg.Done()
	}()

	req, err := http.NewRequest(http.MethodGet, urlObj.String(), nil)
	if err != nil {
		return errors.WithMessage(err, "failed to create HTTP request")
	}

	resp, err := f.client.Do(context.Background(), req)
	if err != nil {
		return errors.WithMessage(err, "failed to do HTTP request")
	}
	defer resp.Body.Close()

	result, err = f.process(resp)
	if err != nil {
		return errors.WithMessage(err, "failed to process HTML response")
	}

	return nil
}

func (f *Fetcher) process(resp *http.Response) (*types.FetchResult, error) {
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
	fileStore, err := store.NewFileStore(resp.Request.URL)
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
		metadata.LastFetchedAt = oldMetadata.FetchedAt
	}

	// Save metadata file.
	if err := fileStore.SaveMetadata(metadata); err != nil {
		return nil, errors.WithMessage(err, "failed to save metadata file")
	}

	// Save HTML doc file
	if err := fileStore.SaveHTMLDoc(domParser.Document); err != nil {
		return nil, errors.WithMessage(err, "failed to save HTML document")
	}

	return &types.FetchResult{
		Metadata: metadata,
		Response: resp,
	}, nil
}

// FetchedCallback is a type alias for the callback function executed upon
// task completion.
type FetchedCallback func(result *types.FetchResult, err error)

// OnFetched registers a callback function to be invoked after each task finishes.
// NB this function is not thread safe.
func (f *Fetcher) OnFetched(cb FetchedCallback) {
	f.callbacks = append(f.callbacks, cb)
}

func (f *Fetcher) handleOnFetched(result *types.FetchResult, err error) {
	for _, cb := range f.callbacks {
		cb(result, err)
	}
}
