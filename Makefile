build-me:
	go build -o build/tctxto

run-me: 
	go build -o build/tctxto && ./build/tctxto

build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build -o build/tctxto_darwin_amd64

build-windows-amd64:
	GOOS=windows GOARCH=amd64 go build -o build/tctxto_windows_amd64.exe

build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build -o build/tctxto_linux_amd64

build-all: build-darwin-amd64 build-windows-amd64 build-linux-amd64
