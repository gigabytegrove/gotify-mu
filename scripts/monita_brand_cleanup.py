#!/usr/bin/env python3
from pathlib import Path
import re

ROOT = Path(__file__).resolve().parents[1]

TEXT_SUFFIXES = {
    ".go", ".ts", ".tsx", ".js", ".jsx", ".json", ".md", ".yml", ".yaml",
    ".sh", ".env", ".example", ".mod", ".sum", ".html", ".css", ".scss",
    ".toml", ".txt"
}
TEXT_NAMES = {"Makefile", "Dockerfile", ".gitignore", ".env.example", "renovate.json"}

HISTORICAL_PREFIXES = ("docs/releases/",)
HISTORICAL_FILES = {"CHANGELOG.md", "LICENSE"}
BRANDING_DOC_EXCLUDES = HISTORICAL_FILES | {"README.md", "CONTRIBUTING.md"}

def is_text(path: Path) -> bool:
    rel = path.relative_to(ROOT).as_posix()
    return path.is_file() and (path.suffix.lower() in TEXT_SUFFIXES or path.name in TEXT_NAMES)

def is_historical(rel: str) -> bool:
    return rel in HISTORICAL_FILES or any(rel.startswith(p) for p in HISTORICAL_PREFIXES)

def write(path: str, content: str):
    p = ROOT / path
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(content, encoding="utf-8")

def replace(path: str, old: str, new: str, required: bool = True):
    p = ROOT / path
    text = p.read_text(encoding="utf-8")
    if old not in text:
        if required:
            raise RuntimeError(f"{path}: expected text not found: {old[:120]!r}")
        return
    p.write_text(text.replace(old, new), encoding="utf-8")

def regex_replace(path: str, pattern: str, repl: str):
    p = ROOT / path
    text = p.read_text(encoding="utf-8")
    text2 = re.sub(pattern, repl, text)
    p.write_text(text2, encoding="utf-8")

# 1) Own the Go module path. Keep true upstream dependencies (location/plugin-api) untouched.
for p in ROOT.rglob("*"):
    if not is_text(p):
        continue
    rel = p.relative_to(ROOT).as_posix()
    if is_historical(rel) or rel.startswith(".github/workflows/"):
        continue
    try:
        text = p.read_text(encoding="utf-8")
    except UnicodeDecodeError:
        continue
    text = text.replace("github.com/gotify/server/v3", "github.com/gigabytegrove/monita")
    p.write_text(text, encoding="utf-8")

# 2) Remove stale active product branding, but do not rewrite legal/history/compatibility docs.
for p in ROOT.rglob("*"):
    if not is_text(p):
        continue
    rel = p.relative_to(ROOT).as_posix()
    if rel in BRANDING_DOC_EXCLUDES or rel.startswith(".github/workflows/") or any(rel.startswith(x) for x in HISTORICAL_PREFIXES):
        continue
    try:
        text = p.read_text(encoding="utf-8")
    except UnicodeDecodeError:
        continue
    text = text.replace("Gotify MU", "Monita")
    text = text.replace("Gotify-MU", "Monita")
    text = text.replace("Gotify server", "Monita server")
    text = text.replace("Gotify plugins", "Monita plugins")
    text = text.replace("Gotify priorities", "Monita priorities")
    text = text.replace("Gotify Backup", "Monita Backup")
    p.write_text(text, encoding="utf-8")

