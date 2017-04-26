FROM scratch

HEALTHCHECK --interval=15s --timeout=500ms CMD ["/app/bin/healthcheck"]
ENTRYPOINT ["/app/bin/honeycomb"]

EXPOSE 8443
EXPOSE 8080

ENV GODEBUG netdns=cgo

COPY artifacts/cacert.pem /app/etc/ca-bundle.pem
COPY artifacts/build/release/linux/amd64/* /app/bin/
