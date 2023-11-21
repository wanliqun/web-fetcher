package types

import (
	"io"
	"net/http"
	"time"
)

// Metadata describes the structure and information of an HTML page.
type Metadata struct {
	// NumLinks: The total number of links found within the HTML page.
	NumLinks int
	// NumImages: The total number of images found within the HTML page.
	NumImages int
	// LastFetchedAt: The last time the HTML page was fetched.
	LastFetchedAt *time.Time
	// FetchedAt: The current time the HTML page was fetched.
	FetchedAt time.Time
}

// EmbeddedAsset represents an embedded asset within an HTML page.
type EmbeddedAsset struct {
	// URLPath: The original URL path of the asset.
	URLPath string
	// DataReader: The io.ReadCloser interface provides methods to read and close
	// the asset's data.
	DataReader io.Reader
}

// FetchResult represents the outcome of fetching an HTML page.
type FetchResult struct {
	// Web page URL
	URL string
	// Metadata extracted from the HTML page.
	Metadata *Metadata
	// HTTP response received from the fetch request.
	Response *http.Response
	// Fetch error if any.
	Err error
}
