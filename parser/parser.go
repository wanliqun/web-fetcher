package parser

import (
	"slices"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/wanliqun/web-fetcher/types"
)

var (
	urlAttributeKeys      = []string{"src", "href", "data"}
	urlAttributeSelectors = "[src], [href], [data]"
)

// Parser parses an HTML response body to extract metadata or update selectors.
type Parser struct {
	// Represents the parsed jQuery like HTML document.
	Document *goquery.Document
}

// ExtractMetadata extracts metadata from the document.
func (p *Parser) ExtractMetadata() *types.Metadata {
	// Currently, we only count the number of links and images in the document.
	// Further analysis of other content elements may be considered in
	// future enhancements.
	return &types.Metadata{
		NumLinks:  p.Document.Find("a").Length(),
		NumImages: p.Document.Find("img").Length(),
	}
}

// URLTransformer is a function type that transforms URLs.
type URLTransformer func(string) string

// ReplaceURLs replaces URLs within the HTML document using the provided
// transformation function.
func (p *Parser) ReplaceURLs(transformer URLTransformer) {
	// Find all the elements that have an attribute with a URL value
	p.Document.Find(urlAttributeSelectors).Each(func(i int, s *goquery.Selection) {
		for _, a := range s.Nodes[0].Attr {
			if slices.Contains(urlAttributeKeys, strings.ToLower(a.Key)) {
				s.SetAttr(a.Key, transformer(a.Val))
			}
		}
	})
}
