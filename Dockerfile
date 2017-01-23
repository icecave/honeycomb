FROM scratch

ADD http://curl.haxx.se/ca/cacert.pem /etc/ssl/certs/

HEALTHCHECK --interval=15s --timeout=500ms CMD ["/app/bin/healthcheck"]
ENTRYPOINT ["/app/bin/honeycomb"]

EXPOSE 8443

ENV CERTIFICATE_PATH      ""
ENV CERTIFICATE_S3_BUCKET ""
ENV AWS_ACCESS_KEY_ID     ""
ENV AWS_SECRET_ACCESS_KEY ""
ENV GODEBUG               netdns=cgo

COPY artifacts/build/release/linux/amd64/* /app/bin/
