OPENAPI_SPEC=api/openapi.yaml
OPENAPI_OUT=internal/transport/httpapi/openapi.gen.go
OPENAPI_PKG=httpapi

.PHONY: openapi-gen
openapi-gen:
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -generate gin,types -package $(OPENAPI_PKG) -o $(OPENAPI_OUT) $(OPENAPI_SPEC)

.PHONY: test-integration
test-integration:
	go test -tags=integration -run ^TestServerContainerAPI$$ ./internal/integration

.PHONY: coverage
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -n 1
