PACKAGE=github.com/zefhemel/matterless
DOCKER_RUNNER_IMAGE=zefhemel/matterless-runner-docker

install:
	go install ${PACKAGE}/cmd/mls

test:
	go test -cover ${PACKAGE}/pkg/...

test-short:
	go test -short ${PACKAGE}/pkg/...

vet:
	go vet ./...

docker:
	GOARCH=amd64 GOOS=linux go build -o runners/docker/mls-lambda.x86_64 ${PACKAGE}/cmd/mls-lambda
	GOARCH=arm64 GOOS=linux go build -o runners/docker/mls-lambda.aarch64 ${PACKAGE}/cmd/mls-lambda
	docker build -t ${DOCKER_RUNNER_IMAGE} runners/docker

docker-push:
	GOARCH=amd64 GOOS=linux go build -o runners/docker/mls-lambda.x86_64 ${PACKAGE}/cmd/mls-lambda
	GOARCH=arm64 GOOS=linux go build -o runners/docker/mls-lambda.aarch64 ${PACKAGE}/cmd/mls-lambda
	docker buildx build --platform linux/amd64,linux/arm64 -t ${DOCKER_RUNNER_IMAGE} --push runners/docker
	docker build -t ${DOCKER_RUNNER_IMAGE} runners/docker
