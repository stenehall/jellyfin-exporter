ARG EXPORTER_VER=0.1.0

FROM golang:1.19-alpine3.17 as builder

RUN apk add --update-cache alpine-sdk upx

WORKDIR /build
ADD go.mod go.sum ./
RUN go mod download

ARG EXPORTER_VER
ADD main.go ./
RUN go build \
        -v \
        -ldflags="-w -s -X 'main.Version=$EXPORTER_VER'" \
        -o /jellyfin_exporter

RUN upx --best --lzma -o /jellyfin_exporter_upx /jellyfin_exporter

# ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

FROM alpine:3.17

ARG EXPORTER_VER

LABEL maintainer="stenehall <stenehall@gmail.com>" \
      org.label-schema.vendor="stenehall" \
      org.label-schema.name="jellyfin-exporter" \
      org.label-schema.url="https://github.com/stenehall/jellyfin-exporter" \
      org.label-schema.description="Jellyfin Prometheus metrics exporter" \
      org.label-schema.version=${EXPORTER_VER}

COPY --from=builder /jellyfin_exporter_upx /usr/bin/jellyfin_exporter

EXPOSE 9453

CMD ["/usr/bin/jellyfin_exporter"]
