package fetcher

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
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
	// Mirror downloads asset resources (such as images, CSS, and JavaScript)
	// within the HTML page to a local folder.
	Mirror bool
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

// Mirror turns on mirror downloading.
func Mirror(a ...bool) FetcherOption {
	return func(f *Fetcher) {
		if len(a) > 0 {
			f.Mirror = a[0]
		} else {
			f.Mirror = true
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

	// Create file store.
	urlBaseName := constructURLBaseName(result.Response.Request.URL)

	fileStore, err := store.NewFileStore(os.Getenv("ROOT_STORE_DIR"), urlBaseName)
	if err != nil {
		result.Err = errors.WithMessage(err, "failed to new file store")
		return result.Err
	}

	// Process response body.
	result.Metadata, err = f.process(fileStore, result.Response)
	if err != nil {
		result.Err = errors.WithMessage(err, "failed to process HTML response")
		return result.Err
	}

	return nil
}

func (f *Fetcher) process(fs *store.FileStore, resp *http.Response) (*types.Metadata, error) {
	// Parse `Content-Type` from header.
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(strings.ToLower(contentType), "html") {
		return nil, errors.Errorf(
			"response content type expected HTML got %s", contentType,
		)
	}

	buf := bytes.NewBuffer(nil)
	teeReader := io.TeeReader(resp.Body, buf)

	// Prepare HTML DOM parser.
	domParser, err := parser.NewParser(teeReader)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to new DOM parser")
	}

	// Process metadata.
	metadata, err := f.processMetadata(fs, domParser)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to process metadata")
	}

	// Process mirror downloading.
	if f.Mirror {
		var assets []*types.EmbeddedAsset
		baseUrlObj := determineBaseURL(resp.Request.URL, domParser)

		domParser.ReplaceAssets(func(assetURL string) (string, bool) {
			// Filter invalid asset URL
			assetUrlObj, err := url.Parse(assetURL)
			if err != nil {
				return "", false
			}

			// We only download the assets with the same domain host as the page, as they are more likely
			// to be relevant and accessible. For the assets with external domain hosts, we should be more
			// careful and selective, as they may be irrelevant, inaccessible, or restricted by CORS.
			assetAbsUrlObj := baseUrlObj.ResolveReference(assetUrlObj)
			if !strings.EqualFold(assetAbsUrlObj.Host, resp.Request.URL.Host) {
				return "", false
			}

			as := &types.EmbeddedAsset{AbsURL: assetAbsUrlObj}
			assets = append(assets, as)

			asFileURL := url.URL{
				Scheme: "file",
				Path:   fs.AssetFilePath(as),
			}
			return asFileURL.String(), true
		})

		if err := f.processAssets(assets, fs); err != nil {
			return nil, errors.WithMessage(err, "failed to process assets")
		}
	}

	// Save HTML doc file.
	if err := fs.SaveDoc(domParser.Document); err != nil {
		return nil, errors.WithMessage(err, "failed to save HTML document")
	}

	return metadata, nil
}

func (f *Fetcher) processAssets(assets []*types.EmbeddedAsset, fs *store.FileStore) error {
	for _, as := range assets {
		// Download the asset
		req, err := http.NewRequest(http.MethodGet, as.AbsURL.String(), nil)
		if err != nil {
			return errors.WithMessage(err, "failed to create HTTP request")
		}

		resp, err := f.client.Do(context.Background(), req)
		if err != nil {
			return errors.WithMessage(err, "failed to do HTTP request")
		}
		defer resp.Body.Close()

		as.DataReader = resp.Body
		if err := fs.SaveAsset(as); err != nil {
			return errors.WithMessage(err, "failed to save asset")
		}
	}

	return nil
}

func (f *Fetcher) processMetadata(
	fs *store.FileStore, parser *parser.Parser) (*types.Metadata, error) {

	// Extract and merge metadata.
	oldMetadata, err := fs.LoadMetadata()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to load metadata")
	}

	// Merge old metadata.
	metadata := parser.ExtractMetadata()
	metadata.FetchedAt = time.Now()
	if oldMetadata != nil {
		metadata.LastFetchedAt = &oldMetadata.FetchedAt
	}

	// Save metadata file.
	if err := fs.SaveMetadata(metadata); err != nil {
		return nil, errors.WithMessage(err, "failed to save metadata file")
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
