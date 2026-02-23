package cache

import (
	"ebook-reader/internal/model"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Cache 管理书籍缓存：内存中的 BookMeta + 磁盘文件 + TTL 清理
type Cache struct {
	dataDir string
	ttl     time.Duration

	mu    sync.RWMutex
	books map[string]*entry // key: url hash
}

type entry struct {
	book     *model.Book
	lastUsed time.Time
}

// New 创建缓存管理器
func New(dataDir string, ttl time.Duration) *Cache {
	c := &Cache{
		dataDir: dataDir,
		ttl:     ttl,
		books:   make(map[string]*entry),
	}
	go c.cleanupLoop()
	return c
}

// Get 获取缓存的书籍元数据，命中则更新访问时间
func (c *Cache) Get(hash string) (*model.Book, bool) {
	c.mu.RLock()
	e, ok := c.books[hash]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	c.mu.Lock()
	e.lastUsed = time.Now()
	c.mu.Unlock()
	return e.book, true
}

// Put 存入书籍元数据
func (c *Cache) Put(hash string, book *model.Book) {
	c.mu.Lock()
	c.books[hash] = &entry{
		book:     book,
		lastUsed: time.Now(),
	}
	c.mu.Unlock()
}

// DataDir 返回数据目录路径
func (c *Cache) DataDir() string {
	return c.dataDir
}

// cleanupLoop 后台定时清理过期缓存
func (c *Cache) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		c.cleanup()
	}
}

func (c *Cache) cleanup() {
	now := time.Now()
	var expired []string

	c.mu.RLock()
	for hash, e := range c.books {
		if now.Sub(e.lastUsed) > c.ttl {
			expired = append(expired, hash)
		}
	}
	c.mu.RUnlock()

	for _, hash := range expired {
		c.mu.Lock()
		// 二次检查，防止刚被访问
		if e, ok := c.books[hash]; ok && now.Sub(e.lastUsed) > c.ttl {
			delete(c.books, hash)
			// 删除磁盘文件
			os.RemoveAll(filepath.Join(c.dataDir, hash))
		}
		c.mu.Unlock()
	}
}
