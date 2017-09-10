FROM golang:1.8
MAINTAINER Sevki <s@sevki.org>

WORKDIR /go/src/willnorris.com/go/imageproxy
ADD . .

WORKDIR /go/src/willnorris.com/go/imageproxy/cmd/imageproxy
RUN go-wrapper download
RUN CGO_ENABLED=0 GOOS=linux go-wrapper install

FROM scratch

WORKDIR /go/bin

COPY --from=0 /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=0 /etc/ssl/certs /etc/ssl/certs
COPY --from=0 /go/bin/imageproxy .

CMD ["-addr", "0.0.0.0:8080"]
ENTRYPOINT ["/go/bin/imageproxy"]

EXPOSE 8080
