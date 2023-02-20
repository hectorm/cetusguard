# syntax=docker.io/docker/dockerfile:1

##################################################
## "build" stage
##################################################

FROM --platform=${BUILDPLATFORM:-linux/amd64} docker.io/golang:1.20.1-bullseye@sha256:c3fbdc381fb6b78325c2a5cc1bf0c288c0d173568fba3f1b8894a51837cccf7f AS build

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
