GOFMT_FILES?=$$(find . -name '*.go' |grep -v vendor)

test:
	go get ./...
	go get github.com/dustinkirkland/golang-petname
	go test -timeout 20m -v ./lxd

testacc:
	TF_LOG=debug TF_ACC=1 go test ./lxd -v $(TESTARGS)

build:
	go build -v
	tar czvf terraform-provider-lxd_${TRAVIS_TAG}_linux_amd64.tar.gz terraform-provider-lxd

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
