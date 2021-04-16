FROM golang:1.16-stretch as builder
WORKDIR /app
COPY . .
RUN make build
ENTRYPOINT ["make"]

FROM golang:1.16-stretch as runner
ADD https://exiftool.org/Image-ExifTool-12.24.tar.gz /usr/local/src/exiftool.tar.gz
RUN apt-get update && apt-get install -y ffmpeg && ffmpeg -version
RUN cd /usr/local/src && gzip -dc exiftool.tar.gz | tar -xf - && \
    cd Image-ExifTool-12.24 && perl Makefile.PL && make test && make install && cd .. && \
    rm -rf /usr/local/src/exiftool.tar.gz /usr/local/src/Image-ExifTool-12.24
COPY --from=builder /app/mediatidy /usr/local/bin/mediatidy
ENTRYPOINT ["/usr/local/bin/mediatidy"]
