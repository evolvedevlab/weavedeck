package apiserver

import (
	"log"
	"net/http"

	"github.com/evolvedevlab/weaveset/internal/queue"
)

type ApiServer struct {
	listenAddr    string
	publicDirPath string

	queue queue.Queuer
}

func New(listenAddr string, publicDirPath string, queue queue.Queuer) *ApiServer {
	return &ApiServer{
		listenAddr:    listenAddr,
		publicDirPath: publicDirPath,
		queue:         queue,
	}
}

func (s *ApiServer) Start() error {
	// endpoints
	http.Handle("/", http.FileServer(http.Dir(s.publicDirPath)))
	http.HandleFunc("/health", handler(handleGetHealth))
	http.HandleFunc("/job", handler(handlePostJob(s.queue)))

	log.Printf("started at %s\n", s.listenAddr)
	return http.ListenAndServe(s.listenAddr, nil)
}
