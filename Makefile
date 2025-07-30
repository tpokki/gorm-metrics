# yellow check mark
YC=\033[0;33m✔︎\033[0m
#green check mark
GC=\033[0;32m✔︎\033[0m
SRC=$(wildcard *.go)

.PHONY: all test

all: test
test:; @ echo "$(YC) running tests..." ;
	@ go test -v ./...
