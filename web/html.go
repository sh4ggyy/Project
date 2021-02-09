package web

import "net/http"

const htmlIndex = `<html><body>
Logged in with <a href="/login">GitHub</a>
</body></html>
`

func HandleMain(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(htmlIndex))
}
