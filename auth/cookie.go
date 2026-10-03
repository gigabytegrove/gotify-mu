package auth

import (
	"net/http"
)

// CookieMaxAge is the lifetime of the session cookie in seconds (7 days).
const CookieMaxAge = 7 * 24 * 60 * 60

const CookieName = "monita-client-token"
const legacyCookieNameForCleanup = "gotify-client-token"

func SetCookie(w http.ResponseWriter, token string, maxAge int, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	if maxAge < 0 {
		http.SetCookie(w, &http.Cookie{
			Name:     legacyCookieNameForCleanup,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			Secure:   secure,
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		})
	}
}
