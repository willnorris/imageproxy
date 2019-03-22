FROM golang:1.9 as build
MAINTAINER Will Norris <will@willnorris.com>

RUN useradd -u 1001 go

WORKDIR /go/src/willnorris.com/go/imageproxy
ADD . .

WORKDIR /go/src/willnorris.com/go/imageproxy/cmd/imageproxy
RUN go-wrapper download
RUN CGO_ENABLED=0 GOOS=linux go-wrapper install

FROM scratch

WORKDIR /go/bin

COPY --from=build /etc/passwd /etc/passwd
COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /go/bin/imageproxy .

USER go

CMD ["-addr", "0.0.0.0:8080"]
ENTRYPOINT ["/go/bin/imageproxy"]

EXPOSE 8080
