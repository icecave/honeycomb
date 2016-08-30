FROM scratch
ENV PATH /bin:$PATH
ENV CERTIFICATE_PATH="/etc/certificates"
COPY artifacts/build/release/linux/amd64/honeycomb /bin/honeycomb
COPY artifacts/certificates /etc/certificates
ENTRYPOINT ["honeycomb"]
