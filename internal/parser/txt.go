package parser

import (
	"bufio"
	"bytes"
	"ebook-reader/internal/model"
	"fmt"
	"io"
	"os"
	"regexp"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// TXTParser TXT 格式解析器
type TXTParser struct{}

// 章节标题匹配正则：第X章、第X节、Chapter X 等
var chapterPattern = regexp.MustCompile(
	`(?m)^\s*(第[零一二三四五六七八九十百千万\d]+[章节回卷集部篇]|Chapter\s+\d+|CHAPTER\s+\d+)(.*)$`,
)

func (p *TXTParser) Parse(filePath string, cachePath string) (*model.Book, error) {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read txt: %w", err)
	}

	// 编码检测与转换
	content := detectAndConvert(raw)

	// 将转换后的 UTF-8 内容写入缓存目录
	utf8Path := filePath
	if !utf8.Valid(raw) {
		utf8Path = cachePath + "/content.txt"
		if err := os.WriteFile(utf8Path, []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("write utf8 txt: %w", err)
		}
	}

	book := &model.Book{
		Title:     "Unknown",
		Author:    "Unknown",
		Format:    "txt",
		CachePath: cachePath,
	}

	// 按正则分章
	matches := chapterPattern.FindAllStringIndex(content, -1)

	if len(matches) == 0 {
		// 没有匹配到章节标题，按固定大小分割
		book.Chapters = splitBySize(content, utf8Path, 8192)
	} else {
		for i, loc := range matches {
			title := content[loc[0]:loc[1]]
			offset := int64(loc[0])
			var length int64
			if i+1 < len(matches) {
				length = int64(matches[i+1][0]) - offset
			} else {
				length = int64(len(content)) - offset
			}
			book.Chapters = append(book.Chapters, model.Chapter{
				ID:       i,
				Title:    trimTitle(title),
				FilePath: utf8Path,
				Offset:   offset,
				Length:   length,
			})
		}
	}

	return book, nil
}

func (p *TXTParser) ReadChapter(book *model.Book, chapterID int, fileURL string) (string, error) {
	if chapterID < 0 || chapterID >= len(book.Chapters) {
		return "", fmt.Errorf("chapter %d out of range", chapterID)
	}
	ch := book.Chapters[chapterID]

	f, err := os.Open(ch.FilePath)
	if err != nil {
		return "", fmt.Errorf("open txt: %w", err)
	}
	defer f.Close()

	if _, err := f.Seek(ch.Offset, io.SeekStart); err != nil {
		return "", fmt.Errorf("seek: %w", err)
	}

	buf := make([]byte, ch.Length)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.ErrUnexpectedEOF {
		return "", fmt.Errorf("read chapter: %w", err)
	}

	// 包裹为简单 HTML 段落
	text := string(buf[:n])
	return txtToHTML(text), nil
}

// detectAndConvert 检测编码并转换为 UTF-8
func detectAndConvert(data []byte) string {
	if utf8.Valid(data) {
		return string(data)
	}

	// 尝试 GBK 解码
	reader := transform.NewReader(bytes.NewReader(data), simplifiedchinese.GBK.NewDecoder())
	decoded, err := io.ReadAll(reader)
	if err == nil && utf8.Valid(decoded) {
		return string(decoded)
	}

	// 尝试 GB18030（GBK 超集）
	reader = transform.NewReader(bytes.NewReader(data), simplifiedchinese.GB18030.NewDecoder())
	decoded, err = io.ReadAll(reader)
	if err == nil && utf8.Valid(decoded) {
		return string(decoded)
	}

	// 兜底：强制当 UTF-8 处理，替换无效字节
	return string(data)
}

// splitBySize 无章节标题时按字节大小分割
func splitBySize(content string, filePath string, chunkSize int) []model.Chapter {
	var chapters []model.Chapter
	data := []byte(content)
	total := len(data)

	for i := 0; i < total; {
		end := i + chunkSize
		if end > total {
			end = total
		}
		// 避免在 UTF-8 多字节字符中间截断
		for end < total && !utf8.RuneStart(data[end]) {
			end++
		}
		chapters = append(chapters, model.Chapter{
			ID:       len(chapters),
			Title:    fmt.Sprintf("Section %d", len(chapters)+1),
			FilePath: filePath,
			Offset:   int64(i),
			Length:   int64(end - i),
		})
		i = end
	}
	return chapters
}

// trimTitle 清理章节标题的前后空白
func trimTitle(s string) string {
	scanner := bufio.NewScanner(bytes.NewReader([]byte(s)))
	if scanner.Scan() {
		line := scanner.Text()
		// 去除前后空白
		result := bytes.TrimSpace([]byte(line))
		return string(result)
	}
	return s
}

// txtToHTML 将纯文本转为简单 HTML
func txtToHTML(text string) string {
	var buf bytes.Buffer
	buf.WriteString("<div class=\"txt-chapter\">")
	scanner := bufio.NewScanner(bytes.NewReader([]byte(text)))
	for scanner.Scan() {
		line := scanner.Text()
		if len(bytes.TrimSpace([]byte(line))) == 0 {
			continue
		}
		buf.WriteString("<p>")
		buf.WriteString(htmlEscape(line))
		buf.WriteString("</p>")
	}
	buf.WriteString("</div>")
	return buf.String()
}

// htmlEscape 基础 HTML 转义
func htmlEscape(s string) string {
	s = bytes.NewBuffer(bytes.ReplaceAll([]byte(s), []byte("&"), []byte("&amp;"))).String()
	s = bytes.NewBuffer(bytes.ReplaceAll([]byte(s), []byte("<"), []byte("&lt;"))).String()
	s = bytes.NewBuffer(bytes.ReplaceAll([]byte(s), []byte(">"), []byte("&gt;"))).String()
	return s
}
