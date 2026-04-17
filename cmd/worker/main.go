package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/evolvedevlab/weaveset/config"
	"github.com/evolvedevlab/weaveset/data"
	"github.com/evolvedevlab/weaveset/scraper"
	"github.com/evolvedevlab/weaveset/util"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func main() {
	quitch := make(chan os.Signal, 1)
	signal.Notify(quitch, os.Interrupt, syscall.SIGTERM)

	godotenv.Load()
	var (
		hostname  = util.GetEnv("HOSTNAME")
		redisAddr = util.GetEnv("REDIS_ADDR", "127.0.0.1:6379")
		redisPass = util.GetEnv("REDIS_PASSWORD")
	)
	if len(hostname) == 0 {
		log.Fatal("HOSTNAME variable not provided")
	}

	ctx := context.Background()
	rc := redis.NewClient(&redis.Options{
		Addr:       redisAddr,
		Password:   redisPass,
		DB:         0,
		ClientName: "worker",
	})

	// security group
	// its noop if already created, will return an error of BUSYGROUP
	err := rc.XGroupCreateMkStream(ctx, config.Stream, config.Group, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		log.Fatal(err)
	}

	if err := rc.Ping(ctx).Err(); err != nil {
		log.Fatal("redis ping error:", err)
	}

	q := data.NewRedisQueue(hostname, "jobs", "workers", rc)
	// q.Enqueue(ctx, &data.Job{
	// 	ID:        uuid.New().String(),
	// 	URL:       "https://www.goodreads.com/list/show/399714",
	// 	CreatedAt: time.Now(),
	// })

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	log.Println("Consume loop started...")
	go func() {
		if err := q.Consume(ctx, scraper.NewHandler(nil)); err != nil {
			log.Println("consume error:", err)
		}
	}()

	<-quitch
	fmt.Println("shutting down in 3secs...")
	time.Sleep(time.Second * 3)
}
