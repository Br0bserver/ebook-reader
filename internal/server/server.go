package server

import (
	"ebook-reader/internal/cache"
	"ebook-reader/internal/downloader"
	"ebook-reader/internal/model"
	"ebook-reader/internal/parser"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Server HTTP 服务
type Server struct {
	dl     *downloader.Downloader
	cache  *cache.Cache
	static fs.FS
}

// New 创建服务实例
func New(dl *downloader.Downloader, c *cache.Cache, static fs.FS) *Server {
	return &Server{dl: dl, cache: c, static: static}
}

// Handler 返回注册好路由的 http.Handler
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/book/meta", s.handleMeta)
	mux.HandleFunc("/api/book/chapter/", s.handleChapter)
	mux.HandleFunc("/api/book/cover/", s.handleCover)
	// /api/book/resource/{hash}/{path...}
	mux.HandleFunc("/api/book/resource/", s.handleResource)
	mux.Handle("/", http.FileServer(http.FS(s.static)))
	return mux
}

// resolveBook 下载 + 解析 + 缓存，返回 book 和对应的 parser
func (s *Server) resolveBook(fileURL string) (*model.Book, parser.Parser, error) {
	hash := downloader.URLHash(fileURL)

	// 内存缓存命中
	if book, ok := s.cache.Get(hash); ok {
		p, err := parser.GetParser(book.Format)
		if err != nil {
			return nil, nil, err
		}
		return book, p, nil
	}

	// 下载
	filePath, cachePath, err := s.dl.Download(fileURL)
	if err != nil {
		return nil, nil, fmt.Errorf("download: %w", err)
	}

	// 解析
	p, err := parser.GetParser(filePath)
	if err != nil {
		return nil, nil, err
	}

	book, err := p.Parse(filePath, cachePath)
	if err != nil {
		return nil, nil, fmt.Errorf("parse: %w", err)
	}

	book.ID = hash
	if book.CoverFilePath != "" {
		book.CoverURL = "/api/book/cover/" + hash
	}

	// 存入内存缓存
	s.cache.Put(hash, book)

	return book, p, nil
}

func (s *Server) handleMeta(w http.ResponseWriter, r *http.Request) {
	fileURL := r.URL.Query().Get("file")
	if fileURL == "" {
		http.Error(w, `{"error":"missing file parameter"}`, http.StatusBadRequest)
		return
	}

	book, _, err := s.resolveBook(fileURL)
	if err != nil {
		log.Printf("resolveBook error: %v", err)
		http.Error(w, `{"error":"failed to load book"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(book)
}

func (s *Server) handleChapter(w http.ResponseWriter, r *http.Request) {
	fileURL := r.URL.Query().Get("file")
	if fileURL == "" {
		http.Error(w, `{"error":"missing file parameter"}`, http.StatusBadRequest)
		return
	}

	// /api/book/chapter/3
	idStr := strings.TrimPrefix(r.URL.Path, "/api/book/chapter/")
	chapterID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid chapter id"}`, http.StatusBadRequest)
		return
	}

	book, p, err := s.resolveBook(fileURL)
	if err != nil {
		log.Printf("resolveBook error: %v", err)
		http.Error(w, `{"error":"failed to load book"}`, http.StatusInternalServerError)
		return
	}

	content, err := p.ReadChapter(book, chapterID, book.ID)
	if err != nil {
		log.Printf("readChapter error: %v", err)
		http.Error(w, `{"error":"failed to read chapter"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"content": content})
}

func (s *Server) handleCover(w http.ResponseWriter, r *http.Request) {
	// /api/book/cover/{hash}
	hash := strings.TrimPrefix(r.URL.Path, "/api/book/cover/")
	if hash == "" {
		http.NotFound(w, r)
		return
	}

	book, ok := s.cache.Get(hash)
	if !ok {
		http.NotFound(w, r)
		return
	}

	if book.CoverFilePath == "" {
		http.NotFound(w, r)
		return
	}

	http.ServeFile(w, r, book.CoverFilePath)
}

func (s *Server) handleResource(w http.ResponseWriter, r *http.Request) {
	// /api/book/resource/{hash}/{path...}
	rest := strings.TrimPrefix(r.URL.Path, "/api/book/resource/")
	slashIdx := strings.Index(rest, "/")
	if slashIdx < 0 {
		http.NotFound(w, r)
		return
	}
	hash := rest[:slashIdx]
	resPath := rest[slashIdx+1:]
	if hash == "" || resPath == "" {
		http.NotFound(w, r)
		return
	}

	// 从缓存中查找 book
	book, ok := s.cache.Get(hash)
	if !ok {
		http.NotFound(w, r)
		return
	}

	// 安全检查：防止路径穿越
	fullPath := filepath.Join(book.CachePath, resPath)
	if !strings.HasPrefix(filepath.Clean(fullPath), filepath.Clean(book.CachePath)+string(os.PathSeparator)) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	ext := filepath.Ext(fullPath)
	ct := mime.TypeByExtension(ext)
	if ct != "" {
		w.Header().Set("Content-Type", ct)
	}

	http.ServeFile(w, r, fullPath)
}

// findCover 在缓存目录中查找封面图片
func findCover(cachePath string) string {
	var found string
	filepath.WalkDir(cachePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := strings.ToLower(d.Name())
		if strings.Contains(name, "cover") {
			ext := filepath.Ext(name)
			if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".webp" {
				found = path
				return filepath.SkipAll
			}
		}
		return nil
	})
	return found
}
