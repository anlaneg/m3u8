all:
	go build -o m3u8 main.go
	go build -o m3u8-detect cmd/m3u8-detect.go
