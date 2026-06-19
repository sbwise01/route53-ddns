FROM golang:1.24-alpine3.21 AS build
ARG OS
ARG ARCH
COPY . /build/
WORKDIR /build
RUN go mod download && go build -o r53ddns

FROM alpine:3.21
ARG VERSION
ARG user=r53ddns
ARG group=r53ddns
ARG uid=1000
ARG gid=1000
USER root
WORKDIR /app
COPY --from=build /build/r53ddns /app/r53ddns
COPY container-entrypoint.sh /app/container-entrypoint.sh
RUN apk update && apk --no-cache add bash ca-certificates && addgroup -g ${gid} ${group} && adduser -h /app -u ${uid} -G ${group} -s /bin/bash -D ${user}
RUN chown r53ddns:r53ddns /app/r53ddns && chmod +x /app/r53ddns && \
    chown r53ddns:r53ddns /app/container-entrypoint.sh && chmod +x /app/container-entrypoint.sh
USER r53ddns
ENTRYPOINT [ "/app/container-entrypoint.sh"]
