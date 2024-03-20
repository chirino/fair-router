FROM docker.io/library/golang:1.22-alpine as build

WORKDIR /src
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build \
    -ldflags="-extldflags=-static" \
    -o ./dist/router ./cmd/router

RUN CGO_ENABLED=0 go build \
    -ldflags="-extldflags=-static" \
    -o ./dist/worker ./cmd/worker

FROM alpine:3.16 as alpine
COPY --from=build /src/dist/router /bin/router
COPY --from=build /src/dist/worker /bin/worker
COPY ./hack/update-ca.sh /update-ca.sh

FROM alpine:3.12
EXPOSE 8080
