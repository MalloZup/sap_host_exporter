# this is the what ends up in the RPM "Version" field and it is also used as suffix for the built binaries
# if you want to commit to OBS it must be a remotely available Git reference
VERSION ?= $(shell git rev-parse --short HEAD)

# we only use this to comply with RPM changelog conventions at SUSE
AUTHOR ?= shap-staff@suse.de

# you can customize any of the following to build forks
OBS_PROJECT ?= server:monitoring
OBS_PACKAGE ?= prometheus-sap_host_exporter
REPOSITORY ?= SUSE/sap_host_exporter

# the Go archs we crosscompile to
ARCHS ?= amd64 arm64 ppc64le s390x

default: clean download mod-tidy generate fmt vet-check test build

download:
	go mod download
	go mod verify

build: amd64

build-all: clean-bin $(ARCHS)

$(ARCHS):
	@mkdir -p build/bin
	CGO_ENABLED=0 GOOS=linux GOARCH=$@ go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o build/bin/sap_host_exporter-$(VERSION)-$@

install:
	go install

static-checks: vet-check fmt-check

vet-check: download
	go vet ./...

fmt:
	go fmt ./...

mod-tidy:
	go mod tidy

fmt-check:
	.ci/go_lint.sh

generate:
	go generate ./...

test: download
	go test -v ./...

coverage: coverage.out
coverage.out:
	go test -cover -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

clean: clean-bin clean-obs
	go clean
	rm -f coverage.out

clean-bin:
	rm -rf build/bin

clean-obs:
	rm -rf build/obs

obs-workdir: build/obs
build/obs:
	osc checkout $(OBS_PROJECT)/$(OBS_PACKAGE) -o build/obs
	rm -f build/obs/*.tar.gz
	cp -rv packaging/obs/* build/obs/
	# we interpolate environment variables in OBS _service file so that we control what is downloaded by the tar_scm source service
	sed -i 's~%%VERSION%%~$(VERSION)~' build/obs/_service
	sed -i 's~%%REPOSITORY%%~$(REPOSITORY)~' build/obs/_service
	cd build/obs; osc service runall
	.ci/gh_release_to_obs_changeset.py $(REPOSITORY) -a $(AUTHOR) -t $(VERSION) -f build/obs/$(OBS_PACKAGE).changes || true

obs-commit: obs-workdir
	cd build/obs; osc addremove
	cd build/obs; osc commit -m "Update to git ref $(VERSION)"

.PHONY: default download install static-checks vet-check fmt fmt-check mod-tidy generate test clean clean-bin clean-obs build build-all obs-commit obs-workdir $(ARCHS)
