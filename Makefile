OPENAPI_SPEC=api/openapi.yaml
OPENAPI_OUT=internal/transport/httpapi/openapi.gen.go
OPENAPI_PKG=httpapi
CLIENT_BIN=dist/gophkeeper-cli
RPM_VERSION?=0.1.0
RPM_RELEASE?=1
RPM_TOPDIR=dist/rpm
RPM_ENGINE?=docker
RPM_CONTAINER?=rockylinux:9
RPM_PLATFORM?=linux/amd64

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

.PHONY: build-rpm
build-rpm: build-rpm-server build-rpm-client clean-rpm-temp

.PHONY: build-rpm-server
build-rpm-server:
	mkdir -p $(RPM_TOPDIR)/SOURCES $(RPM_TOPDIR)/SPECS $(RPM_TOPDIR)/BUILD $(RPM_TOPDIR)/RPMS $(RPM_TOPDIR)/SRPMS dist
	GOOS=linux GOARCH=amd64 go build -o dist/gophkeeper-server-linux-amd64 ./cmd/server
	cp dist/gophkeeper-server-linux-amd64 $(RPM_TOPDIR)/SOURCES/gophkeeper-server
	cp packaging/config/server.yaml $(RPM_TOPDIR)/SOURCES/server.yaml
	cp packaging/rpm/gophkeeper-server.spec $(RPM_TOPDIR)/SPECS/gophkeeper-server.spec
	$(RPM_ENGINE) run --rm --platform=$(RPM_PLATFORM) -v $(PWD):/workspace -w /workspace $(RPM_CONTAINER) /bin/bash -c \
		"dnf -y install rpm-build >/dev/null && rpmbuild -bb $(RPM_TOPDIR)/SPECS/gophkeeper-server.spec --define '_topdir /workspace/$(RPM_TOPDIR)' --define 'version $(RPM_VERSION)' --define 'release $(RPM_RELEASE)'"

.PHONY: build-rpm-client
build-rpm-client:
	mkdir -p $(RPM_TOPDIR)/SOURCES $(RPM_TOPDIR)/SPECS $(RPM_TOPDIR)/BUILD $(RPM_TOPDIR)/RPMS $(RPM_TOPDIR)/SRPMS dist
	GOOS=linux GOARCH=amd64 go build -o dist/gophkeeper-cli-linux-amd64 ./cmd/client
	cp dist/gophkeeper-cli-linux-amd64 $(RPM_TOPDIR)/SOURCES/gophkeeper-cli
	cp packaging/config/client.yaml $(RPM_TOPDIR)/SOURCES/client.yaml
	cp packaging/rpm/gophkeeper-client.spec $(RPM_TOPDIR)/SPECS/gophkeeper-client.spec
	$(RPM_ENGINE) run --rm --platform=$(RPM_PLATFORM) -v $(PWD):/workspace -w /workspace $(RPM_CONTAINER) /bin/bash -c \
		"dnf -y install rpm-build >/dev/null && rpmbuild -bb $(RPM_TOPDIR)/SPECS/gophkeeper-client.spec --define '_topdir /workspace/$(RPM_TOPDIR)' --define 'version $(RPM_VERSION)' --define 'release $(RPM_RELEASE)'"

.PHONY: clean-rpm-temp
clean-rpm-temp:
	find $(RPM_TOPDIR)/RPMS -type f -name "*.rpm" -maxdepth 3 -print0 | xargs -0 -I{} cp {} dist/
	rm -rf $(RPM_TOPDIR)/BUILD $(RPM_TOPDIR)/SOURCES $(RPM_TOPDIR)/SPECS $(RPM_TOPDIR)/SRPMS
	rm -f dist/gophkeeper-server-linux-amd64 dist/gophkeeper-cli-linux-amd64

.PHONY: clean-rpm
clean-rpm:
	rm -rf $(RPM_TOPDIR) dist/gophkeeper-server-linux-amd64 dist/gophkeeper-cli-linux-amd64
