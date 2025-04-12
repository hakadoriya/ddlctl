#!/bin/sh
# shellcheck disable=SC2148

# Define environment variables that are not referenced in the container.
export REPO_ROOT=$(git rev-parse --show-toplevel)
export PATH="${REPO_ROOT}/.local/bin:${REPO_ROOT}/.bin:${PATH}"
export DOCKER_BUILDKIT="1"
export COMPOSE_DOCKER_CLI_BUILD="1"

# Load .env file.
dotenv .versenv.env

[ -e .env ] || touch .env
dotenv .env
