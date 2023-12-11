# syntax=docker.io/docker/dockerfile:1

##################################################
## "build" stage
##################################################

FROM --platform=${BUILDPLATFORM:-linux/amd64} docker.io/golang:1.21.5-bookworm@sha256:2d3b13c2a6368032e9697e64e7c923184e6e3be03cf01eadff27de124114e64e AS build

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