# 3) Canonical Monita names for the former MU internals. Legacy routes remain aliases later.
go_symbol_replacements = {
    "MUCapabilitiesAPI": "MonitaCapabilitiesAPI",
    "MUCapabilities": "MonitaCapabilities",
    "MUCapabilityFlags": "MonitaCapabilityFlags",
    "MUPresenceDatabase": "MonitaPresenceDatabase",
    "MUEventNotifier": "MonitaEventNotifier",
    "MUPresenceAPI": "MonitaPresenceAPI",
    "NotifyMUEvent": "NotifyMonitaEvent",
    "muEventClient": "monitaEventClient",
    "newMUEventClient": "newMonitaEventClient",
    "muClients": "monitaClients",
    "removeMU": "removeMonita",
    "registerMU": "registerMonita",
    "HandleMUEvents": "HandleMonitaEvents",
    "MUEventReadError": "MonitaEventReadError",
    "MUEventWriteError": "MonitaEventWriteError",
    "MUEventPingError": "MonitaEventPingError",
    "muCapabilitiesHandler": "monitaCapabilitiesHandler",
    "muPresenceHandler": "monitaPresenceHandler",
    "TestMUCapabilities": "TestMonitaCapabilities",
    "TestMUPresence": "TestMonitaPresence",
    "TestNotifyMUEvent": "TestNotifyMonitaEvent",
}
for p in ROOT.rglob("*.go"):
    rel = p.relative_to(ROOT).as_posix()
    if is_historical(rel):
        continue
    text = p.read_text(encoding="utf-8")
    for old, new in go_symbol_replacements.items():
        text = text.replace(old, new)
    p.write_text(text, encoding="utf-8")

# Rename legacy-named source files to canonical names.
renames = {
    "api/mu_capabilities.go": "api/monita_capabilities.go",
    "api/mu_capabilities_test.go": "api/monita_capabilities_test.go",
    "api/mu_presence.go": "api/monita_presence.go",
    "api/mu_presence_test.go": "api/monita_presence_test.go",
    "api/stream/mu_event_client.go": "api/stream/monita_event_client.go",
    "api/stream/mu_event_test.go": "api/stream/monita_event_test.go",
    "model/gotifyinfo.go": "model/monitainfo.go",
}
for src, dst in renames.items():
    s, d = ROOT / src, ROOT / dst
    if s.exists():
        d.parent.mkdir(parents=True, exist_ok=True)
        s.rename(d)

# 4) Canonical capability identity.
replace("api/monita_capabilities.go", 'Product:    "gotify-mu",', 'Product:    "monita",')
replace(
    "api/monita_capabilities_test.go",
    'payload.Product != "gotify-mu"',
    'payload.Product != "monita"',
)
replace(
    "api/monita_capabilities.go",
    "// MonitaCapabilitiesAPI exposes a stable discovery contract for Monita-aware\n// clients. A stock Gotify server does not expose this route, allowing clients",
    "// MonitaCapabilitiesAPI exposes the canonical Monita discovery contract.\n// Legacy MU routes remain aliases so older clients continue to work.",
    required=False,
)

# 5) Canonical info model and route, with /gotifyinfo retained only as a compatibility alias.
write("model/monitainfo.go", """package model

// MonitaInfo Model
//
// swagger:model MonitaInfo
type MonitaInfo struct {
	// The current version.
	//
	// required: true
	Version string `json:"version"`
	// Whether OIDC authentication is enabled.
	Oidc bool `json:"oidc"`
	// Whether user registration is enabled.
	Register bool `json:"register"`
	// Whether local username/password authentication is enabled.
	LocalAuth bool `json:"localAuth"`
	// Configured OIDC provider display name.
	OIDCIDPName string `json:"oidcIdpName"`
	// Whether OIDC login should start automatically.
	OIDCAutoRedirect bool `json:"oidcAutoRedirect"`
	// Whether LDAP authentication is enabled.
	LDAP bool `json:"ldap"`
	// Configured LDAP provider display name.
	LDAPIDPName string `json:"ldapIdpName"`
}
""")

