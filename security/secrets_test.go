package security

import (\n\t"sync"
	"os"
	"path/filepath"
	"testing"
)

func TestProtectReveal(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOTIFY_MU_SECRET_KEY_FILE", filepath.Join(dir, "secret.key"))
	keyOnce = sync.Once{}
	keyData = nil
	keyErr = nil

	protected, err := Protect("super-secret")
	if err != nil { t.Fatal(err) }
	if protected == "super-secret" { t.Fatal("secret was not encrypted") }
	revealed, err := Reveal(protected)
	if err != nil { t.Fatal(err) }
	if revealed != "super-secret" { t.Fatalf("unexpected plaintext %q", revealed) }

	info, err := os.Stat(filepath.Join(dir, "secret.key"))
	if err != nil { t.Fatal(err) }
	if info.Mode().Perm() != 0o600 { t.Fatalf("key permissions = %o", info.Mode().Perm()) }
}

func TestRevealLegacyPlaintext(t *testing.T) {
	got, err := Reveal("legacy")
	if err != nil || got != "legacy" { t.Fatalf("got=%q err=%v", got, err) }
}

func TestWebhookVerifier(t *testing.T) {
	a := WebhookVerifier("secret")
	b := WebhookVerifier("secret")
	if a != b || a == "secret" || !IsWebhookVerifier(a) { t.Fatal("invalid verifier") }
}
