package emailcheck

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf16"
)

// Parsed is one normalised address. Error is set when the syntax is invalid.
type Parsed struct {
	Input        string
	Local        string
	Domain       string // lower case, IDN as punycode (xn--)
	Email        string // lower-case local part + "@" + Domain
	UnicodeLocal bool   // non-ASCII characters before @
	Error        string
}

var (
	localRE = regexp.MustCompile("^[A-Za-z0-9!#$%&'*+/=?^_`{|}~-]+(\\.[A-Za-z0-9!#$%&'*+/=?^_`{|}~-]+)*$")
	tldRE   = regexp.MustCompile(`^(xn--[a-z0-9-]+|[a-z]{2,63})$`)
	labelRE = regexp.MustCompile(`^[a-z0-9-]{1,63}$`)
)

// Parse normalises one address (trims, strips mailto: and <>, lower-cases the domain,
// converts an IDN domain to punycode) and checks its syntax.
func Parse(raw string) Parsed {
	input := strings.TrimFunc(raw, unicode.IsSpace)
	if len(input) >= 7 && strings.EqualFold(input[:7], "mailto:") {
		input = input[7:]
	}
	input = strings.TrimPrefix(input, "<")
	input = strings.TrimSuffix(input, ">")
	p := Parsed{Input: input}
	fail := func(msg string) Parsed { p.Error = msg; return p }

	if input == "" {
		return fail("Empty")
	}
	at := strings.LastIndex(input, "@")
	if at < 1 || at == len(input)-1 {
		return fail("Missing @ or missing part")
	}
	local := input[:at]
	domainRaw := strings.TrimSuffix(strings.ToLower(input[at+1:]), ".")
	if strings.Contains(local, "@") {
		return fail("More than one @")
	}
	if strings.IndexFunc(input, unicode.IsSpace) >= 0 {
		return fail("Contains spaces")
	}
	if jsLen(input) > 254 {
		return fail("Longer than 254 characters")
	}
	if jsLen(local) > 64 {
		return fail("Part before @ is longer than 64 characters")
	}

	domain := toASCIIDomain(domainRaw)
	if domain == "" {
		return fail("Domain is not a valid name")
	}
	labels := strings.Split(domain, ".")
	if len(labels) < 2 {
		return fail("Domain has no extension (e.g. .com)")
	}
	for _, l := range labels {
		if !labelRE.MatchString(l) || strings.HasPrefix(l, "-") || strings.HasSuffix(l, "-") {
			return fail("Domain is not a valid name")
		}
	}
	if !tldRE.MatchString(labels[len(labels)-1]) {
		return fail("Domain extension is not valid")
	}
	asciiLocal := isASCII(local)
	if asciiLocal && !localRE.MatchString(local) {
		return fail("Part before @ has characters or dots that are not allowed")
	}

	p.Local, p.Domain = local, domain
	p.Email = strings.ToLower(local) + "@" + domain
	p.UnicodeLocal = !asciiLocal
	return p
}

// jsLen counts UTF-16 code units, like JavaScript's String.length.
func jsLen(s string) int { return len(utf16.Encode([]rune(s))) }

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

func toASCIIDomain(domain string) string {
	if isASCII(domain) {
		return domain
	}
	labels := strings.Split(strings.NewReplacer("。", ".", "．", ".", "｡", ".").Replace(domain), ".")
	for i, l := range labels {
		if !isASCII(l) {
			labels[i] = "xn--" + punycode(l)
		}
	}
	return strings.Join(labels, ".")
}

// punycode encodes one label (RFC 3492).
func punycode(label string) string {
	const base, tMin, tMax, skew, damp = 36, 1, 26, 38, 700
	digit := func(d int) byte {
		if d < 26 {
			return byte('a' + d)
		}
		return byte('0' + d - 26)
	}
	adapt := func(delta, numPoints int, first bool) int {
		if first {
			delta /= damp
		} else {
			delta /= 2
		}
		delta += delta / numPoints
		k := 0
		for ; delta > ((base-tMin)*tMax)/2; k += base {
			delta /= base - tMin
		}
		return k + (base-tMin+1)*delta/(delta+skew)
	}
	cps := []rune(label)
	var out []byte
	for _, c := range cps {
		if c < 0x80 {
			out = append(out, byte(c))
		}
	}
	basic := len(out)
	h, n, delta, bias := basic, 0x80, 0, 72
	if basic > 0 {
		out = append(out, '-')
	}
	for h < len(cps) {
		m := rune(0x7fffffff)
		for _, c := range cps {
			if int(c) >= n && c < m {
				m = c
			}
		}
		delta += (int(m) - n) * (h + 1)
		n = int(m)
		for _, c := range cps {
			if int(c) < n {
				delta++
			}
			if int(c) != n {
				continue
			}
			q := delta
			for k := base; ; k += base {
				t := k - bias
				if k <= bias {
					t = tMin
				} else if k >= bias+tMax {
					t = tMax
				}
				if q < t {
					break
				}
				out = append(out, digit(t+(q-t)%(base-t)))
				q = (q - t) / (base - t)
			}
			out = append(out, digit(q))
			bias = adapt(delta, h+1, h == basic)
			delta = 0
			h++
		}
		delta++
		n++
	}
	return string(out)
}

// SuggestDomain returns the popular mail domain this one is probably a typo of
// (gmial.com → gmail.com), or "" when it does not look like a typo.
func SuggestDomain(domain string) string {
	if freeDomains[domain] {
		return ""
	}
	for _, p := range popularDomains {
		if p == domain {
			return ""
		}
	}
	best, bestDist := "", 3
	for _, p := range popularDomains {
		if d := levenshtein(domain, p); d < bestDist {
			best, bestDist = p, d
		}
	}
	// 1 edit always counts; 2 edits only for longer names (avoids gmx.com -> gmail.com style jumps).
	if best != "" && (bestDist == 1 || (bestDist == 2 && len(domain) >= 9)) {
		return best
	}
	return ""
}

func levenshtein(a, b string) int {
	row := make([]int, len(b)+1)
	for i := range row {
		row[i] = i
	}
	for i := 1; i <= len(a); i++ {
		prev := row[0]
		row[0] = i
		for j := 1; j <= len(b); j++ {
			tmp := row[j]
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			row[j] = min(row[j]+1, row[j-1]+1, prev+cost)
			prev = tmp
		}
	}
	return row[len(b)]
}
