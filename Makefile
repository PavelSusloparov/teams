.PHONY: help
## help: prints this help message
help:
	@echo "Usage: \n"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

.PHONY: test
## test: Runs go test
test:
	@go test -cover ./...

.PHONY: lint
## lint: Runs golanglint-ci
lint:
	@golangci-lint run

.PHONY: build
## build: Runs go build (builds `app` binary)
build:
	@go build -o bin/app .

.PHONY: run-app
## run-app: Run app to generate output/graph.dot file
run-app: build
	@mkdir -p output
	@./bin/app

.PHONY: run-graphviz
## run-graphviz: Run graphviz to generate .png file from .dot file
# Install graphviz with `brew install graphviz`
run-graphviz:
	@dot -Tpng -o output/graph.png output/graph.dot
	@dot -Tsvg -o output/graph.svg output/graph.dot

.PHONY: run
## run: Run app and graphviz
run: run-app run-graphviz

.PHONY: clean
## clean: Clean up
clean:
	@rm -rf bin
	@rm -rf output
