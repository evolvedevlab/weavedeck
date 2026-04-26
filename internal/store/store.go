package store

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"

	"github.com/evolvedevlab/weaveset/data"
	"github.com/evolvedevlab/weaveset/util"
)

type FilepathGeneratorFunc func(*data.List) string

var DefaultFilepathGeneratorFunc = func(dirPath string) FilepathGeneratorFunc {
	return func(list *data.List) string {
		filename := fmt.Sprintf("%s-%s.md", util.GenerateSlug(list.Name, "-"), list.ID)
		return filepath.Join(dirPath, filename)
	}
}

type Storer interface {
	Save(*data.List) error
	Delete(string) error
}

type ReadWriteSeekTruncater interface {
	io.ReadWriteSeeker
	io.Closer
	Truncate(size int64) error
}

type noopRWSeekTruncate struct {
	*bytes.Buffer
}

func (noop *noopRWSeekTruncate) Truncate(int64) error {
	return nil
}

func (noop *noopRWSeekTruncate) Seek(int64, int) (int64, error) {
	return 0, nil
}

func (noop *noopRWSeekTruncate) Close() error {
	return nil
}
