test:
	go get ./...
	go get github.com/dustinkirkland/golang-petname
	go test -v ./lxd

testacc:
	TF_LOG=debug TF_ACC=1 go test ./lxd -v $(TESTARGS)
