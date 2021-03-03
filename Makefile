PACKAGE=github.com/zefhemel/matterless

build:
	go build ${PACKAGE}/cmd/matterless

run: build
	./matterless

install:
	go install ${PACKAGE}/cmd/matterless

test:
	go test ${PACKAGE}/pkg/...
