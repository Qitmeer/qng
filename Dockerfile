FROM golang:1.20.8-alpine3.18 AS base
WORKDIR /qng

COPY . /qng

RUN apk add --update git && apk add linux-headers && apk add --update gcc && \
    apk add musl-dev && apk add --update make

RUN DEV=dev-docker make

FROM alpine:latest

WORKDIR /qng
COPY --from=base /qng/build/bin/qng /qng/

CMD ["./qng"]

