package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/evolvedevlab/weaveset/data"
	"github.com/evolvedevlab/weaveset/util"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func main() {
	godotenv.Load()
	var (
		listenAddr = util.GetEnv("LISTEN_ADDR", ":3000")
		hostname   = util.GetEnv("HOSTNAME")

		redisAddr = util.GetEnv("REDIS_ADDR", "127.0.0.1:6379")
		redisPass = util.GetEnv("REDIS_PASSWORD")
	)

	rc := redis.NewClient(&redis.Options{
		Addr:       redisAddr,
		Password:   redisPass,
		DB:         0,
		ClientName: "apiserver",
	})

	ctx := context.Background()
	if err := rc.Ping(ctx).Err(); err != nil {
		log.Fatal("redis ping:", err)
	}

	// security group
	// its noop if already created, will return an error of BUSYGROUP
	err := rc.XGroupCreateMkStream(ctx, "jobs", "workers", "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		log.Fatal(err)
	}

	q := data.NewRedisQueue(hostname, "jobs", "workers", rc)

	http.HandleFunc("/health", handleGetHealth)
	http.HandleFunc("/job", handlePostJob(q))

	log.Printf("started at %s\n", listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}

func handlePostJob(q data.Queuer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		url := r.URL.Query().Get("url")
		if len(url) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("invalid url"))
			return
		}

		err := q.Enqueue(r.Context(), &data.Job{
			ID:        uuid.New().String(),
			URL:       url,
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
