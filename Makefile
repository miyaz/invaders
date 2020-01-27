gxport GO111MODULE=on

## Install for Development
.PHONY: devel-deps
devel-deps: deps
	GO111MODULE=off go get -u                   \
	  golang.org/x/lint/golint                  \
	  golang.org/x/tools/cmd/goimports

## Clean binaries
.PHONY: clean
clean:
	rm -rf invaders
	rm -rf dist/

## Run tests
.PHONY: test
test: deps
	go test -cover -v ./...

## Install dependencies
.PHONY: deps
deps:
	go install github.com/rakyll/statik
	go get ./...

## Update dependencies
.PHONY: update
update:
	go get -u -d ./...
	go mod tidy

## Run Lint
.PHONY: lint
lint: devel-deps
	go vet ./...
	golint -set_exit_status ./...

## Format source codes
.PHONY: fmt
fmt: devel-deps
	goimports -w ./...

## Cross build binaries
.PHONY: cross-build
cross-build:
	statik -src=./files -f
	for os in darwin linux windows; do \
		GOOS=$$os GOARCH=amd64 go build -o dist/invaders-$$os main.go; \
	done

#GOOS=$$os GOARCH=amd64 CGO_ENABLED=0 go build -o dist/invaders-$$os main.go;
## Build binaries ex. make bin/autotypesetter
.PHONY: build
build:
#go:generate not support go modules
	statik -src=./files -f
	go build -o invaders main.go