router = (ROOT / "router/router.go").read_text(encoding="utf-8")
start = router.index("\t// swagger:operation GET /gotifyinfo info getInfo")
end_marker = "\n\tg.GET(\"/application/current\""
end = router.index(end_marker, start)
replacement = """	// swagger:operation GET /monitainfo info getInfo
	//
	// Get Monita information.
	//
	// ---
	// produces: [application/json]
	// responses:
	//   200:
	//     description: Ok
	//     schema:
	//         $ref: "#/definitions/MonitaInfo"
	monitaInfoHandler := func(ctx *gin.Context) {
		ctx.JSON(200, &model.MonitaInfo{
			Version:          vInfo.Version,
			Oidc:             conf.OIDC.Enabled,
			Register:         conf.Registration,
			LocalAuth:        conf.LocalAuthEnabled,
			OIDCIDPName:      conf.OIDC.IDPName,
			OIDCAutoRedirect: conf.OIDC.AutoRedirect,
			LDAP:             conf.LDAP.Enabled,
			LDAPIDPName:      conf.LDAP.IDPName,
		})
	}
	g.GET("monitainfo", monitaInfoHandler)
	// Legacy Gotify endpoint retained for existing clients.
	g.GET("gotifyinfo", monitaInfoHandler)
"""
router = router[:start] + replacement + router[end:]
router = router.replace(
    '\t\tclientAuth.GET("/api/mu/v1/capabilities", monitaCapabilitiesHandler.Get)\n'
    '\t\tclientAuth.GET("/api/mu/v1/events", streamHandler.HandleMonitaEvents)\n',
    '\t\tclientAuth.GET("/api/monita/v1/capabilities", monitaCapabilitiesHandler.Get)\n'
    '\t\tclientAuth.GET("/api/monita/v1/events", streamHandler.HandleMonitaEvents)\n'
    '\t\t// Legacy MU discovery/event routes retained for pre-Monita clients.\n'
    '\t\tclientAuth.GET("/api/mu/v1/capabilities", monitaCapabilitiesHandler.Get)\n'
    '\t\tclientAuth.GET("/api/mu/v1/events", streamHandler.HandleMonitaEvents)\n'
)
(ROOT / "router/router.go").write_text(router, encoding="utf-8")

# 6) Canonical authentication headers/cookies, with legacy fallbacks.
write("auth/cookie.go", """package auth

import "net/http"

// CookieMaxAge is the lifetime of the session cookie in seconds (7 days).
const CookieMaxAge = 7 * 24 * 60 * 60

const (
	CookieName       = "monita-client-token"
	LegacyCookieName = "gotify-client-token"
)

func setCookie(w http.ResponseWriter, name, token string, maxAge int, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

func SetCookie(w http.ResponseWriter, token string, maxAge int, secure bool) {
	setCookie(w, CookieName, token, maxAge, secure)
	if maxAge < 0 {
		// Clear sessions created before the Monita cookie rename.
		setCookie(w, LegacyCookieName, "", maxAge, secure)
	}
}
""")

authp = ROOT / "auth/authentication.go"
auth = authp.read_text(encoding="utf-8")
auth = auth.replace(
    'const (\n\theaderName = "X-Gotify-Key"\n\tcookieName = "gotify-client-token"\n)',
    'const (\n\theaderName          = "X-Monita-Key"\n\tlegacyHeaderName    = "X-Gotify-Key"\n\tmfaHeaderName       = "X-Monita-MFA-Code"\n\tlegacyMFAHeaderName = "X-Gotify-MFA-Code"\n)'
)
auth = auth.replace('code := strings.TrimSpace(ctx.GetHeader("X-Gotify-MFA-Code"))', 'code := MFACodeFromRequest(ctx)')
auth = auth.replace('a.tokenFromXGotifyHeader(ctx)', 'a.tokenFromKeyHeader(ctx)')
auth = auth.replace(
'''func (a *Auth) tokenFromCookie(ctx *gin.Context) string {
	token, err := ctx.Cookie(cookieName)
	if err != nil {
		return ""
	}
	return token
}
''',
'''func (a *Auth) tokenFromCookie(ctx *gin.Context) string {
	if token, err := ctx.Cookie(CookieName); err == nil {
		return token
	}
	if token, err := ctx.Cookie(LegacyCookieName); err == nil {
		return token
	}
	return ""
}
''')
auth = auth.replace(
'''func (a *Auth) tokenFromXGotifyHeader(ctx *gin.Context) string {
	return ctx.Request.Header.Get(headerName)
}
''',
'''func (a *Auth) tokenFromKeyHeader(ctx *gin.Context) string {
	if token := strings.TrimSpace(ctx.Request.Header.Get(headerName)); token != "" {
		return token
	}
	return strings.TrimSpace(ctx.Request.Header.Get(legacyHeaderName))
}

func MFACodeFromRequest(ctx *gin.Context) string {
	if code := strings.TrimSpace(ctx.GetHeader(mfaHeaderName)); code != "" {
		return code
	}
	return strings.TrimSpace(ctx.GetHeader(legacyMFAHeaderName))
}
''')
authp.write_text(auth, encoding="utf-8")

