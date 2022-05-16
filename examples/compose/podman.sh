#!/bin/sh

set -eu
export LC_ALL='C'

exec podman exec -it cetusguard-podman podman --remote "${@-}"
