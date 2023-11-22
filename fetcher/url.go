package fetcher

import (
	"net/url"

	"github.com/wanliqun/web-fetcher/parser"
)

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

// determineBaseURL determines a final base URL from the HTML document and embedding webpage URL.
// There are different types of relative URLs, like root-relative (such as `/path/a.html`),
// document-relative (such as `./path/a.html` or `path/a.html`), and protocol-relative path.
// The syntax and the meaning of the relative URL depend on the type and the base URL.
func determineBaseURL(urlObj *url.URL, domParser *parser.Parser) (baseUrlObj *url.URL) {
	if baseUrl, ok := domParser.Document.Find("base").Attr("href"); ok {
		if tmpUrlObj, err := url.Parse(baseUrl); err == nil {
			// Base URL can also be a relative path to the embedding page URL.
			baseUrlObj = urlObj.ResolveReference(tmpUrlObj)
		}
	}

	// Derive the base URL from embedding page URL if not set properly before.
	if baseUrlObj == nil {
		baseUrlObj = urlObj.ResolveReference(&url.URL{Path: "."})
	}

	return baseUrlObj
}
