GO ?= go
GOFMT_FILES?=$$(find . -name '*.go' |grep -v vendor)
TARGETS=darwin/amd64 freebsd/386 freebsd/amd64 freebsd/arm linux/386 linux/amd64 linux/arm openbsd/386 openbsd/amd64 windows/386 windows/amd64
TF_LOG?=error

default: build

test:
	$(GO) get -d -t ./...
	$(GO) test -parallel $$(nproc) -race -timeout 60m -v ./internal/...

testacc:
	TF_LOG=$(TF_LOG) TF_ACC=1 $(GO) test -parallel 4 -v -race $(TESTARGS) -timeout 60m ./internal/...

build:
	$(GO) build -v

targets:
	gox -osarch='$(TARGETS)' -output="dist/{{.OS}}_{{.Arch}}/terraform-provider-lxd_${TRAVIS_TAG}_x4"
	find dist -maxdepth 1 -mindepth 1 -type d -print0 | \
	sed -z -e 's,^dist/,,' | \
	xargs -0 --verbose --replace={} zip -r -j "dist/terraform-provider-lxd_${TRAVIS_TAG}_{}.zip" "dist/{}"

dev:
	$(GO) build -v

vet:
	@echo "$(GO) vet ."
	@$(GO) vet $$($(GO) list ./... | grep -v vendor/) ; if [ $$? -eq 1 ]; then \
		echo ""; \
		echo "Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for review."; \
		exit 1; \
	fi

fmt:
	gofmt -w $(GOFMT_FILES)

fmtcheck:
	@echo "==> Checking that code complies with gofmt requirements..." ; \
	files=$$(find . -name '*.go' ) ; \
	gofmt_files=`gofmt -l $$files`; \
	if [ -n "$$gofmt_files" ]; then \
		echo 'gofmt needs running on the following files:'; \
		echo "$$gofmt_files"; \
		echo "You can use the command: \`make fmt\` to reformat code."; \
		exit 1; \
	fi

.PHONY: static-analysis
static-analysis:
	@if command -v golangci-lint > /dev/null; then \
		echo "==> Running golangci-lint"; \
		golangci-lint run --timeout 5m; \
	else \
		echo "Missing \"golangci-lint\" command, not linting .go" >&2; \
	fi
	@if command -v terraform > /dev/null; then \
		echo "==> Running terraform fmt"; \
		terraform fmt -recursive -check -diff; \
	else \
		echo "Missing \"terraform\" command, not checking .tf format" >&2; \
	fi

.PHONY: update-gomod
update-gomod:
	$(GO) get -t -v -u ./...
	$(GO) mod tidy --go=1.22.3
	$(GO) get toolchain@none
	@echo "Dependencies updated"

.PHONY: build test testacc dev vet fmt fmtcheck targets
