package apiserver

import (
	"log"
	"net/http"

	"github.com/evolvedevlab/weavedeck/internal/queue"
	"github.com/evolvedevlab/weavedeck/internal/store"
)

type ApiServer struct {
	listenAddr    string
	publicDirPath string

	queue queue.Queuer
	store store.Storer
}

func New(listenAddr string, publicDirPath string,
	queue queue.Queuer, store store.Storer) *ApiServer {
	return &ApiServer{
		listenAddr:    listenAddr,
		publicDirPath: publicDirPath,
		queue:         queue,
		store:         store,
	}
}

func (s *ApiServer) Start() error {
	// endpoints
	http.Handle("/", http.FileServer(http.Dir(s.publicDirPath)))
	http.HandleFunc("/health", handler(handleGetHealth))
	http.HandleFunc("POST /job", handler(handlePostJob(s.queue)))
	http.HandleFunc("DELETE /list/{slug}", handler(handleDeleteList(s.store)))

	log.Printf("started at %s\n", s.listenAddr)
	return http.ListenAndServe(s.listenAddr, nil)
}
