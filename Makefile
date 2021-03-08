PACKAGE=github.com/zefhemel/matterless

build:
	go build ${PACKAGE}/cmd/mls-bot
	go build ${PACKAGE}/cmd/mls

run: build
	./mls-bot

install:
	go install ${PACKAGE}/cmd/mls-bot
	go install ${PACKAGE}/cmd/mls

test:
	go test ${PACKAGE}/pkg/...

mls-test: build
	./mls test.md