// Package emailcheck checks email addresses for bad syntax, typos in the domain
// (gmial.com → gmail.com), disposable domains, role addresses, duplicates and
// domains that do not exist or have no mail server. It needs no API key.
//
// It is the Go version of the free checker at https://bouncelens.com/ and gives
// the same results. The only network calls are DNS-over-HTTPS lookups of each
// domain's MX and A records (Cloudflare 1.1.1.1). It never connects to the
// recipient's mail server, so it never confirms that a mailbox exists: the best
// result is StatusUnconfirmed.
//
// To check a whole CSV list in your browser, see https://bouncelens.com/.
//
//	results := emailcheck.Check(ctx, []string{"jane@gmial.com", "temp@mailinator.com"})
//	fmt.Println(results[0].Status, *results[0].DidYouMean) // risky jane@gmail.com
package emailcheck

import (
	"context"
	"strings"
	"sync"
)

// Status of a checked address.
type Status string

const (
	// StatusInvalid will bounce: bad syntax, no domain, no mail server, null MX.
	StatusInvalid Status = "invalid"
	// StatusRisky: disposable, likely typo, no MX record, odd syntax or the DNS lookup failed.
	StatusRisky Status = "risky"
	// StatusUnconfirmed: format and domain OK. The mailbox itself is NOT confirmed.
	StatusUnconfirmed Status = "unconfirmed"
)

// Flags are extra facts about an address. They do not change its Status on their own,
// except Disposable (risky) and Duplicate (counted separately by Summarize).
type Flags struct {
	Disposable bool `json:"disposable"`
	Role       bool `json:"role"`
	Free       bool `json:"free"`
	Duplicate  bool `json:"duplicate"`
	Gateway    bool `json:"gateway"`
}

// Result of checking one address. Nil pointers mean "none".
type Result struct {
	Input      string   `json:"input"`
	Email      *string  `json:"email"`
	Status     Status   `json:"status"`
	Reasons    []string `json:"reasons"`
	Flags      Flags    `json:"flags"`
	DidYouMean *string  `json:"did_you_mean"`
	Provider   *string  `json:"provider"`
	MX         *string  `json:"mx"`
}

// Summary counts a checked list. Invalid + Risky + Unconfirmed + Duplicates == Total:
// a duplicate is only counted as a duplicate when it would otherwise be unconfirmed.
type Summary struct {
	Total       int `json:"total"`
	Invalid     int `json:"invalid"`
	Risky       int `json:"risky"`
	Unconfirmed int `json:"unconfirmed"`
	Duplicates  int `json:"duplicates"`
	Disposable  int `json:"disposable"`
	Role        int `json:"role"`
	Free        int `json:"free"`
	Typos       int `json:"typos"`
}

// Options for CheckWith. The zero value uses DNS over HTTPS and the built-in disposable list.
type Options struct {
	// Lookup resolves one domain. Default: LookupDomain with DoH.
	Lookup func(ctx context.Context, domain string) DomainInfo
	// Disposable domains (lower case). Default: the built-in list (~9,000 domains).
	Disposable map[string]bool
	// Concurrency is the number of domains looked up at once. Default 8.
	Concurrency int
}

// Check checks a list of addresses with the default options. Results keep the input order.
func Check(ctx context.Context, emails []string) []Result {
	return CheckWith(ctx, emails, Options{})
}

// CheckOne checks a single address.
func CheckOne(ctx context.Context, email string) Result {
	return Check(ctx, []string{email})[0]
}

// CheckWith checks a list: each unique domain is looked up once, then every address is evaluated.
func CheckWith(ctx context.Context, emails []string, opt Options) []Result {
	if opt.Lookup == nil {
		opt.Lookup = func(ctx context.Context, d string) DomainInfo { return LookupDomain(ctx, d, nil) }
	}
	if opt.Disposable == nil {
		opt.Disposable = DisposableDomains()
	}
	if opt.Concurrency <= 0 {
		opt.Concurrency = 8
	}

	parsed := make([]Parsed, len(emails))
	var domains []string
	seen := map[string]bool{}
	for i, e := range emails {
		parsed[i] = Parse(e)
		if p := parsed[i]; p.Error == "" && !seen[p.Domain] {
			seen[p.Domain] = true
			domains = append(domains, p.Domain)
		}
	}

	dns := make(map[string]DomainInfo, len(domains))
	var mu sync.Mutex
	var wg sync.WaitGroup
	work := make(chan string)
	for w := 0; w < min(opt.Concurrency, len(domains)); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for d := range work {
				info := opt.Lookup(ctx, d)
				mu.Lock()
				dns[d] = info
				mu.Unlock()
			}
		}()
	}
	for _, d := range domains {
		work <- d
	}
	close(work)
	wg.Wait()

	return evaluateList(parsed, dns, opt.Disposable)
}

