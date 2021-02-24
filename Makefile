PACKAGE=github.com/zefhemel/matterless
build:
	go build ${PACKAGE}/cmd/matterless

install:
	go install ${PACKAGE}/cmd/matterless

test:
	go test ${PACKAGE}/pkg/...
