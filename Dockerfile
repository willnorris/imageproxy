FROM alpine:3.7

EXPOSE 9091

COPY dist /opt

# ADD ./dist/swagger.json /one/swagger.json
# ADD ./swagger-ui /one/swagger-ui


CMD ["/opt/imageproxy-linux"]