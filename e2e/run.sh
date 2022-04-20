#!/bin/sh

set -eu
export LC_ALL='C'

SCRIPT_DIR="$(CDPATH='' cd -- "$(dirname -- "${0:?}")" && pwd -P)"

CLI_TREEISH='e7c0659ca607d5a6deb42f3c7f8e81ee77b54c10'
CLI_REMOTE='https://github.com/docker/cli.git'
CLI_PATCH="${SCRIPT_DIR:?}/cli.patch"
CLI_DIR="$(mktemp -d)"

CONTAINER_LOG_TAG="e2e-$(date -u +'%Y%m%d%H%M%S')"

cleanup() { ret="$?"; rm -rf "${CLI_DIR:?}"; trap - EXIT; exit "${ret:?}"; }
trap cleanup EXIT TERM INT HUP

main() {
	git clone "${CLI_REMOTE:?}" "${CLI_DIR:?}"
	git -C "${CLI_DIR:?}" checkout "${CLI_TREEISH:?}"
	git -C "${CLI_DIR:?}" apply -v "${CLI_PATCH:?}"
	printf 'CONTAINER_LOG_TAG=%s\n' "${CONTAINER_LOG_TAG:?}" > "${CLI_DIR:?}"/e2e/.env

	docker build --tag localhost.test/cetusguard:"${CONTAINER_LOG_TAG:?}" "${SCRIPT_DIR:?}"/../
	( cd "${CLI_DIR:?}"; make -f "${CLI_DIR:?}"/docker.Makefile test-e2e-non-experimental; ) || ret="$?"

	journalctl --no-pager --output=cat CONTAINER_TAG="${CONTAINER_LOG_TAG:?}"
	test -n "$(journalctl --output=cat CONTAINER_TAG="${CONTAINER_LOG_TAG:?}" | head -n1)"
	test -z "$(journalctl --output=cat CONTAINER_TAG="${CONTAINER_LOG_TAG:?}" | grep -v '^\(WARNING\|INFO\|DEBUG\):')"

	exit "${ret:-0}"
}

main "${@-}"