replace("api/session.go", 'code := strings.TrimSpace(ctx.GetHeader("X-Gotify-MFA-Code"))', 'code := auth.MFACodeFromRequest(ctx)')
replace("api/session.go", '\t"strings"\n', '')

service = ROOT / "auth/service.go"
text = service.read_text(encoding="utf-8")
text = text.replace('const serviceAccountContextKey = "gotify-service-account"', 'const serviceAccountContextKey = "monita-service-account"')
text = text.replace(
'if token:=strings.TrimSpace(ctx.GetHeader("X-Gotify-Key"));strings.HasPrefix(token,"SA."){return token}',
'if token:=strings.TrimSpace(ctx.GetHeader("X-Monita-Key"));strings.HasPrefix(token,"SA."){return token}\n\tif token:=strings.TrimSpace(ctx.GetHeader("X-Gotify-Key"));strings.HasPrefix(token,"SA."){return token}'
)
service.write_text(text, encoding="utf-8")

# Canonical MFA header from the Web UI.
for path in ["ui/src/CurrentUser.ts", "ui/src/ElevateStore.ts"]:
    replace(path, "X-Gotify-MFA-Code", "X-Monita-MFA-Code")
replace("ui/src/ElevateStore.ts", "gotify-oidc-elevate", "monita-oidc-elevate")

# Canonical webhook signature headers, while accepting the legacy pair.
automation = ROOT / "api/automation.go"
text = automation.read_text(encoding="utf-8")
text = text.replace(
'''		timestamp := strings.TrimSpace(ctx.GetHeader("X-Gotify-MU-Timestamp"))
		signature := strings.TrimSpace(ctx.GetHeader("X-Gotify-MU-Signature"))
''',
'''		timestamp := strings.TrimSpace(ctx.GetHeader("X-Monita-Timestamp"))
		signature := strings.TrimSpace(ctx.GetHeader("X-Monita-Signature"))
		if timestamp == "" {
			timestamp = strings.TrimSpace(ctx.GetHeader("X-Gotify-MU-Timestamp"))
		}
		if signature == "" {
			signature = strings.TrimSpace(ctx.GetHeader("X-Gotify-MU-Signature"))
		}
''')
automation.write_text(text, encoding="utf-8")

# 7) Canonical message extras namespace, with legacy read compatibility.
push = ROOT / "ui/src/message/PushMessageDialog.tsx"
text = push.read_text(encoding="utf-8")
text = text.replace("item.extras?.['gotify-mu::display']", "item.extras?.['monita::display'] ?? item.extras?.['gotify-mu::display']")
text = text.replace("'gotify-mu::display': {", "'monita::display': {")
push.write_text(text, encoding="utf-8")

extras = ROOT / "ui/src/message/extras.ts"
text = extras.read_text(encoding="utf-8")
text = text.replace("const value = extras?.['gotify-mu::display']?.actions;", "const value = (extras?.['monita::display'] ?? extras?.['gotify-mu::display'])?.actions;")
text = text.replace("const value = extras?.['gotify-mu::display']?.fields;", "const value = (extras?.['monita::display'] ?? extras?.['gotify-mu::display'])?.fields;")
extras.write_text(text, encoding="utf-8")

