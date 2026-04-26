package store

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/evolvedevlab/weaveset/config"
	"github.com/evolvedevlab/weaveset/data"
)

type FileSystem struct {
	// file to trigger modify
	file ReadWriteSeekTruncater

	filepathGenerator FilepathGeneratorFunc
	dirPath           string
}

func NewFileSystem(dirPath string, fn FilepathGeneratorFunc) (*FileSystem, error) {
	if fn == nil {
		fn = DefaultFilepathGeneratorFunc(dirPath)
	}

	file, err := os.OpenFile(filepath.Join(dirPath, config.TriggerModifyFilename), os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	return &FileSystem{
		dirPath:           dirPath,
		filepathGenerator: fn,
		file:              file,
	}, nil
}

func (s *FileSystem) Save(list *data.List) error {
	filepath := s.filepathGenerator(list)

	file, err := os.Create(fmt.Sprintf("%s.%d.tmp", filepath, time.Now().UnixNano()))
	if err != nil {
		return err
	}
	defer file.Close()

	if err := s.writeContent(file, list); err != nil {
		return err
	}
	if err := os.Rename(file.Name(), filepath); err != nil {
		return err
	}

	return s.triggerModify()
}

func (s *FileSystem) Delete(slug string) error {
	fullpath := filepath.Join(s.dirPath, slug) + ".md"
	return os.Remove(fullpath)
}

func (s *FileSystem) Close() error {
	return s.file.Close()
}

// writeContent will write markdown expected by hugo
// WARN: it follows the generic markdown syntax except for images
func (s *FileSystem) writeContent(w io.Writer, list *data.List) error {
	// frontmatter
	fmt.Fprintf(w, "---\n")
	fmt.Fprintf(w, "title: %q\n", list.Name)
	fmt.Fprintf(w, "description: %q\n", list.Description)
	fmt.Fprintf(w, "date: %s\n", list.CreatedAt.Format(time.RFC3339))

	if len(list.Items) > 0 && len(list.Items[0].Images) > 0 {
		fmt.Fprintf(w, "image: %s\n", list.Items[0].Images[0])
	}

	// fmt.Fprintf(w, "tags:\n  - dark\n") // TODO: do this
	fmt.Fprintf(w, "categories:\n  - %s\n", list.ListType())
	fmt.Fprintf(w, "draft: false\n")
	fmt.Fprintf(w, "toc: true\n")
	fmt.Fprintf(w, "---\n\n")

	// body
	for _, item := range list.Items {
		if item.Position > 0 {
			fmt.Fprintf(w, "## %d. %s\n\n", item.Position, item.Title)
		} else {
			fmt.Fprintf(w, "## %s\n\n", item.Title)
		}

		if len(item.By) > 0 {
			fmt.Fprintf(w, "By %s ㆍ \n", strings.Join(item.By, ", "))
		}

		if item.AvgRating > 0 {
			fmt.Fprintf(w, "⭐ %.2f (%d ratings)\n",
				item.AvgRating,
				item.TotalRatings,
			)
		}
		fmt.Fprintf(w, "\n")

		if len(item.Images) > 0 {
			fmt.Fprintf(w,
				"![%s](%s \"{width='220px' height='320px'}\")\n\n",
				item.Title,
				item.Images[0],
			)
		}
		fmt.Fprintf(w,
			"[View Source](https://www.goodreads.com/book/show/%s)\n\n",
			item.ID,
		)

		// Divider
		fmt.Fprintf(w, "---\n\n")
	}

	return nil
}

// triggerModify will overwrite the file contents
func (s *FileSystem) triggerModify() error {
	if _, err := s.file.Seek(0, 0); err != nil {
		return err
	}
	if err := s.file.Truncate(0); err != nil {
		return err
	}

	_, err := s.file.Write([]byte(time.Now().String()))
	return err
}
