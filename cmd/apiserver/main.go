package main

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/evolvedevlab/weaveset/config"
	"github.com/evolvedevlab/weaveset/data"
	"github.com/evolvedevlab/weaveset/util"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func main() {
	quitch := make(chan os.Signal, 1)
	signal.Notify(quitch, os.Interrupt, syscall.SIGTERM)

	godotenv.Load()
	var (
		listenAddr    = util.GetEnv("LISTEN_ADDR", ":3000")
		hostname      = util.GetEnv("HOSTNAME")
		publicDirPath = util.GetEnv("PUBLIC_DIR_PATH", "site/public")

		redisAddr = util.GetEnv("REDIS_ADDR", "127.0.0.1:6379")
		redisPass = util.GetEnv("REDIS_PASSWORD")
	)
	if len(hostname) == 0 {
		log.Fatal("HOSTNAME variable not provided")
	}

	rc := redis.NewClient(&redis.Options{
		Addr:       redisAddr,
		Password:   redisPass,
		DB:         0,
		ClientName: "apiserver",
	})
	defer rc.Close()

	if err := rc.Ping(context.Background()).Err(); err != nil {
		log.Fatal("redis ping:", err)
	}

	q := data.NewRedisQueue(hostname, config.Stream, config.Group, rc)

	// routes
	http.Handle("/", http.FileServer(http.Dir(publicDirPath)))
	http.HandleFunc("/health", handleGetHealth)
	http.HandleFunc("/job", handlePostJob(q))

	log.Printf("started at %s\n", listenAddr)
	go func() {
		if err := http.ListenAndServe(listenAddr, nil); err != nil {
			log.Println("http serve error:", err)
		}
	}()

	<-quitch
	log.Println("shutting down...")
}

func handlePostJob(q data.Queuer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := url.Parse(r.URL.Query().Get("url"))
		if err != nil || u.Scheme == "" || u.Host == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("invalid url"))
			return
		}

		err = q.Enqueue(r.Context(), &data.Job{
			ID:        uuid.New().String(),
			URL:       u.String(),
			CreatedAt: time.Now(),
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("task queued!"))
	}
}

func handleGetHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK!"))
}
