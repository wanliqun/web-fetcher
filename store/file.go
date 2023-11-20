package store

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"github.com/PuerkitoBio/goquery"
	"github.com/pkg/errors"
	"github.com/wanliqun/web-fetcher/types"
)

// FileStore manages the storage of scraped HTML documents, downloaded asset
// files, and parsed metadata in a designated directory.
type FileStore struct {
	// Base directory for storing scraped data.
	rootDir string
	// Document base name for generating file or folder names.
	docName string
}

func NewFileStore(rootDir, docName string) *FileStore {
	return &FileStore{
		rootDir: rootDir,
		docName: docName,
	}
}

// SaveHTMLDoc saves HTML document object.
func (fs *FileStore) SaveHTMLDoc(doc *goquery.Document) error {
	content, err := doc.Html()
	if err != nil {
		return errors.WithMessage(err, "invalid HTML document")
	}

	return os.WriteFile(fs.htmlDocPath(), []byte(content), 0644)
}

// HTML doc file path: `${rootDir}/${docName}.html`.
func (fs *FileStore) htmlDocPath() string {
	return filepath.Join(fs.rootDir, fs.docName+".html")
}

// SaveMetadata saves the parsed metadata to `${rootDir}/${docName}.json`.
func (fs *FileStore) SaveMetadata(metadata *types.Metadata) error {
	content, err := json.Marshal(metadata)
	if err != nil {
		return errors.WithMessage(err, "JSON marshal error")
	}

	return os.WriteFile(fs.metadataFilePath(), []byte(content), 0644)
}

// LoadMetadata loads metadata from json file.
func (fs *FileStore) LoadMetadata(docName string) (*types.Metadata, error) {
	data, err := os.ReadFile(fs.metadataFilePath())
	if err != nil {
		if os.IsNotExist(err) { // file not found
			return nil, nil
		}

		return nil, errors.WithMessage(err, "failed to read file")
	}

	var result types.Metadata
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, errors.WithMessage(err, "JSON unmarshal error")
	}

	return &result, nil
}

// Metadata file path format: `${rootDir}/${docName}.json`
func (fs *FileStore) metadataFilePath() string {
	return filepath.Join(fs.rootDir, fs.docName+".json")
}

// SaveAsset saves embedded asset files.
func (fs *FileStore) SaveAsset(as *types.EmbeddedAsset) error {
	file, err := os.Create(fs.assetFilePath(as))
	if err != nil {
		return errors.WithMessage(err, "failed to create file")
	}
	defer file.Close()

	if _, err = io.Copy(file, as.DataReader); err != nil {
		return errors.WithMessage(err, "failed to write file")
	}

	return nil
}

// Asset file path format:  `${rootDir}/${docName}/${assetFileName}`.
func (fs *FileStore) assetFilePath(as *types.EmbeddedAsset) string {
	fileName := filepath.Base(as.URLPath)
	return filepath.Join(fs.rootDir, fs.docName, fileName)
}
