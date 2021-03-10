PACKAGE=github.com/zefhemel/matterless
DOCKER_RUNNER_IMAGE=zefhemel/matterless-runner-docker

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
	cp mls-lambda runners/docker/
	docker build -t ${DOCKER_RUNNER_IMAGE} runners/docker

push: build
	docker push ${DOCKER_RUNNER_IMAGE}