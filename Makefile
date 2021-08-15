.PHONY: windows
windows:
	mkdir -p release/windows/
	GOOS=windows GOARCH=amd64 go build -o release/windows/gemini.exe ./cmd/gemini
	GOOS=windows GOARCH=amd64 go build -o release/windows/ssh-capsule-server.exe ./cmd/ssh-capsule-server
	GOOS=windows GOARCH=amd64 go build -o release/windows/capsule.exe ./cmd/capsule

.PHONY: linux
linux:
	mkdir -p release/linux/
	GOOS=linux GOARCH=amd64 go build -o release/linux/gemini ./cmd/gemini
	GOOS=linux GOARCH=amd64 go build -o release/linux/ssh-capsule-server ./cmd/ssh-capsule-server
	GOOS=linux GOARCH=amd64 go build -o release/linux/capsule ./cmd/capsule

.PHONY: darwin
darwin:
	mkdir -p release/darwin/
	GOOS=darwin GOARCH=amd64 go build -o release/darwin/gemini ./cmd/gemini
	GOOS=darwin GOARCH=amd64 go build -o release/darwin/ssh-capsule-server ./cmd/ssh-capsule-server
	GOOS=darwin GOARCH=amd64 go build -o release/darwin/capsule ./cmd/capsule

.PHONY: build
build:  windows linux darwin
