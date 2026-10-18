package web

import (
	"encoding/json"
	"net/http"
	"net/mail"
	"os"
	"strings"
	"time"
)

// submission is one contact/demo form entry.
type submission struct {
	Kind      string    `json:"kind"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	Company   string    `json:"company"`
	Message   string    `json:"message"`
	Product   string    `json:"product"`
	Lang      string    `json:"lang"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	At        time.Time `json:"at"`
}

// handleContact serves POST /api/contact.
func (s *Server) handleContact(w http.ResponseWriter, r *http.Request) {
	s.handleSubmission(w, r, "contact", "/contact-us")
}

// handleDemo serves POST /api/demo.
func (s *Server) handleDemo(w http.ResponseWriter, r *http.Request) {
	s.handleSubmission(w, r, "demo", "/request-demo")
}

// handleApk serves POST /api/apk (Android download requests).
func (s *Server) handleApk(w http.ResponseWriter, r *http.Request) {
	s.handleSubmission(w, r, "apk", "/apk-download")
}

func (s *Server) handleSubmission(w http.ResponseWriter, r *http.Request, kind, redirect string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, redirect+"?sent=error", http.StatusSeeOther)
		return
	}
	// Honeypot: bots fill "website"; humans never see it.
	if strings.TrimSpace(r.FormValue("website")) != "" {
		http.Redirect(w, r, redirect+"?sent=1", http.StatusSeeOther)
		return
	}
	sub := submission{
		Kind:      kind,
		Name:      clip(r.FormValue("name"), 200),
		Email:     clip(r.FormValue("email"), 200),
		Phone:     clip(r.FormValue("phone"), 60),
		Company:   clip(r.FormValue("company"), 200),
		Message:   clip(r.FormValue("message"), 5000),
		Product:   clip(r.FormValue("product"), 120),
		Lang:      clip(r.FormValue("lang"), 8),
		IP:        clientIP(r),
		UserAgent: clip(r.UserAgent(), 400),
		At:        time.Now().UTC(),
	}
	if sub.Name == "" || sub.Message == "" || !validEmail(sub.Email) {
		http.Redirect(w, r, redirect+"?sent=error", http.StatusSeeOther)
		return
	}
	if s.cfg.SubmissionsFile != "" {
		if err := s.appendSubmission(sub); err != nil {
			s.log.Error("submission persist failed", "err", err)
		}
	}
	s.log.Info("form submission", "kind", kind, "name", sub.Name,
		"email", sub.Email, "product", sub.Product)
	http.Redirect(w, r, redirect+"?sent=1", http.StatusSeeOther)
}

func (s *Server) appendSubmission(sub submission) error {
	f, err := os.OpenFile(s.cfg.SubmissionsFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := json.Marshal(sub)
	if err != nil {
		return err
	}
	_, err = f.Write(append(b, '\n'))
	return err
}

func validEmail(s string) bool {
	if s == "" {
		return false
	}
	_, err := mail.ParseAddress(s)
	return err == nil
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	return r.RemoteAddr
}

func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) > n {
		return s[:n]
	}
	return s
}
