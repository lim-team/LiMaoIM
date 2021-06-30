build:
	go build -o limaoim cmd/app/main.go 
linux:
	CGO_ENABLED=0  GOOS=linux  GOARCH=amd64 go build -o limaoim-linux-amd64 cmd/app/main.go 