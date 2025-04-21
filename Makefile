build-macos:
	GOOS=darwin GOARCH=amd64 go build -o build/macos/tctxtosv

build-linux:
	GOOS=linux GOARCH=amd64 go build -o build/linux/tctxtosv

build-windows:
	GOOS=windows GOARCH=amd64 go build -o build/windows/tctxtosv

build-all: build-macos build-linux build-windows

run:
	go build -o tctxtosv && ./tctxtosv

run-w:
	go build -o tctxtosv.exe && ./tctxtosv.exe