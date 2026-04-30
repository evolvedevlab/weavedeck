package store

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/evolvedevlab/weavedeck/data"
	"github.com/stretchr/testify/assert"
)

func TestStore_Save(t *testing.T) {
	a := assert.New(t)

	dir := t.TempDir()
	list := &data.List{
		ID:        "1",
		Name:      "test 1",
		CreatedAt: time.Now(),
	}

	fs := &FileSystem{
		dirPath: dir,
		file: &noopRWSeekTruncate{
			Buffer: new(bytes.Buffer),
		},
		filepathGenerator: DefaultFilepathGeneratorFunc(dir),
	}

	// save
	err := fs.Save(list)
	a.NoError(err)

	expectedPath := fs.filepathGenerator(list)

	data, err := os.ReadFile(expectedPath)
	a.NoError(err)

	a.NotEmpty(data)
}

func TestStore_Delete(t *testing.T) {
	a := assert.New(t)

	dir := t.TempDir()
	list := &data.List{
		ID:        "1",
		Name:      "test 1",
		CreatedAt: time.Now(),
	}

	fs := &FileSystem{
		dirPath: dir,
		file: &noopRWSeekTruncate{
			Buffer: new(bytes.Buffer),
		},
		filepathGenerator: DefaultFilepathGeneratorFunc(dir),
	}

	// save
	err := fs.Save(list)
	a.NoError(err)

	path := fs.filepathGenerator(list)

	// ensure file exists
	_, err = os.Stat(path)
	a.NoError(err)

	// delete
	slug := strings.TrimSuffix(filepath.Base(path), ".md")

	err = fs.Delete(slug)
	a.NoError(err)

	// confirm delete
	_, err = os.Stat(path)
	a.Error(err)
	a.True(os.IsNotExist(err))
}

func TestStore_WriteContent(t *testing.T) {
	a := assert.New(t)

	buf := new(bytes.Buffer)
	list := &data.List{
		ID:        "1",
		Name:      "test 1",
		CreatedAt: time.Now(),
	}

	fs := &FileSystem{}

	err := fs.writeContent(buf, list)
	a.NoError(err)

	a.Contains(buf.String(), "test 1")
}
