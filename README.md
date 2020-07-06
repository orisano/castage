# castage
castage is multi-stage builds helper for caching.

## Installation
```bash
go get -u github.com/orisano/castage/...
```

## How to use
```
$ cat Dockerfile
FROM golang:1.12-alpine3.9 AS vendor
WORKDIR /var/app/castage
RUN apk add --no-cache git
COPY go.sum go.mod ./
RUN go mod download

FROM golang:1.12-alpine3.9 AS builder

FROM alpine:3.9 AS app

```
```
$ castage -i image_name
set -ex
docker pull image_name:vendor-cache || true
docker build -t image_name:vendor-cache --target=vendor --cache-from=image_name:vendor-cache .
docker pull image_name:builder-cache || true
docker build -t image_name:builder-cache --target=builder --cache-from=image_name:vendor-cache,image_name:builder-cache .
docker pull image_name:app-cache || true
docker build -t image_name:app-cache --target=app --cache-from=image_name:vendor-cache,image_name:builder-cache,image_name:app-cache .
```

## Author
Nao Yonashiro (@orisano)

## License
MIT
