server:
	@go run main.go serve

client:
	@go run main.go client

buf:
	buf generate

fmt:
	go fmt ./...
	buf format -w

# this is to avoid make thinking vendor dir already exists
.PHONY: vendor
vendor:
	go mod tidy && go mod vendor

test:
	go test -v ./...
