
run:
	go run main.go serve

buf:
	buf generate

fmt:
	go fmt ./...
	buf format -w
