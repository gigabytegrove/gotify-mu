# OIDC Testing

## Dex

Check config in ./dex/config/dex.conf and do a `docker-compose up -d`.

Use this monita config.
```ini
MONITA_OIDC_ENABLED=true
MONITA_OIDC_ISSUER=http://127.0.0.1:5556/dex
MONITA_OIDC_CLIENTID=monita
MONITA_OIDC_CLIENTSECRET=secret
MONITA_OIDC_REDIRECTURL=http://127.0.0.1:8080/auth/oidc/callback
```

When testing external apps like monita/android change every occurence of
127.0.0.1 in ./dex/config/dex.conf and in the monita config above to an IP that's
routed in your local network like 192.168.178.2.

## Authelia

Authelia requires SSL to work, so you'll have to create a valid certificate. This has to be executed in the directory this README resides.

```
openssl req -x509 -newkey rsa:4096 -nodes -keyout ./authelia/config/key -out ./authelia/config/cert -days 365 -subj "/CN=127.0.0.1" -addext "subjectAltName=IP:127.0.0.1"
```

Check config in ./authelia/config/configuration.yml and do a `docker-compose up -d`.

Use this monita config.
```ini
MONITA_OIDC_ENABLED=true
MONITA_OIDC_ISSUER=https://127.0.0.1:9091
MONITA_OIDC_CLIENTID=monita
MONITA_OIDC_CLIENTSECRET=secret
MONITA_OIDC_REDIRECTURL=http://127.0.0.1:8080/auth/oidc/callback
MONITA_OIDC_SCOPES=openid,profile,email,groups
# MONITA_OIDC_GROUPS_CLAIM=groups
# MONITA_OIDC_GROUPS_USER=
# MONITA_OIDC_GROUPS_ADMIN=authelia-group
```

When testing external apps like monita/android change every occurence of
127.0.0.1 in ./authelia/config/configuration.yml and in the monita config above
to an IP that's routed in your local network like 192.168.178.2. Also recreate
the certificate with the adjusted IP.
