#!/bin/sh

set -eu
export LC_ALL='C'

exec docker exec -it cetusguard-docker docker "${@-}"
