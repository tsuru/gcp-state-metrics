FROM golang:1.16

COPY . /src

RUN set -ex \
    && cd /src \
    && CGO_ENABLED=0 GOOS=linux go build -ldflags='-s -w -extldflags "-static"' -o /gcp-state-metrics .

FROM alpine:3.14.0
RUN apk add ca-certificates --no-cache
ENV PORT=8080
EXPOSE 8080
COPY --from=0 /gcp-state-metrics /gcp-state-metrics
ENTRYPOINT ["/gcp-state-metrics"]