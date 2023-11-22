package store

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/PuerkitoBio/goquery"
	"github.com/kennygrant/sanitize"
	"github.com/pkg/errors"
	"github.com/wanliqun/web-fetcher/types"
)

var (
	defaultFileStoreRootDir = "."
)

func init() {
	curDir, err := os.Getwd()
	if err != nil {
		panic("failed to get current directory")
	}

	defaultFileStoreRootDir = curDir
}

// FileStore manages the storage of scraped HTML documents, downloaded asset
// files, and parsed metadata in a designated directory.
type FileStore struct {
	// Base directory for storing scraped data.
	rootDir string
	// Document base name for generating file or folder.
	docName string
}

func NewFileStore(rootDir, docName string) (*FileStore, error) {
	if len(rootDir) == 0 {
		rootDir = defaultFileStoreRootDir
	}

	return &FileStore{
		rootDir: rootDir,
		docName: sanitize.BaseName(docName),
	}, nil
}

// SaveDoc saves HTML document object.
func (fs *FileStore) SaveDoc(doc *goquery.Document) error {
	content, err := doc.Html()
	if err != nil {
		return errors.WithMessage(err, "invalid HTML document")
	}

	return os.WriteFile(fs.HtmlDocPath(), []byte(content), 0644)
}

// Abosulte HTML document file format: `${rootDir}/${docName}.html`.
func (fs *FileStore) HtmlDocPath() string {
	return filepath.Join(fs.rootDir, fs.docName+".html")
}

// SaveMetadata saves the parsed metadata to `${rootDir}/${docName}.json`.
func (fs *FileStore) SaveMetadata(metadata *types.Metadata) error {
	content, err := json.Marshal(metadata)
	if err != nil {
		return errors.WithMessage(err, "JSON marshal error")
	}

	return os.WriteFile(fs.MetadataFilePath(), []byte(content), 0644)
}

// LoadMetadata loads metadata from json file.
func (fs *FileStore) LoadMetadata() (*types.Metadata, error) {
	data, err := os.ReadFile(fs.MetadataFilePath())
	if os.IsNotExist(err) { // file not found
		return nil, nil
	}

	if err != nil {
		return nil, errors.WithMessage(err, "failed to read file")
	}

	var result types.Metadata
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, errors.WithMessage(err, "JSON unmarshal error")
	}

	return &result, nil
}

// Metadata file path format: `${rootDir}/${docName}.json`
func (fs *FileStore) MetadataFilePath() string {
	return filepath.Join(fs.rootDir, fs.docName+".json")
}

// SaveAsset saves embedded asset files.
func (fs *FileStore) SaveAsset(as *types.EmbeddedAsset) error {
	assetFilePath := fs.AssetFilePath(as)
	if err := os.MkdirAll(filepath.Dir(assetFilePath), 0755); err != nil {
		return errors.WithMessage(err, "failed to create directory")
	}

	file, err := os.Create(assetFilePath)
	if err != nil {
		return errors.WithMessage(err, "failed to create file")
	}
	defer file.Close()

	if _, err = io.Copy(file, as.DataReader); err != nil {
		return errors.WithMessage(err, "failed to write file")
	}

	return nil
}

// Absolute asset file path format:
// `${rootDir}/${docName}/${assetFilePath}/${assetFileName}`.
func (fs *FileStore) AssetFilePath(as *types.EmbeddedAsset) string {
	return filepath.Join(fs.rootDir, fs.RelativeAssetFilePath(as))
}

// Relative asset file path format:
// `${docName}/${assetFilePath}/${assetFileName}`.
func (fs *FileStore) RelativeAssetFilePath(as *types.EmbeddedAsset) string {
	paths := []string{fs.docName}

	dir, file := path.Split(as.AbsURL.Path)
	if len(dir) > 0 {
		paths = append(paths, dir)
	}

	if len(as.AbsURL.RawQuery) > 0 {
		file = fmt.Sprintf("%v_%v", as.AbsURL.RawQuery, file)
	}

	if len(file) > 0 {
		paths = append(paths, sanitize.Name(file))
	}

	return filepath.Join(paths...)
}
