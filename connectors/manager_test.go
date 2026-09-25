package connectors

import (
	"net"
	"strings"
	"testing"
)

func TestSplitAddresses(t *testing.T) {
	got := splitAddresses("one@example.com; two@example.com\nthree@example.com")
	if len(got) != 3 { t.Fatalf("expected 3 addresses, got %d", len(got)) }
}

func TestSanitizeHeader(t *testing.T) {
	got := sanitizeHeader("subject\r\nBcc: victim@example.com")
	if strings.ContainsAny(got, "\r\n") { t.Fatalf("header injection was not removed: %q", got) }
}

func TestParseRSSAndAtom(t *testing.T) {
	rss := []byte(`<rss><channel><item><title>Alert</title><link>https://example.test/a</link><guid>a1</guid><description>Body</description></item></channel></rss>`)
	items, err := parseFeed(rss)
	if err != nil || len(items) != 1 || items[0].Key != "a1" || items[0].Title != "Alert" {
		t.Fatalf("unexpected RSS parse: %#v err=%v", items, err)
	}

	atom := []byte(`<feed xmlns="http://www.w3.org/2005/Atom"><entry><id>b1</id><title>Entry</title><summary>Body</summary><link href="https://example.test/b"/></entry></feed>`)
	items, err = parseFeed(atom)
	if err != nil || len(items) != 1 || items[0].Key != "b1" || items[0].Title != "Entry" {
		t.Fatalf("unexpected Atom parse: %#v err=%v", items, err)
	}
}

func TestParseSyslogPRI(t *testing.T) {
	facility, severity, body := parseSyslogPRI("<34>Oct 11 host app: failure")
	if facility != 4 || severity != 2 || body != "Oct 11 host app: failure" {
		t.Fatalf("unexpected syslog parse: facility=%d severity=%d body=%q", facility, severity, body)
	}
}

func TestIPAllowed(t *testing.T) {
	ip := net.ParseIP("192.168.10.25")
	if !ipAllowed(ip, "192.168.10.0/24") { t.Fatal("expected CIDR match") }
	if ipAllowed(ip, "10.0.0.0/8") { t.Fatal("unexpected CIDR match") }
	if !ipAllowed(ip, "") { t.Fatal("empty CIDR restriction should allow") }
}

func TestValidateConnectorURL(t *testing.T) {
	if err := ValidateConnectorURL("https://example.test/feed"); err != nil { t.Fatal(err) }
	if err := ValidateConnectorURL("file:///etc/passwd"); err == nil { t.Fatal("file URL should be rejected") }
}
