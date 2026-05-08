# Build context must be the repo root directory:
#   GOOS=linux GOARCH=amd64 make otelcontribcol
#   docker build -f exporter/clickzettaexporter/example/Dockerfile --build-arg TARGETOS=linux --build-arg TARGETARCH=amd64 -t otelcol-clickzetta .
FROM alpine:3 AS certs
RUN apk --no-cache add ca-certificates

FROM scratch
ARG TARGETOS=linux
ARG TARGETARCH=amd64
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY bin/otelcontribcol_${TARGETOS}_${TARGETARCH} /otelcontribcol

EXPOSE 4317 4318 13133

ENTRYPOINT ["/otelcontribcol"]
CMD ["--config=/etc/otelcol/config.yaml"]
