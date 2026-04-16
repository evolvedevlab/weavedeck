package main

import (
	"context"
	"log"

	"github.com/evolvedevlab/weaveset/data"
	"github.com/evolvedevlab/weaveset/scraper"
	"github.com/evolvedevlab/weaveset/util"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func main() {
	godotenv.Load()
	var (
		hostname  = util.GetEnv("HOSTNAME")
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
		ClientName: "worker",
	})

	ctx := context.Background()
	if err := rc.Ping(ctx).Err(); err != nil {
		log.Fatal("redis ping error:", err)
	}

	q := data.NewRedisQueue(hostname, "jobs", "workers", rc)
	// q.Enqueue(ctx, &data.Job{
	// 	ID:        uuid.New().String(),
	// 	URL:       "https://www.goodreads.com/list/show/399714",
	// 	CreatedAt: time.Now(),
	// })

	log.Println("Consume loop started...")
	q.Consume(ctx, scraper.NewHandler(nil))
}
