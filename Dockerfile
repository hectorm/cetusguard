# syntax=docker.io/docker/dockerfile:1

##################################################
## "build" stage
##################################################

FROM --platform=${BUILDPLATFORM:-linux/amd64} docker.io/golang:1.19.1-bullseye@sha256:c41345ea7f7b10a980cb3a8d67943a0c5ca2c4f8bf277f523e09ffee2ea6241d AS build

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

WORKDIR /src/
COPY ./go.mod ./go.sum ./
RUN go mod download
COPY ./ ./
RUN make test
RUN make build \
		GOOS="${TARGETOS-}" \
		GOARCH="${TARGETARCH-}" \
		GOARM="$([ "${TARGETARCH-}" != 'arm' ] || printf '%s' "${TARGETVARIANT#v}")"
RUN test -z "$(readelf -x .interp ./dist/cetusguard-* 2>/dev/null)"

##################################################
## "main" stage
##################################################

FROM scratch AS main

COPY --from=build /src/dist/cetusguard-* /bin/cetusguard

ENV CETUSGUARD_FRONTEND_ADDR='tcp://:2375'

ENTRYPOINT ["/bin/cetusguard"]
