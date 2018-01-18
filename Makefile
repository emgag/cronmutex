.PHONY: build run vet lint
OUT := cronmutex
PKG := github.com/emgag/cronmutex
VERSION := $(shell git describe --always --long --dirty)
PKG_LIST := $(shell go list ${PKG}/... | grep -v /vendor/)
GO_FILES := $(shell find . -name '*.go' | grep -v /vendor/)

all: build

build:
	go build -v -o ${OUT} -ldflags="-X ${PKG}/internal/lib.Version=${VERSION}" ${PKG}

install:
	go install -v -o ${OUT} -ldflags="-X ${PKG}/internal/lib.Version=${VERSION}" ${PKG}

test:
	@go test -v ${PKG_LIST}

vet:
	@go vet ${PKG_LIST}

lint:
	@for file in ${GO_FILES} ;  do \
		golint $$file ; \
	done

fmt:
	@gofmt -l -w -s ${GO_FILES}

run: build
	./${OUT} listen

clean:
	-@rm ${OUT}

