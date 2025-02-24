# build stage
FROM golang:alpine AS builder

ARG PROGRAM_VER=dev-docker

## build gcsfuse v0.41.6 is latest release at point of writing (2022-09-11)
ARG GCSFUSE_VERSION=v0.41.6

# RUN go install github.com/googlecloudplatform₩/gcsfuse@${GCSFUSE_VERSION}

# RUN apt-get -qq update && \
#     apt-get install -yqq upx

COPY src /build
COPY scripts /build/scripts

WORKDIR /build

RUN go build -ldflags "-X main.programVer=${PROGRAM_VER}" -o app
# RUN strip /build/app
# RUN upx -q -9 /build/app

# ---
FROM alpine:3.16

RUN apk add --update --no-cache \
  dcron \
  tar

COPY --from=builder /build/app /app/app
COPY --from=builder /build/configs /app/configs

EXPOSE 8080

COPY --from=builder /build/scripts/start.sh /app/scripts/start.sh
RUN chmod +x /app/scripts/start.sh

WORKDIR /app
# ENTRYPOINT ["./app"]
ENTRYPOINT ["/app/scripts/start.sh"]
