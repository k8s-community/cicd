APP?=k8sapp
PROJECT?=github.com/k8s-community/${APP}
BUILD_PATH?=cmd/k8sapp
REGISTRY?=gcr.io/containers-206912

# Use the 0.0.0 tag for testing, it shouldn't clobber any release builds
RELEASE?=1.2.3
GOOS?=linux
GOARCH?=amd64

# Namespace: dev, prod, release, cte, username ...
NAMESPACE?=k8s-community

# Infrastructure (dev, stable, test ...) and kube-context for helm
INFRASTRUCTURE?=stable
KUBE_CONTEXT?=${INFRASTRUCTURE}
VALUES?=values-${INFRASTRUCTURE}

CONTAINER_NAME?=${NAMESPACE}-${APP}
CONTAINER_IMAGE?=${REGISTRY}/${CONTAINER_NAME}

COMMIT?=$(shell git rev-parse --short HEAD)
BUILD_TIME?=$(shell date -u '+%Y-%m-%d_%H:%M:%S')

BUILDTAGS=

.PHONY: all
all: build

.PHONY: build
build: clean test certs
	@echo "+ $@"
	CGO_ENABLED=0 GOOS=${GOOS} GOARCH=${GOARCH} \
	go build \
	-ldflags "-s -w -X ${PROJECT}/version.Release=${RELEASE} \
		-X ${PROJECT}/version.Commit=${COMMIT} \
		-X ${PROJECT}/version.BuildTime=${BUILD_TIME}" \
		-o ./bin/${GOOS}-${GOARCH}/${APP} ${PROJECT}/${BUILD_PATH}
	docker build --pull -t $(CONTAINER_IMAGE):$(RELEASE) .

.PHONY: push
push: build
	@echo "+ $@"
	docker push $(CONTAINER_IMAGE):$(RELEASE)

.PHONY: deploy
deploy: push
	@echo "+ $@"
	helm upgrade ${CONTAINER_NAME} -f charts/${VALUES}.yaml charts \
		--kube-context ${KUBE_CONTEXT} --namespace ${NAMESPACE} --version=${RELEASE} -i --wait \
		--set image.registry=${REGISTRY} --set image.name=${CONTAINER_NAME} --set image.tag=${RELEASE}

.PHONY: test
test: clean
	@echo "+ $@"
	go test -v -race ./...

.PHONY: clean
clean:
	@rm -f bin/${GOOS}-${GOARCH}/${APP}
