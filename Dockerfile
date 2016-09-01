FROM scratch

EXPOSE 8443

ENV PATH /bin:$PATH
ENV CERTIFICATE_PATH="/etc/certificates"
ENV STATSD_ADDRESS="statsd:8125"

COPY artifacts/build/release/linux/amd64/honeycomb /bin/honeycomb
COPY artifacts/certificates /etc/certificates

ENTRYPOINT ["honeycomb"]
