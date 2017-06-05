FROM alpine:3.5

ADD . /go-trustmachine
RUN \
  apk add --update git go make gcc musl-dev linux-headers && \
  (cd go-trustmachine && make gotrust)                           && \
  cp go-trustmachine/build/bin/gotrust /usr/local/bin/           && \
  apk del git go make gcc musl-dev linux-headers          && \
  rm -rf /go-trustmachine && rm -rf /var/cache/apk/*

EXPOSE 8545
EXPOSE 30303
EXPOSE 30303/udp

ENTRYPOINT ["gotrust"]
