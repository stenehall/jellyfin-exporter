TAG_COMMIT := $(shell git rev-list --abbrev-commit --tags --max-count=1)
GIT_TAG := $(shell git describe --abbrev=0 --tags $(TAG_COMMIT) 2>/dev/null || true)
VERSION := $(shell echo $(GIT_TAG) | sed 's/^.\{1\}//')
DATE := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

build-docker:
	docker build -t stenehall/jellyfin-exporter:$(VERSION) --no-cache --build-arg EXPORTER_VER=$(VERSION) .

publish: build-docker
	docker push stenehall/jellyfin-exporter:$(VERSION)
	docker tag stenehall/jellyfin-exporter:$(VERSION) stenehall/jellyfin-exporter:latest
	docker push stenehall/jellyfin-exporter:latest