// evaluateList flags duplicates (incl. Gmail dot/+tag variants) on the 2nd+ copy.
func evaluateList(parsed []Parsed, dns map[string]DomainInfo, disposable map[string]bool) []Result {
	type first struct{ input, email string }
	seen := map[string]first{}
	out := make([]Result, len(parsed))
	for i, p := range parsed {
		r := Evaluate(p, dns[p.Domain], disposable)
		if p.Error == "" {
			key := mailboxKey(p)
			if f, ok := seen[key]; ok {
				r.Flags.Duplicate = true
				if f.email == p.Email {
					r.Reasons = append(r.Reasons, "Duplicate of "+f.input)
				} else {
					r.Reasons = append(r.Reasons, "Same inbox as "+f.input+" (dots/+tag ignored)")
				}
			} else {
				seen[key] = first{p.Input, p.Email}
			}
		}
		out[i] = r
	}
	return out
}

// Evaluate checks one parsed address against its domain lookup.
func Evaluate(p Parsed, dns DomainInfo, disposable map[string]bool) Result {
	r := Result{Input: p.Input, Status: StatusUnconfirmed, Reasons: []string{}}
	if p.Error != "" {
		r.Status = StatusInvalid
		r.Reasons = append(r.Reasons, p.Error)
		return r
	}
	r.Email = ptr(p.Email)

	suggestion := SuggestDomain(p.Domain)
	if suggestion != "" {
		r.DidYouMean = ptr(p.Local + "@" + suggestion)
	}
	r.Flags.Role = roleLocal[strings.SplitN(strings.ToLower(p.Local), "+", 2)[0]]
	r.Flags.Free = freeDomains[p.Domain]
	// Match the domain or any parent (x.mailinator.com -> mailinator.com).
	labels := strings.Split(p.Domain, ".")
	for i := 0; i < len(labels)-1; i++ {
		if disposable[strings.Join(labels[i:], ".")] {
			r.Flags.Disposable = true
			break
		}
	}

	switch {
	case dns.Error != "":
		r.Status = StatusRisky
		r.Reasons = append(r.Reasons, "Could not look up domain: "+dns.Error)
	case !dns.Exists:
		r.Status = StatusInvalid
		r.Reasons = append(r.Reasons, "Domain does not exist")
	case dns.NullMX:
		r.Status = StatusInvalid
		r.Reasons = append(r.Reasons, "Domain says it accepts no email (null MX)")
	case len(dns.MX) == 0 && !dns.AFallback:
		r.Status = StatusInvalid
		r.Reasons = append(r.Reasons, "Domain has no mail server")
	default:
		if len(dns.MX) > 0 {
			r.MX = ptr(dns.MX[0])
		}
		if dns.Provider != "" {
			r.Provider = ptr(dns.Provider)
		}
		r.Flags.Gateway = gateways[dns.Provider]
		if len(dns.MX) == 0 {
			r.Status = StatusRisky
			r.Reasons = append(r.Reasons, "No MX record; mail would go to the website server")
		}
	}

	if r.Status != StatusInvalid {
		if r.Flags.Disposable {
			r.Status = StatusRisky
			r.Reasons = append(r.Reasons, "Disposable (throwaway) email domain")
		}
		if suggestion != "" {
			r.Status = StatusRisky
			r.Reasons = append(r.Reasons, "Looks like a typo of "+suggestion)
		}
		if p.UnicodeLocal {
			r.Status = StatusRisky
			r.Reasons = append(r.Reasons, "Non-English characters before @; many servers reject these")
		}
	} else if suggestion != "" {
		r.Reasons = append(r.Reasons, "Did you mean "+suggestion+"?")
	}
	if r.Status == StatusUnconfirmed {
		r.Reasons = append(r.Reasons, "Format and domain OK, mailbox not confirmed")
	}
	if r.Flags.Role && r.Status != StatusInvalid {
		r.Reasons = append(r.Reasons, "Role address (shared inbox, lower reply rate)")
	}
	return r
}

// Summarize counts the results of a checked list.
func Summarize(results []Result) Summary {
	s := Summary{Total: len(results)}
	for _, r := range results {
		switch {
		case r.Flags.Duplicate && r.Status == StatusUnconfirmed:
			s.Duplicates++
		case r.Status == StatusInvalid:
			s.Invalid++
		case r.Status == StatusRisky:
			s.Risky++
		default:
			s.Unconfirmed++
		}
		if r.Flags.Disposable {
			s.Disposable++
		}
		if r.Flags.Role {
			s.Role++
		}
		if r.Flags.Free {
			s.Free++
		}
		if r.DidYouMean != nil {
			s.Typos++
		}
	}
	return s
}

// Same-mailbox key: Gmail ignores dots and +tags; most providers ignore +tags.
func mailboxKey(p Parsed) string {
	l := strings.ToLower(p.Local)
	d := p.Domain
	if d == "googlemail.com" {
		d = "gmail.com"
	}
	if i := strings.Index(l, "+"); i > 0 {
		l = l[:i]
	}
	if d == "gmail.com" {
		l = strings.ReplaceAll(l, ".", "")
	}
	return l + "@" + d
}

func ptr(s string) *string { return &s }
