package apiserver

import (
	"net/http"
	"net/url"
	"time"

	"github.com/evolvedevlab/weaveset/data"
	"github.com/evolvedevlab/weaveset/internal/queue"
	"github.com/google/uuid"
)

func handlePostJob(q queue.Queuer) ApiHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		u, err := url.Parse(r.URL.Query().Get("url"))
		if err != nil || u.Scheme == "" || u.Host == "" {
			return NewBadRequestError("invalid URL", nil)
		}

		err = q.Enqueue(r.Context(), &data.Job{
			ID:        uuid.New().String(),
			URL:       u.String(),
			CreatedAt: time.Now(),
		})
		if err != nil {
			return err
		}

		return writeJSON(w, http.StatusOK, "task queued!")
	}
}

func handleGetHealth(w http.ResponseWriter, r *http.Request) error {
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("OK!"))
	return err
}
