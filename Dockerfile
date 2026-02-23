# Build
FROM docker.io/library/node:18-alpine AS frontend
WORKDIR /build/frontend
COPY frontend/package*.json ./
RUN npm install
COPY frontend/ ./
RUN npm run build

FROM docker.io/library/golang:1.24-alpine AS backend
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /build/frontend/../static/dist ./static/dist
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /ebook-reader ./cmd/server/

# Runtime
FROM scratch
COPY --from=backend /ebook-reader /ebook-reader
COPY --from=backend /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
EXPOSE 8080
ENTRYPOINT ["/ebook-reader"]
