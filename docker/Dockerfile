FROM scratch
ENV PORT 8443
EXPOSE 8443
COPY artifacts/build/release/linux/amd64/honeycomb /bin/honeycomb
COPY docker/etc /etc
WORKDIR /bin
ENTRYPOINT ["honeycomb"]