# 8) Canonical backup names/product with legacy restore compatibility.
backup = ROOT / "operations/backup.go"
text = backup.read_text(encoding="utf-8")
text = text.replace(
'''	ManifestName = "gotify-mu-backup.json"
	PendingRestoreName = ".gotify-mu-restore-pending.zip"
''',
'''	ManifestName              = "monita-backup.json"
	LegacyManifestName        = "gotify-mu-backup.json"
	PendingRestoreName        = ".monita-restore-pending.zip"
	LegacyPendingRestoreName  = ".gotify-mu-restore-pending.zip"
	BackupProduct             = "Monita"
	LegacyBackupProduct       = "Gotify MU"
''')
text = text.replace('Product: "Monita",', 'Product: BackupProduct,')
text = text.replace(
'if filepath.ToSlash(clean) == ManifestName {',
'if filepath.ToSlash(clean) == ManifestName || filepath.ToSlash(clean) == LegacyManifestName {'
)
text = text.replace(
'if !foundManifest || manifest.Product != "Monita" || manifest.FormatVersion != 1 {\n\t\treturn manifest, errors.New("file is not a supported Monita backup")\n\t}',
'if !foundManifest || (manifest.Product != BackupProduct && manifest.Product != LegacyBackupProduct) || manifest.FormatVersion != 1 {\n\t\treturn manifest, errors.New("file is not a supported Monita backup")\n\t}'
)
text = text.replace(
'\t\tpending := filepath.Join(dataDir, PendingRestoreName)',
'\t\tpending := filepath.Join(dataDir, PendingRestoreName)',
)
text = text.replace(
'''	pending := filepath.Join(dataDir, PendingRestoreName)
	if _, err := os.Stat(pending); errors.Is(err, os.ErrNotExist) {
		return "", false, nil
	} else if err != nil {
		return "", false, err
	}
''',
'''	pending := filepath.Join(dataDir, PendingRestoreName)
	if _, err := os.Stat(pending); errors.Is(err, os.ErrNotExist) {
		legacyPending := filepath.Join(dataDir, LegacyPendingRestoreName)
		if _, legacyErr := os.Stat(legacyPending); legacyErr == nil {
			pending = legacyPending
		} else if !errors.Is(legacyErr, os.ErrNotExist) {
			return "", false, legacyErr
		} else {
			return "", false, nil
		}
	} else if err != nil {
		return "", false, err
	}
''')
text = text.replace('relSlash == PendingRestoreName ||', 'relSlash == PendingRestoreName || relSlash == LegacyPendingRestoreName ||')
text = text.replace('if entry.Name() == ManifestName { continue }', 'if entry.Name() == ManifestName || entry.Name() == LegacyManifestName { continue }')
text = text.replace('if relSlash == PendingRestoreName || strings.HasPrefix', 'if (relSlash == PendingRestoreName || relSlash == LegacyPendingRestoreName) || strings.HasPrefix')
text = text.replace('if filepath.ToSlash(filepath.Clean(entry.Name)) == ManifestName { continue }', 'if filepath.ToSlash(filepath.Clean(entry.Name)) == ManifestName || filepath.ToSlash(filepath.Clean(entry.Name)) == LegacyManifestName { continue }')
backup.write_text(text, encoding="utf-8")

# Current backup/diagnostic filenames and messages.
replace("api/system_backup.go", 'os.MkdirTemp("", "gotify-mu-backup-*")', 'os.MkdirTemp("", "monita-backup-*")')
replace("api/system_backup.go", 'filepath.Join(tempDir, "gotify.db")', 'filepath.Join(tempDir, "monita.db")')
replace("api/system_backup.go", '"gotify-mu-backup-" +', '"monita-backup-" +')
replace("api/system_backup.go", 'filename=gotify-mu-diagnostics.json', 'filename=monita-diagnostics.json')

# 9) Docker/build/test internals.
docker = ROOT / "docker/Dockerfile"
text = docker.read_text(encoding="utf-8")
text = text.replace("/src/gotify", "/src/monita")
text = text.replace(
"ARG GOTIFY_SERVER_EXPOSE=80\nENV GOTIFY_SERVER_PORT=$GOTIFY_SERVER_EXPOSE",
"ARG GOTIFY_SERVER_EXPOSE=80\nARG MONITA_SERVER_EXPOSE=$GOTIFY_SERVER_EXPOSE\nENV MONITA_SERVER_PORT=$MONITA_SERVER_EXPOSE"
)
text = text.replace("$GOTIFY_SERVER_PORT/health", "$MONITA_SERVER_PORT/health")
text = text.replace("EXPOSE $GOTIFY_SERVER_EXPOSE", "EXPOSE $MONITA_SERVER_EXPOSE")
docker.write_text(text, encoding="utf-8")

