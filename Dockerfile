# syntax=docker.io/docker/dockerfile:1

##################################################
## "build" stage
##################################################

FROM docker.io/golang:1.18-bullseye AS build

WORKDIR /src/
COPY ./Makefile ./
COPY ./go.mod ./go.sum ./
RUN go mod download
COPY ./ ./
RUN make test
RUN make build
RUN ./dist/cetusguard-* -version
RUN test -z "$(readelf -x .interp ./dist/cetusguard-* 2>/dev/null)"

##################################################
## "main" stage
##################################################

FROM scratch AS main

COPY --from=build /src/dist/cetusguard-* /bin/cetusguard

ENV CETUSGUARD_FRONTEND_ADDR='tcp://:2375'

ENTRYPOINT ["/bin/cetusguard"]
