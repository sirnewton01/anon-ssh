.PHONY: windows_amd64
windows_amd64:
	mkdir -p release/windows-amd64/
	GOOS=windows GOARCH=amd64 go build -o release/windows-amd64/gemini.exe ./cmd/gemini
	GOOS=windows GOARCH=amd64 go build -o release/windows-amd64/ssh-capsule-server.exe ./cmd/ssh-capsule-server
	GOOS=windows GOARCH=amd64 go build -o release/windows-amd64/capsule.exe ./cmd/capsule

.PHONY: linux_amd64
linux_amd64:
	mkdir -p release/linux-amd64/
	GOOS=linux GOARCH=amd64 go build -o release/linux-amd64/gemini ./cmd/gemini
	GOOS=linux GOARCH=amd64 go build -o release/linux-amd64/ssh-capsule-server ./cmd/ssh-capsule-server
	GOOS=linux GOARCH=amd64 go build -o release/linux-amd64/capsule ./cmd/capsule

.PHONY: darwin_amd64
darwin_amd64:
	mkdir -p release/darwin-amd64/
	GOOS=darwin GOARCH=amd64 go build -o release/darwin-amd64/gemini ./cmd/gemini
	GOOS=darwin GOARCH=amd64 go build -o release/darwin-amd64/ssh-capsule-server ./cmd/ssh-capsule-server
	GOOS=darwin GOARCH=amd64 go build -o release/darwin-amd64/capsule ./cmd/capsule

.PHONY: darwin_arm64
darwin_arm64:
	mkdir -p release/darwin-arm64/
	GOOS=darwin GOARCH=arm64 go build -o release/darwin-arm64/gemini ./cmd/gemini
	GOOS=darwin GOARCH=arm64 go build -o release/darwin-arm64/ssh-capsule-server ./cmd/ssh-capsule-server
	GOOS=darwin GOARCH=arm64 go build -o release/darwin-arm64/capsule ./cmd/capsule

.PHONY: build
build:  windows_amd64 linux_amd64 darwin_amd64 darwin_arm64

.PHONY: clean
clean:
	rm -rf release
