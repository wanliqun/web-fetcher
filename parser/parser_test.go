package parser_test

import (
	"os"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/wanliqun/web-fetcher/parser"
)

const testHTMLString = `
<!DOCTYPE html>
<html>
<head>
<title>Test HTML Page</title>
</head>
<body>
<h1>Test Page</h1>
<p>This is a test page with links.</p>
<a href="https://www.google.com">Google</a>
<a href="https://www.wikipedia.org">Wikipedia</a>
<a href="https://www.youtube.com">YouTube</a>
<img src="https://upload.wikimedia.org/wikipedia/commons/a/a9/Example.png" alt="Example image">
<p>This is some more text with a link:</p>
<a href="https://www.example.com">Example website</a>
</body>
</html>
`

var parserT *parser.Parser

func setup() error {
	reader := strings.NewReader(testHTMLString)
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return err
	}

	parserT = &parser.Parser{Document: doc}
	return nil
}

func TestMain(m *testing.M) {
	if err := setup(); err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

func TestExtractMetadata(t *testing.T) {
	metadata := parserT.ExtractMetadata()
	assert.NotNil(t, metadata, "Extracted metadata should not be nil")

	assert.Equal(t, metadata.NumLinks, 4, "Expected 4 links, but found %d", metadata.NumLinks)
	assert.Equal(t, metadata.NumImages, 1, "Expected 1 image, but found %d", metadata.NumImages)
}

func TestReplaceURLs(t *testing.T) {
	expectedNewImageURL := "test.png"
	parserT.ReplaceURLs(func(originalURL string) string {
		return expectedNewImageURL
	})

	newImgUrl, found := parserT.Document.Find("img").Attr("src")
	assert.True(t, found, "Failed to find image element")
	assert.Equal(t, expectedNewImageURL, newImgUrl, "Image URL should be replaced with '%s', but found '%s'", expectedNewImageURL, newImgUrl)
}
