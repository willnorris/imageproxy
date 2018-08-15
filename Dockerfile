FROM alpine:3.7

RUN mkdir /one
ADD ./dist/imageproxy-linux /one/imageproxy
# ADD ./dist/swagger.json /one/swagger.json
# ADD ./swagger-ui /one/swagger-ui

EXPOSE 9091

CMD ["/one/imageproxy"]