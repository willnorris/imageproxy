FROM golang:1.15 as build

RUN useradd -u 1001 go

WORKDIR /app

COPY go.mod go.sum ./
COPY third_party/envy/go.mod ./third_party/envy/
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -v ./cmd/imageproxy

FROM alpine:3.14
RUN apk update && apk add pngquant jpegoptim libwebp-tools

COPY --from=build /etc/passwd /etc/passwd
WORKDIR /go/bin

COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo

USER go

COPY --from=build /etc/ssl/certs /etc/ssl/certs
COPY --from=build /go/bin/imageproxy .
COPY --from=build /go/src/willnorris.com/go/imageproxy/indicators-size /indicators-size
COPY --from=build /go/src/willnorris.com/go/imageproxy/assets /assets

CMD ["-addr", "0.0.0.0:8080"]
ENTRYPOINT ["/go/bin/imageproxy"]

EXPOSE 8080
