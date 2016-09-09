# We're temporarily using Alpine as a base imagine, because there is a bug with
# health check execution such that it always requires a shell.
#
# This issue has been fixed in https://github.com/docker/docker/pull/26208, once
# this makes it into a Docker release we can switch back to "scratch".
FROM alpine:3.4
MAINTAINER James Harris <james.harris@icecave.com.au>

# Likewise, we can switch back to the "exec" form of HEALTHCHECK once the above
# fix is released.
HEALTHCHECK --interval=15s --timeout=500ms CMD /app/bin/healthcheck -check
ENTRYPOINT ["/app/bin/honeycomb"]

EXPOSE 8443

ENV CA_CERT "/app/etc/certificates/ca.crt"
ENV CA_KEY  "/app/etc/certificates/ca.key"

ENV SERVER_CERT "/app/etc/certificates/server.crt"
ENV SERVER_KEY  "/app/etc/certificates/server.key"

COPY artifacts/build/release/linux/amd64/* /app/bin/
COPY artifacts/certificates                /app/etc/certificates
