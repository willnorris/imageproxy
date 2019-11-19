FROM golang:1.9 as build
MAINTAINER Will Norris <will@willnorris.com>

WORKDIR /go/src/willnorris.com/go/imageproxy
ADD . .

WORKDIR /go/src/willnorris.com/go/imageproxy/cmd/imageproxy
RUN go-wrapper download
RUN CGO_ENABLED=0 GOOS=linux go-wrapper install

FROM alpine:3.8
RUN apk update && apk add pngquant jpegoptim libwebp-tools

WORKDIR /go/bin


COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=build /etc/ssl/certs /etc/ssl/certs
COPY --from=build /go/bin/imageproxy .

CMD ["-addr", "0.0.0.0:8080"]
ENTRYPOINT ["/go/bin/imageproxy"]

EXPOSE 8080
