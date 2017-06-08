FROM alpine:3.5

RUN \
  apk add --update go git make gcc musl-dev linux-headers ca-certificates && \
  git clone --depth 1 --branch release/1.6 https://github.com/trust-tech/go-trustmachine && \
  (cd go-trustmachine && make gotrust) && \
  cp go-trustmachine/build/bin/gotrust /gotrust && \
  apk del go git make gcc musl-dev linux-headers && \
  rm -rf /go-trustmachine && rm -rf /var/cache/apk/*

EXPOSE 8545
EXPOSE 30303

ENTRYPOINT ["/gotrust"]
