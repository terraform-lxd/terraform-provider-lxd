GOFMT_FILES?=$$(find . -name '*.go' |grep -v vendor)
TARGETS=darwin/amd64 freebsd/386 freebsd/amd64 freebsd/arm linux/386 linux/amd64 linux/arm openbsd/386 openbsd/amd64 windows/386 windows/amd64

default: build

test:
	go get -d -t ./...
	go test -timeout 20m -v ./lxd

testacc:
	TF_LOG=debug TF_ACC=1 go test -v $(TESTARGS) -timeout 60m ./lxd

build:
	go build -v

targets:
	gox -osarch='$(TARGETS)' -output="dist/{{.OS}}_{{.Arch}}/terraform-provider-lxd_${TRAVIS_TAG}_x4"
	find dist -maxdepth 1 -mindepth 1 -type d -print0 | \
	sed -z -e 's,^dist/,,' | \
	xargs -0 --verbose --replace={} zip -r -j "dist/terraform-provider-lxd_${TRAVIS_TAG}_{}.zip" "dist/{}"

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

.PHONY: build test testacc dev vet fmt fmtcheck targets
