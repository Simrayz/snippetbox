package main

import (
	"fmt"
	"net/http"

	"github.com/justinas/nosurf"
)

func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Restrict where resources can be loaded from.
		// Helps against a variety of cross-site scripting, clickjacking and other code-injection attacks.
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' fonts.googleapis.com; font-src fonts.gstatic.com")
		// Control what is included in the Referer headers when the user navigates away from the page.
		w.Header().Set("Referrer-Policy", "origin-when-cross-origin")
		// Instruct browsers to not sniff the content-type, avoiding content-sniffing attacks
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// Prevent Clickjacking when CSP is not supported. https://developer.mozilla.org/en-US/docs/Web/Security/Types_of_attacks#click-jacking
		w.Header().Set("X-Frame-Options", "deny")
		// Disable blocking of XSS attacks, as recommended with CSP headers
		// https://owasp.org/www-project-secure-headers/#x-xss-protection
		w.Header().Set("X-XSS-Protection", "0")

		next.ServeHTTP(w, r)
	})
}

func (app *application) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			ip     = r.RemoteAddr
			proto  = r.Proto
			method = r.Method
			url    = r.URL.RequestURI()
		)
		app.logger.Info("received request", "ip", ip, "proto", proto, "method", method, "url", url)
		next.ServeHTTP(w, r)
	})
}

// Page 180
func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a deferred function (which will always be run in the event // of a panic as Go unwinds the stack).
		defer func() {
			// Use the builtin recover function to check if there has been a // panic or not. If there has...
			if err := recover(); err != nil {
				// Set a "Connection: close" header on the response.
				w.Header().Set("Connection", "close")
				// Call the app.serverError helper method to return a 500 // Internal Server response.
				app.serverError(w, r, fmt.Errorf("%s", err))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (app *application) requireAuthenticatedUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !app.isAuthenticated(r) {
			http.Redirect(w, r, "/user/login", http.StatusSeeOther)
			return
		}
		w.Header().Add("Cachce-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func noSurf(next http.Handler) http.Handler {
	csrfHandler := nosurf.New(next)
	csrfHandler.SetBaseCookie(http.Cookie{
		HttpOnly: true,
		Path:     "/",
		Secure:   true,
	})
	return csrfHandler
}
