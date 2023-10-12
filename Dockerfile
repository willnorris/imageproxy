FROM golang:1.15 as build

WORKDIR /go/src/willnorris.com/go/imageproxy
ADD . .

#WORKDIR /go/src/willnorris.com/go/imageproxy/cmd/imageproxy

COPY go.mod go.sum ./
COPY third_party/envy/go.mod ./third_party/envy/
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go install -v ./cmd/imageproxy

FROM alpine:3.14
RUN apk update && apk add pngquant jpegoptim libwebp-tools

COPY --from=build /etc/passwd /etc/passwd

WORKDIR /go/bin

COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=build /etc/ssl/certs /etc/ssl/certs
COPY --from=build /go/bin/imageproxy .
COPY --from=build /go/src/willnorris.com/go/imageproxy/assets /assets

CMD ["-addr", "0.0.0.0:8080"]
CMD ["-verbose", "true"]
ENTRYPOINT ["/go/bin/imageproxy", "-scaleUp"]

EXPOSE 8080