replace("plugin/manager.go", '".gotify-mu-plugin-*.so"', '".monita-plugin-*.so"')
replace("Makefile", "GOTIFY_EXE=../removeme/monita", "MONITA_EXE=../removeme/monita")

# Vite uses canonical env/config first and keeps the old config file/variable as fallback.
write("ui/vite.config.ts", (ROOT / "ui/vite.config.ts").read_text(encoding="utf-8")
    .replace(
'''try {
    process.loadEnvFile('../gotify-server.env');
} catch {
    // file is optional
}

const GOTIFY_SERVER_PORT = process.env.GOTIFY_SERVER_PORT ?? '80';
''',
'''for (const envFile of ['../monita-server.env', '../gotify-server.env']) {
    try {
        process.loadEnvFile(envFile);
        break;
    } catch {
        // file is optional; the legacy name is checked second
    }
}

const MONITA_SERVER_PORT = process.env.MONITA_SERVER_PORT ?? process.env.GOTIFY_SERVER_PORT ?? '80';
''')
    .replace("GOTIFY_SERVER_PORT", "MONITA_SERVER_PORT")
    .replace("process.env.MONITA_SERVER_PORT ?? process.env.MONITA_SERVER_PORT", "process.env.MONITA_SERVER_PORT ?? process.env.GOTIFY_SERVER_PORT")
)

# 10) Test harness/internal naming.
for p in (ROOT / "ui/src/tests").glob("*.ts"):
    text = p.read_text(encoding="utf-8")
    text = text.replace("GotifyTest", "MonitaTest")
    text = text.replace("GOTIFY_", "MONITA_")
    text = text.replace("X-Gotify-Key", "X-Monita-Key")
    text = text.replace("Gotify", "Monita")
    text = re.sub(r"\bgotify\b", "monita", text)
    text = text.replace("gotify.net", "monita.test")
    text = text.replace("gotify://", "monita://")
    p.write_text(text, encoding="utf-8")

for p in (ROOT / "test/oidc").rglob("*"):
    if not p.is_file():
        continue
    try:
        text = p.read_text(encoding="utf-8")
    except UnicodeDecodeError:
        continue
    text = text.replace("Gotify", "Monita")
    text = text.replace("gotify", "monita")
    p.write_text(text, encoding="utf-8")

# Go config tests use canonical MONITA_* names. Legacy behavior is covered explicitly below.
for path in ["config/config_test.go", "config/origin_test.go"]:
    p = ROOT / path
    text = p.read_text(encoding="utf-8")
    text = text.replace("GOTIFY_", "MONITA_")
    text = text.replace("TestGotify", "TestMonita")
    text = text.replace("gotify", "monita")
    text = text.replace("Gotify", "Monita")
    p.write_text(text, encoding="utf-8")

config_test = ROOT / "config/config_test.go"
text = config_test.read_text(encoding="utf-8")
if "TestLegacyGotifyEnvironmentFallback" not in text:
    text += '''

func TestLegacyGotifyEnvironmentFallback(t *testing.T) {
	mode.Set(mode.TestDev)
	t.Setenv("GOTIFY_SERVER_PORT", "9187")

	conf, _ := Get()
	assert.Equal(t, 9187, conf.Server.Port)
}
'''
config_test.write_text(text, encoding="utf-8")

# Auth tests use the canonical header/cookie, with one explicit legacy compatibility case.
auth_test = ROOT / "auth/authentication_test.go"
text = auth_test.read_text(encoding="utf-8")
text = text.replace("X-Gotify-Key", "X-Monita-Key")
text = text.replace("cookieName", "CookieName")
if "TestLegacyGotifyKeyHeaderCompatibility" not in text:
    text += '''

func (s *AuthenticationSuite) TestLegacyGotifyKeyHeaderCompatibility() {
	s.assertHeaderRequest("X-Gotify-Key", "clienttoken", s.auth.RequireClient, 200)
}
'''
auth_test.write_text(text, encoding="utf-8")

