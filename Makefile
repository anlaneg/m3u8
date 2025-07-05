all:
	go build -o m3u8 main.go
	go build -o m3u8-detect cmd/m3u8-detect.go
format:
	find . -type f -name '*.go' | xargs -t -i go fmt {}
