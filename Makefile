.DEFAULT_GOAL := gen

gen:
	GOOS=linux GOARCH=amd64 go build ./cmd/app/main.go