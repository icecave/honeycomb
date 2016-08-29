FROM scratch
ENV PATH /bin:$PATH
COPY artifacts/build/release/linux/amd64/honeycomb /bin/honeycomb
ENTRYPOINT ["honeycomb"]
