package store

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/evolvedevlab/weaveset/data"
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

	err := fs.Save(list)
	a.NoError(err)

	expectedPath := fs.filepathGenerator(list)
	buf := new(bytes.Buffer)

	err = fs.writeContent(buf, list)
	a.NoError(err)

	data, err := os.ReadFile(expectedPath)
	a.NoError(err)

	a.NotEmpty(data)
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
