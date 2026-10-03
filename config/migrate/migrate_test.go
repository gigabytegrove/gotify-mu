package migrate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func runMigrate(t *testing.T, yaml string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yml")
	assert.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))
	got, err := Config(path)
	assert.NoError(t, err)
	return got
}

func TestMigrateConfigAllOptions(t *testing.T) {
	yaml := `
server:
  keepaliveperiodseconds: 30
  listenaddr: 0.0.0.0
  port: 8080
  ssl:
    enabled: true
    redirecttohttps: false
    listenaddr: 127.0.0.1
    port: 8443
    certfile: /cert.pem
    certkey: /key.pem
    letsencrypt:
      enabled: true
      accepttos: true
      cache: /le
      directoryurl: https://acme.example
      hosts:
        - a.tld
        - b.tld
  responseheaders:
    X-Custom: hello
  stream:
    pingperiodseconds: 30
    allowedorigins:
      - o1
      - o2
  cors:
    alloworigins:
      - c1
    allowmethods:
      - GET
    allowheaders:
      - Authorization
  trustedproxies:
    - 10.0.0.1
  securecookie: true
database:
  dialect: postgres
  connection: postgres://localhost/gotify
defaultuser:
  name: root
  pass: secret
passstrength: 12
uploadedimagesdir: /images
pluginsdir: /plugins
registration: true
oidc:
  enabled: true
  issuer: https://issuer.example
  clientid: client
  clientsecret: topsecret
  usernameclaim: email
  redirecturl: https://gotify.example/callback
  autoregister: false
  scopes:
    - openid
    - custom
`
	assert.Equal(t, `MONITA_DATABASE_CONNECTION="postgres://localhost/gotify"
MONITA_DATABASE_DIALECT="postgres"
MONITA_DEFAULTUSER_NAME="root"
MONITA_DEFAULTUSER_PASS="secret"
MONITA_OIDC_AUTOREGISTER="false"
MONITA_OIDC_CLIENTID="client"
MONITA_OIDC_CLIENTSECRET="topsecret"
MONITA_OIDC_ENABLED="true"
MONITA_OIDC_ISSUER="https://issuer.example"
MONITA_OIDC_REDIRECTURL="https://gotify.example/callback"
MONITA_OIDC_SCOPES="openid,custom"
MONITA_OIDC_USERNAMECLAIM="email"
MONITA_PASSSTRENGTH=12
MONITA_PLUGINSDIR="/plugins"
MONITA_REGISTRATION="true"
MONITA_SERVER_CORS_ALLOWHEADERS="Authorization"
MONITA_SERVER_CORS_ALLOWMETHODS="GET"
MONITA_SERVER_CORS_ALLOWORIGINS="c1"
MONITA_SERVER_KEEPALIVEPERIODSECONDS=30
MONITA_SERVER_LISTENADDR="0.0.0.0"
MONITA_SERVER_PORT=8080
MONITA_SERVER_RESPONSEHEADERS="{\"X-Custom\":\"hello\"}"
MONITA_SERVER_SECURECOOKIE="true"
MONITA_SERVER_SSL_CERTFILE="/cert.pem"
MONITA_SERVER_SSL_CERTKEY="/key.pem"
MONITA_SERVER_SSL_ENABLED="true"
MONITA_SERVER_SSL_LETSENCRYPT_ACCEPTTOS="true"
MONITA_SERVER_SSL_LETSENCRYPT_CACHE="/le"
MONITA_SERVER_SSL_LETSENCRYPT_DIRECTORYURL="https://acme.example"
MONITA_SERVER_SSL_LETSENCRYPT_ENABLED="true"
MONITA_SERVER_SSL_LETSENCRYPT_HOSTS="a.tld,b.tld"
MONITA_SERVER_SSL_LISTENADDR="127.0.0.1"
MONITA_SERVER_SSL_PORT=8443
MONITA_SERVER_SSL_REDIRECTTOHTTPS="false"
MONITA_SERVER_STREAM_ALLOWEDORIGINS="o1,o2"
MONITA_SERVER_STREAM_PINGPERIODSECONDS=30
MONITA_SERVER_TRUSTEDPROXIES="10.0.0.1"
MONITA_UPLOADEDIMAGESDIR="/images"`, runMigrate(t, yaml))
}

func TestMigrateConfigNoOptions(t *testing.T) {
	assert.Equal(t, "", runMigrate(t, ""), "an empty config produces no settings")
}

func TestMigrateConfigOnlyDefaultValues(t *testing.T) {
	yaml := `server:
  port: 80
  ssl:
    redirecttohttps: true
database:
  dialect: sqlite3
oidc:
  autoregister: true
  scopes:
    - openid
    - profile
    - email
`
	assert.Equal(t, `MONITA_DATABASE_DIALECT="sqlite3"
MONITA_OIDC_AUTOREGISTER="true"
MONITA_OIDC_SCOPES="openid,profile,email"
MONITA_SERVER_PORT=80
MONITA_SERVER_SSL_REDIRECTTOHTTPS="true"`, runMigrate(t, yaml))
}

func TestMigrateConfigEscapesListEntries(t *testing.T) {
	yaml := `server:
  cors:
    alloworigins:
      - a,b
      - 'say "hi"'
      - c
`
	// The CSV-encoded list contains commas and quotes, which godotenv then
	// double-quotes and escapes.
	assert.Contains(t, runMigrate(t, yaml),
		`MONITA_SERVER_CORS_ALLOWORIGINS="\"a,b\",\"say \"\"hi\"\"\",c"`)
}

func TestMigrateConfigErrors(t *testing.T) {
	_, err := Config("")
	assert.ErrorContains(t, err, "requires one argument", "no path -> usage error")

	missing := filepath.Join(t.TempDir(), "missing.yml")
	_, err = Config(missing)
	assert.ErrorContains(t, err, "cannot read config file", "unreadable file -> error")
}
