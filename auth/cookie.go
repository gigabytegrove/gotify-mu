package auth

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