# 11) User-facing Web text and active source comments.
active_replacements = {
    "ui/src/admin/SystemAdministration.tsx": {
        "Restart Monita to apply it.": "Restart Monita to apply it.",
    },
    "ui/src/apiAuth.ts": {
        "Gotify server is not reachable": "Monita server is not reachable",
    },
    "ui/src/application/AddApplicationDialog.tsx": {
        "in Gotify\n                                            MU desktop and mobile clients": "in Monita\n                                            desktop and mobile clients",
    },
    "ui/src/audit/Audit.tsx": {},
    "ui/src/integration/Integrations.tsx": {
        "'gotify_mu_test'": "'monita_test'",
    },
    "ui/src/plugin/Plugins.tsx": {},
    "model/message.go": {},
    "model/oidc.go": {},
    "api/ldap.go": {},
    "api/oidc.go": {},
    "api/application.go": {},
}
for path, mapping in active_replacements.items():
    p = ROOT / path
    text = p.read_text(encoding="utf-8")
    for old, new in mapping.items():
        text = text.replace(old, new)
    text = text.replace("Gotify MU", "Monita")
    text = text.replace("Gotify server", "Monita server")
    text = text.replace("Gotify plugins", "Monita plugins")
    text = text.replace("Gotify priorities", "Monita priorities")
    text = text.replace("Gotify session", "Monita session")
    text = text.replace("gotify client token", "Monita client token")
    text = text.replace("gotify client", "Monita client")
    text = text.replace("gotify.", "Monita.")
    p.write_text(text, encoding="utf-8")

# OIDC model examples are Monita-native.
oidc_model = ROOT / "model/oidc.go"
text = oidc_model.read_text(encoding="utf-8")
text = text.replace("gotify://oidc/callback", "monita://oidc/callback")
text = text.replace("client_id=gotify", "client_id=monita")
oidc_model.write_text(text, encoding="utf-8")

# Plugin model examples use the Monita repository.
pluginconf = ROOT / "model/pluginconf.go"
text = pluginconf.read_text(encoding="utf-8")
text = text.replace("github.com/gotify/server/plugin/example/echo", "github.com/gigabytegrove/monita/plugin/example/echo")
text = text.replace("gotify.net", "monita.example")
pluginconf.write_text(text, encoding="utf-8")

# Current server env template examples.
env_example = ROOT / "monita-server.env.example"
text = env_example.read_text(encoding="utf-8")
text = text.replace("# Example: gotify", "# Example: monita")
text = text.replace("gotify://oidc/callback", "monita://oidc/callback")
text = text.replace("https://gotify.example.org", "https://monita.example.org")
text = text.replace("messenger,gotify", "messenger,monita")
text = text.replace("admins,gotify-admins", "admins,monita-admins")
text = text.replace("gotify:secret@tcp(localhost:3306)/gotifydb", "monita:secret@tcp(localhost:3306)/monitadb")
text = text.replace("user=gotify dbname=gotifydb", "user=monita dbname=monitadb")
text = text.replace("CN=gotify,", "CN=monita,")
text = text.replace("/run/secrets/gotify_mu_secret_key", "/run/secrets/monita_secret_key")
env_example.write_text(text, encoding="utf-8")

