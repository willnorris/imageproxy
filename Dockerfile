FROM golang:1.9
MAINTAINER Sevki <s@sevki.org>

ADD . /go/src/willnorris.com/go/imageproxy
RUN go get willnorris.com/go/imageproxy/cmd/imageproxy

CMD []
ENTRYPOINT /go/bin/imageproxy -addr 0.0.0.0:80

EXPOSE 80
