PACKAGE=github.com/zefhemel/matterless
DOCKER_RUNNER_IMAGE=zefhemel/matterless-runner-docker

build:
	go build ${PACKAGE}/cmd/mls-bot
	go build ${PACKAGE}/cmd/mls
	go build ${PACKAGE}/cmd/mls-lambda

install:
	go install ${PACKAGE}/cmd/mls
	go install ${PACKAGE}/cmd/matterlessd
	go install ${PACKAGE}/cmd/mls-watch

run-bot: build
	./mls-bot

test:
	go test ${PACKAGE}/pkg/...

test-short:
	go test -short ${PACKAGE}/pkg/...

run-mls-test: build
	./mls test.md

docker:
	GOARCH=amd64 GOOS=linux go build -o runners/docker/mls-lambda.x86_64 ${PACKAGE}/cmd/mls-lambda
	GOARCH=arm64 GOOS=linux go build -o runners/docker/mls-lambda.aarch64 ${PACKAGE}/cmd/mls-lambda
	docker build -t ${DOCKER_RUNNER_IMAGE} runners/docker

docker-push:
	GOARCH=amd64 GOOS=linux go build -o runners/docker/mls-lambda.x86_64 ${PACKAGE}/cmd/mls-lambda
	GOARCH=arm64 GOOS=linux go build -o runners/docker/mls-lambda.aarch64 ${PACKAGE}/cmd/mls-lambda
	docker buildx build --platform linux/amd64,linux/arm64 -t ${DOCKER_RUNNER_IMAGE} --push runners/docker
	docker build -t ${DOCKER_RUNNER_IMAGE} runners/docker
