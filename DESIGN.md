# Design

## Scope and Objective

Develop a command-line tool that fetches web pages and stores them (along with their assets and metadata) on disk for subsequent retrieval or browsing.

## Performance, Security, and Reliability

-   Efficiently handle concurrent requests to ensure timely downloads.
-   Validate URLs and check HTTP status codes before initiating downloads.
-   Implement fault tolerance mechanisms to handle network errors, file system errors, and other unexpected situations.

## Assumptions and Constraints

-   Store HTML, assets, and metadata files in the current working directory.

## Acceptance Criteria and Success Metrics

-   Fetch and save HTML, assets, and metadata files for the provided URL.
-   Print metadata to the console upon request.
-   Appropriately handle exceptions and display error messages to the console.
-   Local mirrors should load correctly in browsers, with all assets accessible.

## High-Level Design

### Main Components and Responsibilities

1.  Fetcher: Responsible for scraping HTML pages and downloading embedded assets for specified URLs.
2.  Parser: Parses HTML responses to extract metadata and modify downloaded asset URLs accordingly.
3.  Store: Manages the persistence of HTML, assets, and metadata files on disk.

## Detail Design

### Fetcher
```go
// Fetcher provides scraper instances for fetching web pages.
type Fetcher struct {
	// Callback functions to be called upon fetching done.
	callbacks []*FetchedCallback
	...
}

// Fetch starts scraping by HTTP requesting to the specified URL.
// Fetching result will be notified by callback functions if registered.
func (f *Fetcher) Fetch(url string) error {...}

// FetchedCallback is a type alias for the callback function executed upon
// task completion.
type FetchedCallback func(result *types.FetchResult)

// OnFetched registers a callback function to be invoked after each task finishes.
// NB this function is not thread safe.
func (f *Fetcher) OnFetched(cb FetchedCallback) {...}
```
### Parser
```go
// Parser parses an HTML response body to extract metadata or update selectors.  
type Parser struct { 
	// Represents the parsed jQuery like HTML document. 
	document *goquery.Document 
}

// ExtractMetadata extracts metadata from the document.
func (p *Parser) ExtractMetadata() (*Metadata, error) {...}

// URLTransformer is a function type that transforms URLs.
type URLTransformer func(string) (string, bool)

// ReplaceAssets replaces assets within the HTML document using the provided transformation
// function. These assets that are linked or embedded in the document, such as images, CSS files,
// JavaScript files, fonts, etc., are essential for rendering the web page correctly and
// providing the desired functionality and appearance.
func (p *Parser) ReplaceAssets(transformer URLTransformer) {...}
```
### FileStore
```go
// FileStore manages the storage of scraped HTML documents, downloaded asset 
// files, and parsed metadata in a designated directory.
type FileStore struct {
	// Base directory for storing scraped data. 
	rootDir string  
	// Document base name for generating file or folder names.
	docName string
}

// SaveDoc saves HTML document object.
func (fs *FileStore) SaveDoc(doc *goquery.Document) error {...}

// SaveMetadata saves the parsed metadata.
func (fs *FileStore) SaveMetadata(metadata *types.Metadata) error {...}

// LoadMetadata loads metadata from json file.
func (fs *FileStore) LoadMetadata() (*types.Metadata, error)

// SaveAsset saves embedded asset files.
func (fs *FileStore) SaveAsset(as *types.EmbeddedAsset) error {...}
```
### Auxillary Types
```go
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
	// AbsURL: The absolute URL path of the asset.
	AbsURL *url.URL
	// DataReader: The io.ReadCloser interface provides methods to read
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
```