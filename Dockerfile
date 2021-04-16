FROM golang:1.16-stretch as builder
ADD https://exiftool.org/Image-ExifTool-12.24.tar.gz /usr/local/src/exiftool.tar.gz
RUN cd /usr/local/src && gzip -dc exiftool.tar.gz | tar -xf - && \
    cd Image-ExifTool-12.24 && perl Makefile.PL && make test && make install && cd .. && \
    rm -rf /usr/local/src/exiftool.tar.gz /usr/local/src/Image-ExifTool-12.24
WORKDIR /app
COPY . .
RUN make build
ENTRYPOINT ["make"]

FROM golang:1.16-stretch as runner
WORKDIR /app
COPY --from=builder /usr/local/bin/exiftool /usr/local/bin/exiftool
COPY --from=builder /app/mediatidy /usr/local/bin/mediatidy
ENTRYPOINT ["/usr/local/bin/mediatidy"]