# 12) Current docs/meta.
write("docs/package.go", """// Package docs Monita REST API.
//
// This is the documentation of the Monita REST API.
//
//	# Authentication
//	Monita uses two primary token types:
//	__clientToken__: a client receives messages and manages account resources.
//	__appToken__: an application sends messages.
//
//	The token can be transmitted in a header named `X-Monita-Key`, in a query parameter named `token`, or
//	through an `Authorization` header with the value prefixed with `Bearer` (for example, `Bearer randomtoken`).
//	For Gotify compatibility, `X-Gotify-Key` remains accepted as a legacy alias.
//	Basic auth is also available for supported login/elevation flows.
//
//	\---
//
//	Found a bug or have some questions? [Create an issue on GitHub](https://github.com/gigabytegrove/monita/issues)
//
//	    Schemes: http, https
//	    Host: localhost
//	    Version: 1.3.7
//	    License: MIT https://github.com/gigabytegrove/monita/blob/master/LICENSE
//
//	    Consumes:
//	    - application/json
//
//	    Produces:
//	    - application/json
//
//	    SecurityDefinitions:
//	       appTokenQuery:
//	          type: apiKey
//	          name: token
//	          in: query
//	       clientTokenQuery:
//	          type: apiKey
//	          name: token
//	          in: query
//	       appTokenHeader:
//	          type: apiKey
//	          name: X-Monita-Key
//	          in: header
//	       clientTokenHeader:
//	          type: apiKey
//	          name: X-Monita-Key
//	          in: header
//	       appTokenAuthorizationHeader:
//	          type: apiKey
//	          name: Authorization
//	          in: header
//	          description: >-
//	              Enter an application token with the `Bearer` prefix, e.g. `Bearer Axxxxxxxxxx`.
//	       clientTokenAuthorizationHeader:
//	          type: apiKey
//	          name: Authorization
//	          in: header
//	          description: >-
//	              Enter a client token with the `Bearer` prefix, e.g. `Bearer Cxxxxxxxxxx`.
//	       basicAuth:
//	          type: basic
//
//	swagger:meta
package docs
""")

replace(
    "CODE_OF_CONDUCT.md",
    "Gotify MU repository maintainers",
    "Monita repository maintainers",
    required=False,
)

# Workflow definitions are updated separately through the repository API because
# GitHub Actions tokens cannot rewrite workflow files without workflow permission.

# 13) Release this complete cleanup separately so an already-running 1.3.6 cannot mask it.
(ROOT / "VERSION").write_text("1.3.7\n", encoding="utf-8")
changelog = ROOT / "CHANGELOG.md"
text = changelog.read_text(encoding="utf-8")
entry = """## 1.3.7 — 2026-10-03

### Complete Monita source identity cleanup

- Migrated the internal Go module/import path from the upstream server path to `github.com/gigabytegrove/monita`.
- Renamed former MU-only source files, types, realtime internals, and capability identity to canonical Monita names.
- Added canonical `/monitainfo` and `/api/monita/v1/*` endpoints while retaining legacy aliases for existing clients.
- Added canonical `X-Monita-Key`, `X-Monita-MFA-Code`, and Monita webhook signature headers with legacy Gotify header fallbacks.
- Moved new session cookies, backup names, message extras, Docker/build paths, test harnesses, and current UI copy to Monita naming.
- Preserved only deliberate compatibility/legal/history references to Gotify.

### Compatibility

- Existing Gotify-style headers, compatibility routes, legacy environment variables, legacy config/database/secret paths, and legacy backup bundles remain accepted.
- External `github.com/gotify/plugin-api` and `github.com/gotify/location` dependencies remain because they are upstream compatibility dependencies.
- Historical changelog/release notes and license attribution are intentionally not rewritten.

"""
if "## 1.3.7 — 2026-10-03" not in text:
    text = text.replace("# Changelog\n\n", "# Changelog\n\n" + entry)
changelog.write_text(text, encoding="utf-8")

write("docs/releases/v1.3.7.md", """# Monita v1.3.7

This release completes the source-level Monita identity cleanup rather than limiting the rename to visible UI surfaces.

## Highlights

- Monita now owns its Go module/import path: `github.com/gigabytegrove/monita`.
- Former MU-prefixed source files/types and realtime internals use Monita names.
- Canonical Monita discovery/info routes are available, with legacy aliases retained for compatibility.
- Canonical Monita authentication, MFA, webhook, cookie, backup, message-extra, test, Docker, and UI naming is now primary.
- Current user-facing strings and active workflow text no longer present Gotify MU as the product.

## Compatibility boundaries

A small number of Gotify references remain intentionally because removing them would break compatibility or legal attribution. These include legacy request headers/routes/config names, legacy on-disk migration paths, the upstream `github.com/gotify/plugin-api` and `github.com/gotify/location` dependencies, required plugin ABI entrypoint names, license attribution, and historical release notes.
""")

print("Monita complete cleanup applied.")
