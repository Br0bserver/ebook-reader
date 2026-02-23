package parser

import (
	"ebook-reader/internal/model"
	"fmt"
	"path/filepath"
	"strings"
)

// Parser 电子书解析器统一接口
type Parser interface {
	// Parse 解析电子书，cachePath 为解压/存储目录
	Parse(filePath string, cachePath string) (*model.Book, error)
	// ReadChapter 按需读取章节内容，返回 HTML 字符串
	// fileURL 用于改写资源路径中的 ?file= 参数
	ReadChapter(book *model.Book, chapterID int, fileURL string) (string, error)
}

// GetParser 根据文件扩展名或格式名返回对应的解析器
func GetParser(nameOrFormat string) (Parser, error) {
	s := strings.ToLower(nameOrFormat)
	// 支持直接传格式名 "epub" 或文件路径 "xxx.epub"
	ext := filepath.Ext(s)
	if ext == "" {
		ext = "." + s
	}
	switch ext {
	case ".epub":
		return &EPUBParser{}, nil
	case ".txt":
		return &TXTParser{}, nil
	default:
		return nil, fmt.Errorf("unsupported format: %s", ext)
	}
}
