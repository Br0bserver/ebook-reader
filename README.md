# eBook Reader

纯 Go 实现的在线电子书解析阅读器，前端嵌入单二进制文件，支持 `scratch` 镜像部署。

## 特性

- 支持 EPUB / TXT 格式，自动检测 GBK/GB18030/UTF-8 编码
- 通过 `?file=URL` 传入远程电子书地址，自动下载解析
- 前端 Vue 2 构建，兼容 Chromium 40+ / IE9
- `go:embed` 嵌入前端，单二进制零依赖运行
- 纯 Go 实现，`CGO_ENABLED=0` 静态编译，无第三方 C 库依赖
- 内存缓存书籍元数据 + 磁盘 TTL 自动清理
- 并发下载 singleflight 去重

## 快速开始

```bash
./ebook-reader -p 8080
```

浏览器访问 `http://localhost:8080/?file=https://example.com/book.epub`

参数说明：

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-p` | 8080 | 监听端口（也可通过环境变量 `PORT` 设置） |
| `-d` | data | 缓存目录 |
| `-ttl` | 24h | 缓存过期时间 |

优先级：命令行参数 > 环境变量 > 默认值

## 构建

### 前端

```bash
cd frontend
npm install
npm run build   # 输出到 ../static/dist/
```

### 后端

```bash
CGO_ENABLED=0 go build -ldflags="-s -w" -o ebook-reader ./cmd/server/
```

### 一键构建

```bash
make build
```

### Docker 多阶段构建

```bash
# 构建镜像（node 编译前端 -> go 编译后端 -> scratch 最终镜像）
make docker

# 运行（外部 3000 -> 容器内 8080，后台运行）
docker run -d -p 3000:8080 ebook-reader

# 通过环境变量修改容器内端口
docker run -d -p 3000:3000 -e PORT=3000 ebook-reader

# 通过命令行参数修改容器内端口
docker run -d -p 3000:3000 ebook-reader -p 3000
```
