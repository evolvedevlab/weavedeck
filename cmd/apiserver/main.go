package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/evolvedevlab/weaveset/apiserver"
	"github.com/evolvedevlab/weaveset/config"
	"github.com/evolvedevlab/weaveset/internal"
	"github.com/evolvedevlab/weaveset/internal/queue"
	"github.com/evolvedevlab/weaveset/util"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func main() {
	quitch := make(chan os.Signal, 1)
	signal.Notify(quitch, os.Interrupt, syscall.SIGTERM)

	godotenv.Load()
	var (
		isProd        = util.GetEnv("ENVIRONMENT") == "production"
		listenAddr    = util.GetEnv("LISTEN_ADDR", ":3000")
		hostname      = util.GetEnv("HOSTNAME")
		publicDirPath = util.GetEnv("PUBLIC_DIR_PATH", "site/public")

		redisAddr = util.GetEnv("REDIS_ADDR", "127.0.0.1:6379")
		redisPass = util.GetEnv("REDIS_PASSWORD")
	)
	if len(hostname) == 0 {
		log.Fatal("HOSTNAME variable not provided")
	}

	l := internal.NewLogger(isProd)
	slog.SetDefault(l)

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

	q := queue.NewRedisQueue(hostname, config.Stream, config.Group, rc)
	s := apiserver.New(listenAddr, publicDirPath, q)

	go func() {
		if err := s.Start(); err != nil {
			log.Println("http serve error:", err)
		}
	}()

	<-quitch
	log.Println("shutting down...")
}
