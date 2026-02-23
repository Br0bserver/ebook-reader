package parser

import (
	"archive/zip"
	"ebook-reader/internal/model"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// EPUBParser EPUB 格式解析器
type EPUBParser struct{}

// opfPackage OPF 包文档结构
type opfPackage struct {
	XMLName  xml.Name    `xml:"package"`
	Metadata opfMetadata `xml:"metadata"`
	Manifest opfManifest `xml:"manifest"`
	Spine    opfSpine    `xml:"spine"`
}

type opfMetadata struct {
	Title   string `xml:"title"`
	Creator string `xml:"creator"`
}

type opfManifest struct {
	Items []opfItem `xml:"item"`
}

type opfItem struct {
	ID         string `xml:"id,attr"`
	Href       string `xml:"href,attr"`
	MediaType  string `xml:"media-type,attr"`
	Properties string `xml:"properties,attr"`
}

type opfSpine struct {
	ItemRefs []opfItemRef `xml:"itemref"`
}

type opfItemRef struct {
	IDRef string `xml:"idref,attr"`
}

// container.xml 结构
type container struct {
	XMLName   xml.Name       `xml:"container"`
	RootFiles []rootFileItem `xml:"rootfiles>rootfile"`
}

type rootFileItem struct {
	FullPath string `xml:"full-path,attr"`
}

// 匹配 src="..." href="..." xlink:href="..." 中的相对路径资源引用
var resourceAttrRe = regexp.MustCompile(`(?i)(src|href|xlink:href)\s*=\s*"([^"]*)"`)

func (p *EPUBParser) Parse(filePath string, cachePath string) (*model.Book, error) {
	// 解压 EPUB 到 cachePath
	if err := unzipEPUB(filePath, cachePath); err != nil {
		return nil, fmt.Errorf("unzip epub: %w", err)
	}

	// 读取 META-INF/container.xml 找到 OPF 路径
	containerPath := filepath.Join(cachePath, "META-INF", "container.xml")
	containerData, err := os.ReadFile(containerPath)
	if err != nil {
		return nil, fmt.Errorf("read container.xml: %w", err)
	}

	var cont container
	if err := xml.Unmarshal(containerData, &cont); err != nil {
		return nil, fmt.Errorf("parse container.xml: %w", err)
	}
	if len(cont.RootFiles) == 0 {
		return nil, fmt.Errorf("no rootfile in container.xml")
	}

	opfPath := filepath.Join(cachePath, cont.RootFiles[0].FullPath)
	opfDir := filepath.Dir(opfPath)

	// 解析 OPF
	opfData, err := os.ReadFile(opfPath)
	if err != nil {
		return nil, fmt.Errorf("read opf: %w", err)
	}

	var pkg opfPackage
	if err := xml.Unmarshal(opfData, &pkg); err != nil {
		return nil, fmt.Errorf("parse opf: %w", err)
	}

	// 构建 manifest id -> item 映射
	itemMap := make(map[string]opfItem, len(pkg.Manifest.Items))
	for _, item := range pkg.Manifest.Items {
		itemMap[item.ID] = item
	}

	// 按 spine 顺序构建章节列表
	book := &model.Book{
		Title:     pkg.Metadata.Title,
		Author:    pkg.Metadata.Creator,
		Format:    "epub",
		CachePath: cachePath,
	}

	for i, ref := range pkg.Spine.ItemRefs {
		item, ok := itemMap[ref.IDRef]
		if !ok {
			continue
		}
		chapter := model.Chapter{
			ID:       i,
			Title:    fmt.Sprintf("Chapter %d", i+1),
			FilePath: filepath.Join(opfDir, item.Href),
		}
		book.Chapters = append(book.Chapters, chapter)
	}

	// 查找封面图片：优先 properties="cover-image"，其次 id 包含 cover
	var coverHref string
	for _, item := range pkg.Manifest.Items {
		if item.Properties == "cover-image" {
			coverHref = item.Href
			break
		}
	}
	if coverHref == "" {
		for _, item := range pkg.Manifest.Items {
			if strings.Contains(strings.ToLower(item.ID), "cover") && strings.HasPrefix(item.MediaType, "image/") {
				coverHref = item.Href
				break
			}
		}
	}
	if coverHref != "" {
		book.CoverFilePath = filepath.Join(opfDir, coverHref)
	}

	return book, nil
}

func (p *EPUBParser) ReadChapter(book *model.Book, chapterID int, fileURL string) (string, error) {
	if chapterID < 0 || chapterID >= len(book.Chapters) {
		return "", fmt.Errorf("chapter %d out of range", chapterID)
	}
	ch := book.Chapters[chapterID]
	data, err := os.ReadFile(ch.FilePath)
	if err != nil {
		return "", fmt.Errorf("read chapter file: %w", err)
	}

	content := string(data)

	// 改写章节内容中的资源路径为 API 代理地址
	// 章节文件所在目录（相对于 cachePath 解析相对路径）
	chapterDir := filepath.Dir(ch.FilePath)

	content = resourceAttrRe.ReplaceAllStringFunc(content, func(match string) string {
		subs := resourceAttrRe.FindStringSubmatch(match)
		if len(subs) < 3 {
			return match
		}
		attr := subs[1]
		val := subs[2]

		// 跳过: data URI, 绝对 URL, 锚点链接, 空值
		if val == "" || strings.HasPrefix(val, "data:") || strings.HasPrefix(val, "http://") || strings.HasPrefix(val, "https://") || strings.HasPrefix(val, "#") {
			return match
		}

		// 对于 href 属性，只改写指向资源文件的（图片/CSS/字体），跳过 .xhtml/.html 章节链接
		if strings.EqualFold(attr, "href") {
			ext := strings.ToLower(filepath.Ext(val))
			if ext == ".xhtml" || ext == ".html" || ext == ".htm" || ext == "" {
				return match
			}
		}

		// 解析相对路径为相对于 cachePath 的路径
		absPath := filepath.Join(chapterDir, val)
		relPath, err := filepath.Rel(book.CachePath, absPath)
		if err != nil {
			return match
		}
		// 统一用正斜杠
		relPath = filepath.ToSlash(relPath)

		return fmt.Sprintf(`%s="/api/book/resource/%s/%s"`, attr, fileURL, relPath)
	})

	return content, nil
}

// unzipEPUB 解压 EPUB 文件到目标目录
func unzipEPUB(src string, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		target := filepath.Join(dest, f.Name)

		// 防止 zip slip 攻击
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(target, 0755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		outFile, err := os.Create(target)
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
