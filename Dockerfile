FROM alpine:3.7

EXPOSE 9091

COPY dist /opt

# ADD ./dist/swagger.json /one/swagger.json
# ADD ./swagger-ui /one/swagger-ui

CMD ["-addr", "0.0.0.0:9091"]
ENTRYPOINT [ "/opt/pixie-linux" ]