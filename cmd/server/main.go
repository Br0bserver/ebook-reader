package main

import (
	"ebook-reader/internal/cache"
	"ebook-reader/internal/downloader"
	"ebook-reader/internal/server"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"ebook-reader/static"
)

func main() {
	port := flag.Int("p", 8080, "listen port")
	dataDir := flag.String("d", "data", "data directory for cache")
	ttl := flag.Duration("ttl", 24*time.Hour, "cache TTL duration")
	flag.Parse()

	// 确保数据目录存在
	if err := os.MkdirAll(*dataDir, 0755); err != nil {
		log.Fatalf("create data dir: %v", err)
	}

	dl := downloader.New(*dataDir)
	c := cache.New(*dataDir, *ttl)

	// 获取嵌入的静态文件
	staticFS, err := fs.Sub(static.StaticFS, "dist")
	if err != nil {
		log.Fatalf("static fs: %v", err)
	}

	srv := server.New(dl, c, staticFS)

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("ebook-reader listening on %s", addr)
	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		log.Fatalf("server: %v", err)
	}
}
