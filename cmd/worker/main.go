package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/evolvedevlab/weavedeck/config"
	"github.com/evolvedevlab/weavedeck/internal"
	"github.com/evolvedevlab/weavedeck/internal/queue"
	"github.com/evolvedevlab/weavedeck/internal/scraper"
	"github.com/evolvedevlab/weavedeck/internal/store"
	"github.com/evolvedevlab/weavedeck/util"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

func main() {
	quitch := make(chan os.Signal, 1)
	signal.Notify(quitch, os.Interrupt, syscall.SIGTERM)

	godotenv.Load()
	var (
		isProd            = util.GetEnv("ENVIRONMENT") == "production"
		metricsListenAddr = util.GetEnv("METRICS_LISTEN_ADDR", ":2122")
		hostname          = util.GetEnv("HOSTNAME")
		redisAddr         = util.GetEnv("REDIS_ADDR", "127.0.0.1:6379")
		redisPass         = util.GetEnv("REDIS_PASSWORD")

		contentDirPath = util.GetEnv("CONTENT_DIR_PATH", "site/content/list")
	)
	if len(hostname) == 0 {
		log.Fatal("HOSTNAME variable not provided")
	}

	l := internal.NewLogger(isProd)
	slog.SetDefault(l)

	ctx := context.Background()
	rc := redis.NewClient(&redis.Options{
		Addr:       redisAddr,
		Password:   redisPass,
		DB:         0,
		ClientName: "worker",
	})
	defer rc.Close()

	if err := rc.Ping(ctx).Err(); err != nil {
		log.Fatal("redis ping error:", err)
	}

	// security group
	// its noop if already created, will return an error of BUSYGROUP
	err := rc.XGroupCreateMkStream(ctx, config.Stream, config.Group, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		log.Fatal(err)
	}

	q := queue.NewRedisQueue(hostname, config.Stream, config.Group, rc)

	fsStore, err := store.NewFileSystem(contentDirPath, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer fsStore.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	log.Println("Consume loop started:", hostname)
	go func() {
		if err := q.Consume(ctx, scraper.NewHandler(fsStore)); err != nil {
			log.Println("consume error:", err)
		}
	}()
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServe(metricsListenAddr, nil); err != nil {
			close(quitch)
			log.Println("metrics serve error:", err)
		}
	}()

	<-quitch
	fmt.Println("shutting down in 3secs...")
	time.Sleep(time.Second * 3)
}
