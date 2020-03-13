FROM golang:1.13
MAINTAINER Sevki <s@sevki.org>

ADD . /go/src/willnorris.com/go/imageproxy
RUN go get willnorris.com/go/imageproxy/cmd/imageproxy

CMD []
ENTRYPOINT /go/bin/imageproxy -addr 0.0.0.0:80 -contentTypes=image/png,image/jpg,image/gif,image/webp,image/jpeg,image/pjpeg,image/tiff,image/bmp

EXPOSE 80
