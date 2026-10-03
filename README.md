# go-email-check

[![Go Reference](https://pkg.go.dev/badge/github.com/bouncelens/go-email-check.svg)](https://pkg.go.dev/github.com/bouncelens/go-email-check)

Go library that checks email addresses before you send to them or save them:

- **Typos** in the domain: `jane@gmial.com` → did you mean `jane@gmail.com`?
- **Disposable / throwaway** domains (mailinator.com and ~9,000 more, list built in)
- **Dead domains**: the domain doesn't exist, has no mail server, or accepts no email (null MX)
- **Bad syntax**, **role addresses** (info@, sales@), **free providers**, **duplicates** (Gmail dot and +tag variants count as the same inbox)
- The **mail provider** behind the domain (Google, Microsoft, Proofpoint, …)

No API key, no account, no dependencies outside the standard library. It's the Go version of the free checker at **[bouncelens.com](https://bouncelens.com/)** and returns the same results.

> It never connects to the recipient's mail server, so it can't confirm that a mailbox exists. The best result is `unconfirmed` (format and domain OK).

## Install

```sh
go get github.com/bouncelens/go-email-check
```

## Use

```go
import emailcheck "github.com/bouncelens/go-email-check"

results := emailcheck.Check(ctx, []string{"jane@gmial.com", "temp@mailinator.com", "jane@gmail.com"})
for _, r := range results {
    fmt.Println(r.Input, r.Status, r.Reasons)
    if r.DidYouMean != nil {
        fmt.Println("  did you mean", *r.DidYouMean)
    }
}
fmt.Printf("%+v\n", emailcheck.Summarize(results))
```

One address (e.g. a sign-up form): `r := emailcheck.CheckOne(ctx, email)`.

| `Status` | Meaning |
|----------|---------|
| `invalid` | Will bounce: bad syntax, domain doesn't exist, no mail server, or null MX |
| `risky` | Disposable domain, likely typo, no MX record, non-English characters before @, or the DNS lookup failed |
| `unconfirmed` | Format and domain OK. The mailbox itself isn't checked |

`Result` marshals to the same JSON as the [BounceLens](https://bouncelens.com/) API and MCP server (`input`, `email`, `status`, `reasons`, `flags`, `did_you_mean`, `provider`, `mx`).

### Options

```go
emailcheck.CheckWith(ctx, emails, emailcheck.Options{
    Concurrency: 4,                         // domains looked up at once (default 8)
    Disposable:  myList,                    // replace the built-in disposable list
    Lookup:      func(ctx context.Context, d string) emailcheck.DomainInfo { … }, // own DNS / cache
})
```

Each unique domain is looked up once per call. Only the **domain** is sent to Cloudflare's DNS-over-HTTPS resolver (1.1.1.1); addresses never leave your machine.

## Self-hosted API (Docker)

`cmd/emailcheck-api` is a small HTTP API around the library, with the same request and response format as the [BounceLens](https://bouncelens.com/) API. The image is built from `scratch` (a few MB, runs as non-root).

```sh
docker run -p 8080:8080 bouncelens/email-check-api
curl 'localhost:8080/api/check?email=jane@gmial.com'
curl -X POST localhost:8080/api/check -d '{"emails":["jane@gmail.com","temp@mailinator.com"]}'
```

| Endpoint | |
|----------|--|
| `GET /api/check?email=…` | one address |
| `POST /api/check` `{"emails": [...]}` | up to 500 addresses (`MAX_EMAILS`) → `summary` + `results` |
| `GET /healthz` | health check |

Environment: `PORT` (default 8080), `MAX_EMAILS` (default 500). Without Docker: `go run ./cmd/emailcheck-api`.

## Tests

```sh
go test ./...
```

Offline: `testdata/dns.json` holds recorded DNS answers, and `testdata/golden.json` holds the BounceLens JavaScript engine's results for the same inputs. The Go code must match them field for field.

## More from BounceLens

- [bouncelens.com](https://bouncelens.com/) – free email list checker in your browser (CSV in, clean list out)
- [MCP server](https://github.com/bouncelens/mcp-email-check) – the same checks for Claude, Cursor and other AI agents
- [Craft CMS plugin](https://bouncelens.com/craft/)

## License

MIT. Disposable-domain list: [disposable-email-domains](https://github.com/disposable-email-domains/disposable-email-domains) (CC0).
