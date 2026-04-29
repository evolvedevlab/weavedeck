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
	"syscall"
	"time"

	"github.com/evolvedevlab/weaveset/apiserver"
	"github.com/evolvedevlab/weaveset/config"
	"github.com/evolvedevlab/weaveset/internal"
	"github.com/evolvedevlab/weaveset/internal/queue"
	"github.com/evolvedevlab/weaveset/internal/scraper"
	"github.com/evolvedevlab/weaveset/internal/store"
	"github.com/evolvedevlab/weaveset/util"
	"github.com/joho/godotenv"
)

func main() {
	quitch := make(chan os.Signal, 1)
	signal.Notify(quitch, os.Interrupt, syscall.SIGTERM)

	godotenv.Load()
	var (
		isProd     = util.GetEnv("ENVIRONMENT") == "production"
		listenAddr = util.GetEnv("LISTEN_ADDR", ":3000")

		contentDirPath = util.GetEnv("CONTENT_DIR_PATH", "site/content/list")
		publicDirPath  = util.GetEnv("PUBLIC_DIR_PATH", "site/public")
	)

	l := internal.NewLogger(isProd)
	slog.SetDefault(l)

	fsStore, err := store.NewFileSystem(contentDirPath, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer fsStore.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool := queue.NewWorkerPool(10, 100)

	log.Println("Consume loop started")
	go rebuildHugoLoop(contentDirPath, time.Millisecond*5)
	go func() {
		if err := pool.Consume(ctx, scraper.NewHandler(fsStore)); err != nil {
			close(quitch)
			log.Println("consume error:", err)
		}
	}()

	s := apiserver.New(listenAddr, publicDirPath, pool, fsStore)
	go func() {
		if err := s.Start(); err != nil {
			close(quitch)
			log.Println("api serve error:", err)
		}
	}()

	<-quitch
	fmt.Println("shutting down in 3secs...")
	time.Sleep(time.Second * 3)
}

func rebuildHugoLoop(dirPath string, dur time.Duration) {
	ticker := time.NewTicker(dur)
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
