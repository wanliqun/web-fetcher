package parser

import (
	"io"

	"github.com/PuerkitoBio/goquery"
	"github.com/pkg/errors"
	"github.com/wanliqun/web-fetcher/types"
)

var (
	assetUrlAttrSelectors = "img[src], link[rel=stylesheet], script[src]"
)

// Parser parses an HTML response body to extract metadata or update selectors.
type Parser struct {
	// Represents the parsed jQuery like HTML document.
	Document *goquery.Document
}

func NewParser(dataReader io.Reader) (*Parser, error) {
	doc, err := goquery.NewDocumentFromReader(dataReader)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create goquery document")
	}

	return &Parser{Document: doc}, nil
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
type URLTransformer func(string) (string, bool)

// ReplaceAssets replaces assets within the HTML document using the provided transformation
// function. These assets that are linked or embedded in the document, such as images, CSS files,
// JavaScript files, fonts, etc., are essential for rendering the web page correctly and
// providing the desired functionality and appearance.
func (p *Parser) ReplaceAssets(transformer URLTransformer) {
	// Find all the elements that have an attribute with a URL value
	p.Document.Find(assetUrlAttrSelectors).Each(func(i int, s *goquery.Selection) {
		var urlAttrKey string
		switch {
		case s.Is("img"), s.Is("script"):
			urlAttrKey = "src"
		case s.Is("link"):
			urlAttrKey = "href"
		}

		assetURL, ok := s.Attr(urlAttrKey)
		if !ok || len(assetURL) == 0 {
			return
		}

		if newAssetURL, ok := transformer(assetURL); ok {
			// Replace selection URL
			s.SetAttr(urlAttrKey, newAssetURL)
		}
	})
}
