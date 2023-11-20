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
2.  Parser: Parses HTML responses to extract metadata, identify embedded assets, and modify local URLs accordingly.
3.  Store: Manages the persistence of HTML, assets, and metadata files on disk.

## Detail Design

### PageFetcher
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
type FetchedCallback func(result *FetchResult, err error)
// OnFetched registers a callback function to be invoked after each task finishes.
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
type URLTransformer func(string) string
// ReplaceURLs replaces URLs within the HTML document using the provided 
// transformation function.
func (p *Parser) ReplaceURLs(t URLTransformer) error {...}
```
### FileStore
```go
// FileStore manages the storage of scraped HTML documents, downloaded asset 
// files, and parsed metadata in a designated directory.
type FileStore struct {
	// Base directory for storing scraped data. 
	rootDir string  
	// Document name for generating file or folder names.
	docName string
}

// SaveHTMLDoc saves the HTML document object to `${rootDir}/${docName}.html`.  
func (fs *FileStore) SaveHTMLDoc(doc *goquery.Document) error {...}

// SaveMetadata saves the parsed metadata to `${rootDir}/${docName}_meta.json`.  
func (fs *FileStore) SaveMetadata(meta *Metadata) error {...}

// SaveAsset saves the asset file to `${rootDir}/${docName}_assets/${assetFileName}`.  
func (fs *FileStore) SaveAsset(as *EmbeddedAsset) error {...}
```
### Auxillary Types
```go
// EmbeddedAsset represents an embedded asset within an HTML page.
type EmbeddedAsset struct {
    // URLPath: The original URL path of the asset.
    URLPath string
    // DataReader: The io.ReadCloser interface provides methods to read and close 
    // the asset's data.
    DataReader io.ReadCloser
}

// FetchResult represents the outcome of fetching an HTML page.
type FetchResult struct {
    // Metadata extracted from the HTML page.
    Metadata *Metadata
    // HTTP response received from the fetch request.
    Response *http.Response
}

// Metadata describes the structure and information of an HTML page.
type Metadata struct {
    // NumLinks: The total number of links found within the HTML page.
    NumLinks uint
    // NumImages: The total number of images found within the HTML page.
    NumImages uint
    // LastFetchedAt: The last time the HTML page was fetched.
    LastFetchedAt *time.Time
    // FetchedAt: The current time the HTML page was fetched.
    FetchedAt *time.Time
}
```