# We're temporarily using Alpine as a base imagine, because there is a bug with
# health check execution such that it always requires a shell.
#
# This issue has been fixed in https://github.com/docker/docker/pull/26208, once
# this makes it into a Docker release we can switch back to "scratch".
FROM alpine:3.4
MAINTAINER James Harris <james.harris@icecave.com.au>

# Likewise, we can swtich back to the "exec" form of HEALTHCHECK once the above
# fix is released.
HEALTHCHECK --interval=15s --timeout=500ms CMD honeycomb -check
ENTRYPOINT ["honeycomb"]

EXPOSE 8443

ENV PATH             "/bin:$PATH"
ENV CERTIFICATE_PATH "/etc/certificates"
ENV STATSD_ADDRESS   "statsd:8125"

COPY artifacts/build/release/linux/amd64/honeycomb /bin/honeycomb
COPY artifacts/certificates /etc/certificates
