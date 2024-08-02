.PHONY: build test

test:
	-@rm -rf ~/.locreg
	@echo "====================Running tests===================="
	@go clean -testcache
	@go test $(filter-out build test,$(MAKECMDGOALS)) ./...
	-@rm -rf ~/.locreg

build:
	@$(MAKE) test
	@echo "====================Building binary===================="
	go build -o locreg
	@echo "Binary built successfully"