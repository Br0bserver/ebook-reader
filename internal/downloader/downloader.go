package downloader

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

// Downloader HTTP 下载器，支持 singleflight 去重
type Downloader struct {
	dataDir  string
	mu       sync.Mutex
	inflight map[string]*call
}

type call struct {
	wg  sync.WaitGroup
	err error
}

// New 创建下载器
func New(dataDir string) *Downloader {
	return &Downloader{
		dataDir:  dataDir,
		inflight: make(map[string]*call),
	}
}

// URLHash 计算 URL 的 sha256 hash
func URLHash(url string) string {
	h := sha256.Sum256([]byte(url))
	return fmt.Sprintf("%x", h[:16]) // 取前 16 字节，32 字符 hex
}

// Download 下载文件到缓存目录，返回本地文件路径和缓存目录
// 如果已缓存则直接返回，并发请求同一 URL 只下载一次
func (d *Downloader) Download(url string) (filePath string, cachePath string, err error) {
	hash := URLHash(url)
	cachePath = filepath.Join(d.dataDir, hash)
	filePath = filepath.Join(cachePath, "raw"+extFromURL(url))

	// 已缓存，直接返回
	if _, err := os.Stat(filePath); err == nil {
		return filePath, cachePath, nil
	}

	// singleflight: 同一 URL 只下载一次
	d.mu.Lock()
	if c, ok := d.inflight[hash]; ok {
		d.mu.Unlock()
		c.wg.Wait()
		if c.err != nil {
			return "", "", c.err
		}
		return filePath, cachePath, nil
	}
	c := &call{}
	c.wg.Add(1)
	d.inflight[hash] = c
	d.mu.Unlock()

	// 执行下载
	c.err = d.doDownload(url, filePath, cachePath)
	c.wg.Done()

	d.mu.Lock()
	delete(d.inflight, hash)
	d.mu.Unlock()

	if c.err != nil {
		return "", "", c.err
	}
	return filePath, cachePath, nil
}

func (d *Downloader) doDownload(url string, filePath string, cachePath string) error {
	if err := os.MkdirAll(cachePath, 0755); err != nil {
		return fmt.Errorf("mkdir cache: %w", err)
	}

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http status: %d", resp.StatusCode)
	}

	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		os.Remove(filePath)
		return fmt.Errorf("download: %w", err)
	}
	return nil
}

// extFromURL 从 URL 提取文件扩展名
func extFromURL(url string) string {
	// 去掉 query string
	path := url
	if idx := len(path) - 1; idx > 0 {
		for i := len(path) - 1; i >= 0; i-- {
			if path[i] == '?' {
				path = path[:i]
				break
			}
		}
	}
	ext := filepath.Ext(path)
	if ext == "" {
		ext = ".bin"
	}
	return ext
}
