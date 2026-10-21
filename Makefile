export SHELL              := /usr/bin/env bash -Eeu -o pipefail
export REPO_ROOT          := $(shell git rev-parse --show-toplevel || exit 1)
export REPO_LOCAL_DIR     := ${REPO_ROOT}/.local
export PATH               := ${REPO_LOCAL_DIR}/bin:${REPO_ROOT}/.bin:${PATH}
export REPO_TMP_DIR       := ${REPO_ROOT}/.tmp
# pre-push は worktree からでも共有の .git/hooks に置く。--git-path hooks/pre-push は core.hooksPath に従うので使わない
# (グローバルな core.hooksPath があると、このリポジトリの pre-push をリポジトリの外にコピーしてしまう)
export GIT_COMMON_DIR     := $(shell git rev-parse --path-format=absolute --git-common-dir || exit 1)
export PRE_PUSH           := $(if ${GIT_COMMON_DIR},${GIT_COMMON_DIR}/hooks/pre-push)
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
lint:  ## Run gitleaks, go mod tidy -diff, golangci-lint without modifying the working tree
	# gitleaks ref. https://github.com/gitleaks/gitleaks
	@if ! command -v gitleaks >/dev/null 2>&1; then \
		printf "\033[31;1m%s\033[0m\n" "gitleaks is not installed: brew install gitleaks" 1>&2; \
		exit 1; \
	fi
	gitleaks detect --source . -v
	# tidy
	# make ci (pre-push hook と make release の前) で作業ツリーを書き換えないように、差分を出すだけにする
	go mod tidy -diff
	# lint
	# ref. https://golangci-lint.run/usage/linters/
	# --fix は付けない。付けると、失敗したときに書き換わった作業ツリーが残る (v0.0.18-rc.6 はその作業ツリーからビルドされた)
	golangci-lint run

.PHONY: lint-fix
lint-fix:  ## Run go mod tidy and golangci-lint --fix (modifies the working tree)
	go mod tidy
	golangci-lint run --fix

.PHONY: credits
credits:  ## Generate CREDITS file
	command -v gocredits || go install github.com/Songmu/gocredits/cmd/gocredits@latest
	gocredits -skip-missing . > CREDITS

.PHONY: credits-check
credits-check:  ## Check that CREDITS is up to date without modifying the working tree
	command -v gocredits || go install github.com/Songmu/gocredits/cmd/gocredits@latest
	gocredits -skip-missing . | diff -u CREDITS -

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
ci: lint credits-check test ## CI command set

.PHONY: git-push-skip-local-ci
git-push-skip-local-ci:  ## Run git push with skip local CI
	-mv ${PRE_PUSH}{,.bak}
	git push
	-mv ${PRE_PUSH}{.bak,}

.PHONY: git-force-push-skip-local-ci
git-force-push-skip-local-ci:  ## Run git push with skip local CI
	-mv ${PRE_PUSH}{,.bak}
	git push -f
	-mv ${PRE_PUSH}{.bak,}

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
	goxz -n ddlctl -d "${REPO_TMP_DIR}" -os=linux,darwin,windows -arch=amd64,arm64 -pv "`git describe --tags --abbrev=0`" -trimpath -build-ldflags "-s -w -X github.com/hakadoriya/z.go/buildinfoz.buildVersion=${BUILD_VERSION} -X github.com/hakadoriya/z.go/buildinfoz.buildRevision=${BUILD_REVISION} -X github.com/hakadoriya/z.go/buildinfoz.buildBranch=${BUILD_BRANCH} -X github.com/hakadoriya/z.go/buildinfoz.buildTimestamp=${BUILD_TIMESTAMP}" ./cmd/ddlctl

.PHONY: release
release:  ## Build the release archives from a clean checkout of a tag and upload them (run by .github/workflows/release-assets.yml)
	@command -v goxz >/dev/null || go install github.com/Songmu/goxz/cmd/goxz@latest
	# HEAD がタグそのもので、作業ツリーがクリーンなときだけビルドしてアップロードする。
	# v0.0.18-rc.6 は golangci-lint --fix で書き換わった作業ツリーからビルドされ、INTERVAL 7 DAY を INTERVAL7DAY と出力するバイナリが配布された。
	# - 確認・ビルド・アップロードを 1 行の && の連鎖にする。make -i が無視するのは行ごとの終了コードなので、確認に失敗したらビルドもアップロードもされない
	# - 確認には git status --porcelain を使う。go build がバージョンに +dirty を付ける (vcs.modified) のと同じ基準で、追跡していないファイルも含む
	# - 同じタグの古い成果物を消してからビルドする。ビルドに失敗したときに古い成果物をアップロードしないため
	# - -n ddlctl でアーカイブの名前を固定する。goxz は既定でディレクトリ名を使い、versenv は ddlctl_<version>_<os>_<arch> の名前でダウンロードする
	# - RELEASE_DRY_RUN=true のときはアップロードしない
	tag="$$(git describe --tags --exact-match HEAD)" \
	&& { test -z "$$(git status --porcelain)" || { echo "release: the working tree is not clean" 1>&2; git status --porcelain 1>&2; exit 1; }; } \
	&& rm -f "${REPO_TMP_DIR}"/*"$${tag}"* \
	&& goxz -n ddlctl -d "${REPO_TMP_DIR}" -os=linux,darwin,windows -arch=amd64,arm64 -pv "$${tag}" -trimpath -build-ldflags "-s -w -X github.com/hakadoriya/z.go/buildinfoz.buildVersion=${BUILD_VERSION} -X github.com/hakadoriya/z.go/buildinfoz.buildRevision=${BUILD_REVISION} -X github.com/hakadoriya/z.go/buildinfoz.buildBranch=${BUILD_BRANCH} -X github.com/hakadoriya/z.go/buildinfoz.buildTimestamp=${BUILD_TIMESTAMP}" ./cmd/ddlctl \
	&& if [ "$${RELEASE_DRY_RUN:-}" = true ]; then echo "release: RELEASE_DRY_RUN=true, skip uploading" 1>&2; ls -l "${REPO_TMP_DIR}"/*"$${tag}"*; else gh release upload "$${tag}" "${REPO_TMP_DIR}"/*"$${tag}"*; fi
