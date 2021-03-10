PACKAGE=github.com/zefhemel/matterless

build:
	go build ${PACKAGE}/cmd/mls-bot
	go build ${PACKAGE}/cmd/mls
	go build ${PACKAGE}/cmd/mls-lambda

run: build
	./mls-bot

test:
	go test ${PACKAGE}/pkg/...

mls-test: build
	./mls test.md

build-docker: build
	GOARCH=arm64 GOOS=linux go build ${PACKAGE}/cmd/mls-lambda
	cp mls-lambda runners/node-docker/
	docker build -t zefhemel/matterless-runner-docker-node runners/node-docker

push: build
	docker push zefhemel/matterless-runner-docker-node