# syntax=docker.io/docker/dockerfile:1

##################################################
## "build" stage
##################################################

FROM --platform=${BUILDPLATFORM:-linux/amd64} docker.io/golang:1.18.2-bullseye@sha256:a966d36a1bdf8769ebfd18eaf1b2cdfc86dfda1a2c61791ccd36ed8237e43a69 AS build

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src/
COPY ./go.mod ./go.sum ./
RUN go mod download
COPY ./ ./
RUN make test
RUN make build GOOS="${TARGETOS-}" GOARCH="${TARGETARCH-}"
RUN test -z "$(readelf -x .interp ./dist/cetusguard-* 2>/dev/null)"

##################################################
## "main" stage
##################################################

FROM scratch AS main

COPY --from=build /src/dist/cetusguard-* /bin/cetusguard

ENV CETUSGUARD_FRONTEND_ADDR='tcp://:2375'

ENTRYPOINT ["/bin/cetusguard"]
