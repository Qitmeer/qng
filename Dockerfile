FROM golang:1.18.3-alpine3.16 AS base
WORKDIR /qng

COPY . /qng

RUN apk add --update git && apk add linux-headers && apk add --update gcc && \
    apk add musl-dev && apk add --update make

RUN make

FROM alpine:latest

WORKDIR /qng
COPY --from=base /qng/build/bin/qng /qng/

CMD ["./qng"]

