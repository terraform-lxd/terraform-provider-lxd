GOFMT_FILES?=$$(find . -name '*.go' |grep -v vendor)
TARGETS=darwin linux windows

default: build

test:
	go get -d -t ./...
	go test -timeout 20m -v ./lxd

testacc:
	TF_LOG=debug TF_ACC=1 go test -v $(TESTARGS) -timeout 60m ./lxd

build:
	go build -v

targets: $(TARGETS)

$(TARGETS):
	GOOS=$@ GOARCH=amd64 go build -o "dist/$@/terraform-provider-lxd_${TRAVIS_TAG}_x4"
	zip -j dist/terraform-provider-lxd_${TRAVIS_TAG}_$@_amd64.zip dist/$@/terraform-provider-lxd_${TRAVIS_TAG}_x4

dev:
	go build -v

vet:
	@echo "go vet ."
	@go vet $$(go list ./... | grep -v vendor/) ; if [ $$? -eq 1 ]; then \
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

.PHONY: build test testacc dev vet fmt fmtcheck targets $(TARGETS)
