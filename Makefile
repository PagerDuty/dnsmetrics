APP = dnsmetrics
SOURCEDIR = .

SOURCES := $(shell find $(SOURCEDIR) -name '*.go')

DOCKER_TAG := $(shell bash build/determine_docker_tag.sh)
DOCKER_NAME = quay.io/pagerduty/$(APP):$(DOCKER_TAG)

OS := $(shell uname)

.DEFAULT_GOAL: $(APP)

$(APP): $(SOURCES) .get
	gofmt -w .
	go build -o ${APP} ${SOURCES}
	go test -v ./...

.PHONY: clean container install
.get:
	go get ./...
	@touch .get

clean:
	go clean ./...

install: .get
	go install ./...

container:
ifneq ($(OS),Linux)
	GOOS=linux GOARCH=amd64 go build -o ${APP} ${SOURCES}
endif
ifneq ($(strip $(DOCKER_TAG)),)
	docker build -t ${DOCKER_NAME} .
ifdef TRAVIS
	envsubst < build/dockercfg > ~/.dockercfg
endif
	docker push ${DOCKER_NAME}
endif
