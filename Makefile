all:
	@echo "available make targets: test test-extended"

test:
	go test ./parser
	go test ./pas2go

test-extended: test
	go test ./tests/pat
	go test ./tests/prt
