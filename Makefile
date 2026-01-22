OPENAPI_SPEC=api/openapi.yaml
OPENAPI_OUT=internal/transport/httpapi/openapi.gen.go
OPENAPI_PKG=httpapi
CLIENT_BIN=dist/gophkeeper-cli

.PHONY: openapi-gen
openapi-gen:
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -generate gin,types -package $(OPENAPI_PKG) -o $(OPENAPI_OUT) $(OPENAPI_SPEC)

.PHONY: test-integration
test-integration:
	go test -tags=integration ./internal/integration

.PHONY: coverage
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -n 1

.PHONY: build-client-linux
build-client-linux:
	mkdir -p dist
	GOOS=linux GOARCH=amd64 go build -o $(CLIENT_BIN)-linux-amd64 ./cmd/client

.PHONY: build-client-windows
build-client-windows:
	mkdir -p dist
	GOOS=windows GOARCH=amd64 go build -o $(CLIENT_BIN)-windows-amd64.exe ./cmd/client

.PHONY: build-client-mac
build-client-mac:
	mkdir -p dist
	GOOS=darwin GOARCH=arm64 go build -o $(CLIENT_BIN)-darwin-arm64 ./cmd/client
