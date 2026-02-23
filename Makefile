.PHONY: build frontend backend docker clean

build: frontend backend

frontend:
	cd frontend && npm install && npm run build

backend:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o ebook-reader ./cmd/server/

docker:
	podman build -t ebook-reader .

clean:
	rm -rf static/dist/* ebook-reader data/
