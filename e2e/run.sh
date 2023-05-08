#!/bin/sh

set -eu
export LC_ALL='C'

SCRIPT_DIR="$(CDPATH='' cd -- "$(dirname -- "${0:?}")" && pwd -P)"

CLI_TREEISH='v23.0.6'
CLI_REMOTE='https://github.com/docker/cli.git'
CLI_PATCH="${SCRIPT_DIR:?}/cli.patch"
CLI_DIR="$(mktemp -d)"

TEST_ID="e2e-$(date -u +'%Y%m%d%H%M%S')"

cleanup() { ret="$?"; rm -rf "${CLI_DIR:?}"; trap - EXIT; exit "${ret:?}"; }
trap cleanup EXIT TERM INT HUP

main() {
	git clone "${CLI_REMOTE:?}" "${CLI_DIR:?}"
	git -C "${CLI_DIR:?}" checkout "${CLI_TREEISH:?}"
	git -C "${CLI_DIR:?}" apply -v "${CLI_PATCH:?}"
	printf 'TEST_ID=%s\n' "${TEST_ID:?}" > "${CLI_DIR:?}"/e2e/.env

	docker build --tag localhost.test/cetusguard:"${TEST_ID:?}" "${SCRIPT_DIR:?}"/../
	( cd "${CLI_DIR:?}"; make -f "${CLI_DIR:?}"/docker.Makefile test-e2e-non-experimental; ) || ret="$?"

	journalctl --no-pager --output=cat CONTAINER_TAG="${TEST_ID:?}"
	test -n "$(journalctl --output=cat CONTAINER_TAG="${TEST_ID:?}" | head -n1)"
	test -z "$(journalctl --output=cat CONTAINER_TAG="${TEST_ID:?}" | grep -v '^\(WARNING\|INFO\|DEBUG\):')"

	exit "${ret:-0}"
}

main "${@-}"
