# Self-hosted BounceLens email check API: typos, disposable domains, dead domains. No API key.
# https://bouncelens.com/
FROM golang:1.23-alpine AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /emailcheck-api ./cmd/emailcheck-api

FROM scratch
# CA certificates for the DNS-over-HTTPS lookups (Cloudflare 1.1.1.1).
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /emailcheck-api /emailcheck-api
LABEL org.opencontainers.image.title="BounceLens email check API" \
      org.opencontainers.image.description="Self-hosted API that checks emails for typos, disposable domains and dead domains. No API key." \
      org.opencontainers.image.url="https://bouncelens.com/" \
      org.opencontainers.image.source="https://github.com/bouncelens/go-email-check" \
      org.opencontainers.image.licenses="MIT"
USER 65534:65534
EXPOSE 8080
ENTRYPOINT ["/emailcheck-api"]
