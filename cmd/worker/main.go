package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/evolvedevlab/weaveset/config"
	"github.com/evolvedevlab/weaveset/internal"
	"github.com/evolvedevlab/weaveset/internal/queue"
	"github.com/evolvedevlab/weaveset/internal/scraper"
	"github.com/evolvedevlab/weaveset/internal/store"
	"github.com/evolvedevlab/weaveset/util"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func main() {
	quitch := make(chan os.Signal, 1)
	signal.Notify(quitch, os.Interrupt, syscall.SIGTERM)

	godotenv.Load()
	var (
		isProd    = util.GetEnv("ENVIRONMENT") == "production"
		hostname  = util.GetEnv("HOSTNAME")
		redisAddr = util.GetEnv("REDIS_ADDR", "127.0.0.1:6379")
		redisPass = util.GetEnv("REDIS_PASSWORD")

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

	<-quitch
	fmt.Println("shutting down in 3secs...")
	time.Sleep(time.Second * 3)
}

// WARN: shouldn't be used in HA setup
// kept for reference
func rebuildHugoLoop(dirPath string) {
	ticker := time.NewTicker(time.Second * 30)
	defer ticker.Stop()

	filepath := filepath.Join(dirPath, config.TriggerModifyFilename)

	var lastModAt int64
	for range ticker.C {
		info, err := os.Stat(filepath)
		if err == nil {
			mod := info.ModTime().Unix()
			if mod > lastModAt {
				log.Println("changes detected → rebuilding")

				cmd := exec.Command("hugo", "-s", "site", "--minify")
				if err := cmd.Run(); err != nil {
					slog.Error("rebuild error", "err", err)
					continue
				}

				lastModAt = mod
			}
		}
	}
}
