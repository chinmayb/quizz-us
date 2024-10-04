
server:
	@go run main.go serve

run-client:
	@go run main.go client

buf:
	buf generate

fmt:
	go fmt ./...
	buf format -w
