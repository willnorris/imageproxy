FROM golang:1.8
MAINTAINER Sevki <s@sevki.org>

ADD . /go/src/willnorris.com/go/imageproxy
RUN go get willnorris.com/go/imageproxy/cmd/imageproxy
RUN useradd -ms /bin/bash go && chown -R go /go

USER go
CMD []
ENTRYPOINT ["/go/bin/imageproxy"]

EXPOSE 8080
