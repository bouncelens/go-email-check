// Code generated from site/assets/engine.js by scripts/build-go.mjs. DO NOT EDIT.

package emailcheck

// roleLocal are shared-inbox local parts (info@, sales@, …).
var roleLocal = setOf("abuse", "admin", "administrator", "all", "billing", "contact", "contactus", "customerservice", "dev", "devnull", "enquiries", "enquiry", "feedback", "finance", "hello", "help", "helpdesk", "hostmaster", "hr", "info", "inquiries", "inquiry", "jobs", "legal", "mail", "mailer-daemon", "marketing", "media", "newsletter", "no-reply", "noc", "noreply", "null", "office", "orders", "postmaster", "press", "privacy", "recruitment", "root", "sales", "security", "service", "spam", "support", "team", "test", "webmaster", "welcome", "careers", "accounts", "accounting", "partners")

// freeDomains are free mailbox providers.
var freeDomains = setOf("gmail.com", "googlemail.com", "yahoo.com", "yahoo.co.in", "yahoo.co.uk", "ymail.com", "rocketmail.com", "outlook.com", "hotmail.com", "hotmail.co.uk", "live.com", "msn.com", "icloud.com", "me.com", "mac.com", "aol.com", "proton.me", "protonmail.com", "pm.me", "zoho.com", "zohomail.in", "gmx.com", "gmx.de", "gmx.net", "web.de", "mail.com", "yandex.com", "yandex.ru", "mail.ru", "rediffmail.com", "tutanota.com", "tuta.io", "fastmail.com", "hey.com", "qq.com", "163.com", "126.com", "naver.com", "daum.net", "libero.it", "orange.fr", "free.fr", "laposte.net", "t-online.de")

// popularDomains are the domains people most often mistype. Typo suggestions only point at these.
var popularDomains = []string{"gmail.com", "yahoo.com", "hotmail.com", "outlook.com", "icloud.com", "aol.com", "live.com", "msn.com", "yahoo.co.in", "rediffmail.com", "protonmail.com", "proton.me", "zoho.com", "gmx.com", "yandex.com", "mail.com", "hotmail.co.uk", "yahoo.co.uk", "googlemail.com", "me.com"}

// providers maps an MX host suffix to the mail provider. Order matters: gateways before mailbox hosts.
var providers = [][2]string{
	{"pphosted.com", "Proofpoint"},
	{"ppe-hosted.com", "Proofpoint"},
	{"mimecast.com", "Mimecast"},
	{"barracudanetworks.com", "Barracuda"},
	{"iphmx.com", "Cisco Secure Email"},
	{"messagelabs.com", "Broadcom MessageLabs"},
	{"trendmicro.com", "Trend Micro"},
	{"google.com", "Google"},
	{"googlemail.com", "Google"},
	{"outlook.com", "Microsoft"},
	{"hotmail.com", "Microsoft"},
	{"yahoodns.net", "Yahoo"},
	{"zoho.com", "Zoho"},
	{"zoho.in", "Zoho"},
	{"zoho.eu", "Zoho"},
	{"protonmail.ch", "Proton"},
	{"messagingengine.com", "Fastmail"},
	{"icloud.com", "Apple iCloud"},
	{"duck.com", "DuckDuckGo relay"},
	{"mx.cloudflare.net", "Cloudflare Email Routing"},
	{"secureserver.net", "GoDaddy"},
	{"titan.email", "Titan"},
	{"yandex.net", "Yandex"},
	{"mail.ru", "Mail.ru"},
	{"amazonaws.com", "Amazon SES/WorkMail"},
	{"emailsrvr.com", "Rackspace"},
}

// gateways are security gateways in front of the real mailbox host.
var gateways = setOf("Proofpoint", "Mimecast", "Barracuda", "Cisco Secure Email", "Broadcom MessageLabs", "Trend Micro")

func setOf(items ...string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, s := range items {
		m[s] = true
	}
	return m
}
