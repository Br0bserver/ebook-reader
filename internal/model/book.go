package model

// Book 书籍元数据，解析后缓存在内存中
type Book struct {
	ID       string    `json:"id"` // URL 的 sha256 hash
	Title    string    `json:"title"`
	Author   string    `json:"author"`
	Format   string    `json:"format"`   // "epub" / "txt"
	CoverURL string    `json:"coverUrl"` // /api/book/cover?file=...
	Chapters []Chapter `json:"chapters"`
	// 内部字段，不序列化
	CachePath     string `json:"-"` // 磁盘缓存路径 data/cache/{hash}/
	CoverFilePath string `json:"-"` // 封面图片在磁盘上的绝对路径
}

// Chapter 章节信息
type Chapter struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	// EPUB: 解压后的文件绝对路径
	// TXT: 源文件路径
	FilePath string `json:"-"`
	// EPUB: 章节文件相对于 OPF 目录的路径（用于解析相对资源引用）
	RelDir string `json:"-"`
	// TXT 专用: 文件内的字节偏移和长度
	Offset int64 `json:"-"`
	Length int64 `json:"-"`
}
