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

RUN CGO_ENABLED=0 go test \
    -ldflags="-extldflags=-static" \
    -c -o ./dist/router-test ./internal/tests

FROM alpine:3.16 as alpine
COPY --from=build /src/dist/router /bin/router
COPY --from=build /src/dist/worker /bin/worker
COPY --from=build /src/dist/router-test /bin/router-test

EXPOSE 8080
