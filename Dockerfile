FROM golang:1.22 as build
MAINTAINER Will Norris <will@willnorris.com>

RUN apk update && apk add build-base git openssh go-1.21

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -v ./cmd/imageproxy

FROM cgr.dev/chainguard/static:latest

COPY --from=build /app/imageproxy /app/imageproxy

CMD ["-addr", "0.0.0.0:8080"]
ENTRYPOINT ["/app/imageproxy"]

EXPOSE 8080
