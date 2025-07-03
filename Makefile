export SHELL              := /usr/bin/env bash -Eeu -o pipefail
export REPO_ROOT          := $(shell git rev-parse --show-toplevel || exit 1)
export REPO_LOCAL_DIR     := ${REPO_ROOT}/.local
export PATH               := ${REPO_LOCAL_DIR}/bin:${REPO_ROOT}/.bin:${PATH}
export REPO_TMP_DIR       := ${REPO_ROOT}/.tmp
export PRE_PUSH           := ${REPO_ROOT}/.git/hooks/pre-push
export GIT_TAG_LATEST     := $(shell git describe --tags --abbrev=0)
export GIT_BRANCH_CURRENT := $(shell git rev-parse --abbrev-ref HEAD)
export GO_MODULE_NAME     := github.com/hakadoriya/ddlctl
export BUILD_VERSION       = $(shell git describe --tags --exact-match HEAD 2>/dev/null || git rev-parse --short HEAD)
export BUILD_REVISION      = $(shell git rev-parse HEAD)
export BUILD_BRANCH        = $(shell git rev-parse --abbrev-ref HEAD | tr / -)
export BUILD_TIMESTAMP     = $(shell git log -n 1 --format='%cI')

.DEFAULT_GOAL := help
.PHONY: help
help: githooks ## display this help documents
	@grep -E '^[0-9a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-40s\033[0m %s\n", $$1, $$2}'

.PHONY: setup
setup: githooks ## Setup tools for development
	# == SETUP =====================================================
	# versenv
	make versenv
	# --------------------------------------------------------------

.PHONY: versenv
versenv:
	# direnv
	direnv allow .
	# golangci-lint
	golangci-lint --version

.PHONY: githooks
githooks:
	@diff -q "${REPO_ROOT}/.githooks/pre-push" "${PRE_PUSH}" || cp -ai "${REPO_ROOT}/.githooks/pre-push" "${PRE_PUSH}"

clean: clean-go clean-golangci-lint ## Clean up cache

clean-go:  ## Clean up cache
	go clean -x -cache -testcache -modcache -fuzzcache

clean-golangci-lint:  ## Clean up lint cache
	golangci-lint cache clean

.PHONY: lint
lint:  ## Run gitleaks, go mod tidy, golangci-lint
	# gitleaks ref. https://github.com/gitleaks/gitleaks
	@if ! command -v gitleaks >/dev/null 2>&1; then \
		printf "\033[31;1m%s\033[0m\n" "gitleaks is not installed: brew install gitleaks" 1>&2; \
		exit 1; \
	fi
	gitleaks detect --source . -v
	# tidy
	go mod tidy
	git diff --exit-code go.mod go.sum
	# lint
	# ref. https://golangci-lint.run/usage/linters/
	golangci-lint run --fix
	git diff --exit-code

.PHONY: credits
credits:  ## Generate CREDITS file
	command -v gocredits || go install github.com/Songmu/gocredits/cmd/gocredits@latest
	gocredits -skip-missing . > CREDITS
	git diff --exit-code

.PHONY: test
test: githooks ## Run go test and display coverage
	# test
	go test -v -race -p=4 -parallel=8 -timeout=300s ./...

.PHONY: test-cover
test-cover: githooks ## Run go test and display coverage
	# test
	go test -v -race -p=4 -parallel=8 -timeout=300s -shuffle=on -cover -coverprofile=./coverage.txt ./...
	go tool cover -func=./coverage.txt

.PHONY: ci
ci: lint credits test ## CI command set

.PHONY: git-push-skip-local-ci
git-push-skip-local-ci:  ## Run git push with skip local CI
	-mv ${REPO_ROOT}/.git/hooks/pre-push{,.bak}
	git push
	-mv ${REPO_ROOT}/.git/hooks/pre-push{.bak,}

.PHONY: git-force-push-skip-local-ci
git-force-push-skip-local-ci:  ## Run git push with skip local CI
	-mv ${REPO_ROOT}/.git/hooks/pre-push{,.bak}
	git push -f
	-mv ${REPO_ROOT}/.git/hooks/pre-push{.bak,}

.PHONY: act-check
act-check:
	@if ! command -v act >/dev/null 2>&1; then \
		printf "\033[31;1m%s\033[0m\n" "act is not installed: brew install act" 1>&2; \
		exit 1; \
	fi

.PHONY: act-go-lint
act-go-lint: act-check ## Run go-lint workflow in act
	act pull_request --container-architecture linux/amd64 -P ubuntu-latest=catthehacker/ubuntu:act-latest -W .github/workflows/go-lint.yml

.PHONY: act-go-test
act-go-test: act-check ## Run go-test workflow in act
	act pull_request --container-architecture linux/amd64 -P ubuntu-latest=catthehacker/ubuntu:act-latest -W .github/workflows/go-test.yml

.PHONY: act-go-vuln
act-go-vuln: act-check ## Run go-vuln workflow in act
	act pull_request --container-architecture linux/amd64 -P ubuntu-latest=catthehacker/ubuntu:act-latest -W .github/workflows/go-vuln.yml

.PHONY: build
build:  ## Run goxz for build
	@command -v goxz >/dev/null || go install github.com/Songmu/goxz/cmd/goxz@latest
	goxz -d "${REPO_TMP_DIR}" -os=linux,darwin,windows -arch=amd64,arm64 -pv "`git describe --tags --abbrev=0`" -trimpath -build-ldflags "-s -w -X github.com/hakadoriya/z.go/buildinfoz.buildVersion=${BUILD_VERSION} -X github.com/hakadoriya/z.go/buildinfoz.buildRevision=${BUILD_REVISION} -X github.com/hakadoriya/z.go/buildinfoz.buildBranch=${BUILD_BRANCH} -X github.com/hakadoriya/z.go/buildinfoz.buildTimestamp=${BUILD_TIMESTAMP}" ./cmd/ddlctl

.PHONY: release
release: ci ## Run goxz and gh release upload
	@command -v goxz >/dev/null || go install github.com/Songmu/goxz/cmd/goxz@latest
	git checkout main
	git checkout "${GIT_TAG_LATEST}"
	-goxz -d "${REPO_TMP_DIR}" -os=linux,darwin,windows -arch=amd64,arm64 -pv "`git describe --tags --abbrev=0`" -trimpath -build-ldflags "-s -w -X github.com/hakadoriya/z.go/buildinfoz.buildVersion=${BUILD_VERSION} -X github.com/hakadoriya/z.go/buildinfoz.buildRevision=${BUILD_REVISION} -X github.com/hakadoriya/z.go/buildinfoz.buildBranch=${BUILD_BRANCH} -X github.com/hakadoriya/z.go/buildinfoz.buildTimestamp=${BUILD_TIMESTAMP}" ./cmd/ddlctl
	-gh release upload "`git describe --tags --abbrev=0`" "${REPO_TMP_DIR}"/*"`git describe --tags --abbrev=0`"*
	git checkout "${GIT_BRANCH_CURRENT}"
