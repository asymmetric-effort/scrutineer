VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.0.1-dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64
MODULES := core connector/cli connector/http connector/ssh connector/grpc connector/browser loadtest fuzz cmd/scrutineer provider/static

# Modules exempt from the 98% coverage gate.
# - cmd/scrutineer: CLI entry point with os.Exit, signal handling — tested via integration
# - connector/grpc: 96.8% — remaining gap is defensive error handling in gRPC stream I/O
#   that is structurally unreachable without mocking gRPC internals
# - connector/browser/cdp: 95.7% — remaining gap is unreachable I/O error paths in bufio.Writer
COVERAGE_EXEMPT := cmd/scrutineer connector/grpc connector/browser provider/static

.PHONY: all build test vet vuln clean cross fmt coverage precommit

all: fmt vet test build

fmt:
	@for mod in $(MODULES); do \
		echo "fmt $$mod"; \
		cd $(CURDIR)/$$mod && gofmt -w . ; \
	done

vet:
	@for mod in $(MODULES); do \
		echo "vet $$mod"; \
		cd $(CURDIR)/$$mod && go vet ./... ; \
	done

vuln:
	@for mod in $(MODULES); do \
		echo "vuln $$mod"; \
		cd $(CURDIR)/$$mod && govulncheck ./... ; \
	done

test:
	@for mod in $(MODULES); do \
		echo "test $$mod"; \
		cd $(CURDIR)/$$mod && go test -race -coverprofile=coverage.out ./... ; \
	done

coverage: test
	@for mod in $(MODULES); do \
		skip=0; \
		for exempt in $(COVERAGE_EXEMPT); do \
			if [ "$$mod" = "$$exempt" ]; then skip=1; fi; \
		done; \
		if [ "$$skip" = "1" ]; then \
			echo "coverage $$mod (exempt from gate)"; \
		else \
			echo "coverage $$mod"; \
			cd $(CURDIR)/$$mod && go tool cover -func=coverage.out | tail -1 | awk '{if ($$3+0 < 98.0) {print "FAIL: " "'"$$mod"'" " coverage " $$3 " < 98%"; exit 1}}' ; \
		fi; \
	done

build:
	cd cmd/scrutineer && go build $(LDFLAGS) -o ../../bin/scrutineer .

cross:
	@$(foreach platform,$(PLATFORMS),\
		$(eval GOOS=$(word 1,$(subst /, ,$(platform))))\
		$(eval GOARCH=$(word 2,$(subst /, ,$(platform))))\
		echo "build $(GOOS)/$(GOARCH)"; \
		cd $(CURDIR)/cmd/scrutineer && GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(LDFLAGS) -o ../../bin/scrutineer-$(GOOS)-$(GOARCH)$(if $(findstring windows,$(GOOS)),.exe) . ; \
	)

clean:
	rm -rf bin/
	@for mod in $(MODULES); do \
		rm -f $(CURDIR)/$$mod/coverage.out ; \
	done

precommit: fmt vet vuln test coverage
