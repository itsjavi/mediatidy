FROM golang:1.16-stretch as builder
WORKDIR /app
COPY . .
RUN make build
ENTRYPOINT ["make"]

FROM alpine:3.13 as runner
ENV EXIFTOOL_VERSION=12.24
# add deps
RUN apk add --no-cache perl make
# install exiftool
RUN cd /tmp \
	&& wget https://exiftool.org/Image-ExifTool-${EXIFTOOL_VERSION}.tar.gz \
	&& gzip -dc Image-ExifTool-${EXIFTOOL_VERSION}.tar.gz | tar -xf - \
	&& cd Image-ExifTool-${EXIFTOOL_VERSION} \
	&& perl Makefile.PL \
	&& make test \
	&& make install \
	&& cd .. \
	&& rm -rf Image-ExifTool-${EXIFTOOL_VERSION}
# import mediatidy binary
COPY --from=builder /app/mediatidy /usr/local/bin/mediatidy
ENTRYPOINT ["/usr/local/bin/mediatidy"]
