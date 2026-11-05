# syntax=docker.io/docker/dockerfile:1

##################################################
## "build" stage
##################################################

FROM --platform=${BUILDPLATFORM:-linux/amd64} docker.io/golang:1.27.2-trixie@sha256:47667934874913a449b96faa7ef4777fa285d9ec29126ee80ed61c45b7cccf0e AS build

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
ARG SOURCE_DATE_EPOCH

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

WORKDIR /rootfs/
RUN install -DTm 0555 /src/dist/cetusguard-* ./cetusguard

##################################################
## "main" stage
##################################################

FROM scratch AS main

COPY --from=build /rootfs/ /

ENV CETUSGUARD_FRONTEND_ADDR='tcp://:2375'

ENTRYPOINT ["/cetusguard"]
