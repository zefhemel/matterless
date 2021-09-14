PACKAGE=github.com/zefhemel/matterless
DOCKER_RUNNER_IMAGE=

install:
	go install ${PACKAGE}/cmd/mls

test:
	go test -cover ${PACKAGE}/pkg/...

test-short:
	go test -short ${PACKAGE}/pkg/...

vet:
	go vet ./...

docker-node-function:
	docker build -t zefhemel/mls-node-function runners/node-function

docker-python3-job:
	docker build -t zefhemel/mls-python3-job runners/python3-job

docker: docker-node-function docker-python3-job

docker-push: docker-node-function-push docker-python3-job-push

docker-node-function-push:
	docker buildx build --platform linux/amd64,linux/arm64 -t zefhemel/mls-node-function --push runners/node-function
	docker build -t zefhemel/mls-node-function runners/node-function

docker-python3-job-push:
	docker buildx build --platform linux/amd64,linux/arm64 -t zefhemel/mls-python3-job --push runners/python3-job
	docker build -t zefhemel/mls-python3-job runners/python3-job
