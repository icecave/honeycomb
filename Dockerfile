FROM scratch

ADD http://curl.haxx.se/ca/cacert.pem /etc/ssl/certs/

HEALTHCHECK --interval=15s --timeout=500ms CMD ["/app/bin/healthcheck"]
ENTRYPOINT ["/app/bin/honeycomb"]

EXPOSE 8443
EXPOSE 8080

ENV GODEBUG netdns=cgo

COPY artifacts/build/release/linux/amd64/* /app/bin/
