test:
	go get ./...
	go get github.com/dustinkirkland/golang-petname
	go test -v ./lxd

testacc:
	TF_LOG=debug TF_ACC=1 go test ./lxd -v $(TESTARGS)

build:
	go build -v
	tar czvf terraform-provider-lxd_${TRAVIS_TAG}_linux_amd64.tar.gz terraform-provider-lxd

dev:
	go build -v