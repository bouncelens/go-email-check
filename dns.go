package emailcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// DoHURL is the DNS-over-HTTPS JSON endpoint (Cloudflare 1.1.1.1).
const DoHURL = "https://cloudflare-dns.com/dns-query"

// DomainInfo is what the DNS says about a domain's mail setup.
type DomainInfo struct {
	Exists    bool
	NullMX    bool     // RFC 7505: the domain accepts no email
	MX        []string // mail hosts, best (lowest preference) first
	Provider  string   // mail provider behind the top MX, e.g. "Google"
	AFallback bool     // no MX, but an A record (mail would go to the web server)
	Error     string   // the lookup failed; the result is unknown
}

// DefaultClient is used for DoH lookups when LookupDomain gets a nil client.
var DefaultClient = &http.Client{Timeout: 10 * time.Second}

type dohAnswer struct {
	Status int `json:"Status"`
	Answer []struct {
		Type int    `json:"type"`
		Data string `json:"data"`
	} `json:"Answer"`
}

// dohResolve asks the DoH endpoint for one record type and returns the rcode + answers of that type.
func dohResolve(ctx context.Context, client *http.Client, name, typ string) (int, []string, error) {
	u := DoHURL + "?name=" + url.QueryEscape(name) + "&type=" + typ
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("accept", "application/dns-json")
	res, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return 0, nil, fmt.Errorf("DNS lookup failed (%d)", res.StatusCode)
	}
	var a dohAnswer
	if err := json.NewDecoder(res.Body).Decode(&a); err != nil {
		return 0, nil, err
	}
	want := map[string]int{"MX": 15, "A": 1, "AAAA": 28}[typ]
	var out []string
	for _, x := range a.Answer {
		if x.Type == want {
			out = append(out, x.Data)
		}
	}
	return a.Status, out, nil
}

// LookupDomain looks a domain up once: MX first, then A as the fallback delivery target.
// A nil client uses DefaultClient.
func LookupDomain(ctx context.Context, domain string, client *http.Client) DomainInfo {
	if client == nil {
		client = DefaultClient
	}
	rcode, answers, err := dohResolve(ctx, client, domain, "MX")
	if err != nil {
		return DomainInfo{Error: err.Error()}
	}
	if info, needA := domainInfoFromMX(rcode, answers); !needA {
		return info
	}
	rcode, answers, err = dohResolve(ctx, client, domain, "A")
	if err != nil {
		return DomainInfo{Error: err.Error()}
	}
	if rcode == 3 {
		return DomainInfo{}
	}
	return DomainInfo{Exists: true, AFallback: len(answers) > 0}
}

func domainInfoFromMX(rcode int, answers []string) (DomainInfo, bool) {
	if rcode == 3 {
		return DomainInfo{}, false
	}
	type mx struct {
		pref int
		host string
	}
	var hosts []mx
	for _, d := range answers {
		f := strings.Fields(d)
		h := mx{}
		if len(f) > 0 {
			h.pref, _ = strconv.Atoi(f[0])
		}
		if len(f) > 1 {
			h.host = strings.ToLower(strings.TrimSuffix(f[1], "."))
		}
		hosts = append(hosts, h)
	}
	sort.SliceStable(hosts, func(i, j int) bool { return hosts[i].pref < hosts[j].pref })
	if len(hosts) == 1 && hosts[0].host == "" {
		return DomainInfo{Exists: true, NullMX: true, MX: []string{}}, false
	}
	if len(hosts) == 0 {
		return DomainInfo{}, true
	}
	info := DomainInfo{Exists: true}
	for _, h := range hosts {
		info.MX = append(info.MX, h.host)
	}
	top := hosts[0].host
	for _, p := range providers {
		if top == p[0] || strings.HasSuffix(top, "."+p[0]) {
			info.Provider = p[1]
			break
		}
	}
	return info, false
}
