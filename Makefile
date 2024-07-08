.PHONY: build test

test:
	-@rm -rf ~/.locreg
	@echo "====================Running tests===================="
	@go clean -testcache
	@go test $(filter-out $@,$(MAKECMDGOALS)) ./...
	-@rm -rf ~/.locreg

build: test
	@echo "====================Building binary===================="
	go build -o locreg
	@echo "Binary built successfully"