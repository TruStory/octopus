FROM alpine:3.9
RUN apk add --update --no-cache ca-certificates
COPY ./bin/pushd /usr/bin/pushd
ENTRYPOINT /usr/bin/pushd