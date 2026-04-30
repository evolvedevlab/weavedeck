package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/evolvedevlab/weavedeck/data"
	"github.com/stretchr/testify/assert"
)

func Test_Filepathgenerator(t *testing.T) {
	a := assert.New(t)

	dir := "tesdata"
	list := &data.List{
		ID:        "1",
		Name:      "test 1",
		CreatedAt: time.Now(),
	}

	expectedPath := filepath.Join(dir, "test-1-1.md")
	filepath := DefaultFilepathGeneratorFunc(dir)(list)

	a.Equal(expectedPath, filepath)
}
