FROM golang:1.12.9-alpine as build
ARG BUILD_VERSION=undefined
WORKDIR /go/src/
COPY . .
RUN GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build -mod=vendor -ldflags "-w -s -X main.VERSION=$BUILD_VERSION" -a -installsuffix cgo -v ./cmd/ratingsapp

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata && \
        cp /usr/share/zoneinfo/Australia/Sydney /etc/localtime && \
        echo "Australia/Sydney" >  /etc/timezone

WORKDIR /usr/local/opt/ratingsapp/
COPY --from=build /go/src/ratingsapp .
COPY ./migrations ./migrations
EXPOSE 8000/tcp
CMD ["/usr/local/opt/ratingsapp/ratingsapp"]