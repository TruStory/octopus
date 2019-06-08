FROM alpine:3.9
RUN apk add --update --no-cache ca-certificates
COPY ./bin/truapid /usr/bin/truapid
ENTRYPOINT /usr/bin/truapid