package httpapi

import (
	"context"
	"encoding/json"
	"log"
	"math/rand/v2"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
	"wtotem-test/internal/config"
	"wtotem-test/internal/mailer"
	"wtotem-test/internal/report"
	"wtotem-test/internal/zipper"
)

type Server struct {
	http   *http.Server
	cfg    config.Config
	rb     *report.Builder
	z      *zipper.Zipper
	mailer mailer.Sender
}

func NewServer(cfg config.Config, rb *report.Builder, z *zipper.Zipper, m mailer.Sender) *Server {
	mux := http.NewServeMux()
	s := &Server{
		http: &http.Server{
			Addr:              cfg.HTTPAddr,
			Handler:           logging(mux),
			ReadHeaderTimeout: 10 * time.Second,
		},
		cfg:    cfg,
		rb:     rb,
		z:      z,
		mailer: m,
	}

	lim := newLimiter()
	mux.HandleFunc("POST /submit", func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if !lim.allow(ip) {
			writeErr(w, http.StatusTooManyRequests, "slow down")
			return
		}
		s.handleSubmit(w, r)
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	return s
}

func (s *Server) Start() error                       { return s.http.ListenAndServe() }
func (s *Server) Shutdown(ctx context.Context) error { return s.http.Shutdown(ctx) }

// Handler exposes underlying handler for tests.
func (s *Server) Handler() http.Handler { return s.http.Handler }

type submitReq struct {
	CVURL string `json:"cv_url"`
	Email string `json:"email"`
}

func (s *Server) handleSubmit(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	var req submitReq
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.CVURL = strings.TrimSpace(req.CVURL)
	req.Email = strings.TrimSpace(req.Email)
	if req.CVURL == "" || req.Email == "" {
		writeErr(w, http.StatusBadRequest, "cv_url and email are required")
		return
	}

	rep, reportName, zipName, err := s.rb.BuildAndPersist(req.CVURL, req.Email, s.cfg.SourceDir, func(dir, dst string) error {
		return s.z.ZipDir(dir, dst)
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	subject := "Golang Test – " + rep.UserID
	body := "Автоматическая отправка отчёта"
	if err := s.mailer.Send(ctx, req.Email, s.cfg.TargetEmail, subject, body, []string{reportName, zipName}); err != nil {
		writeErr(w, http.StatusBadGateway, "email send failed")
		return
	}

	writeJSON(w, map[string]any{
		"status":  "ok",
		"user_id": rep.UserID,
		"report":  reportName,
		"zip":     zipName,
	})
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := genReqID()
		w.Header().Set("X-Request-ID", id)
		start := time.Now()
		lrw := &logResp{ResponseWriter: w, code: 200}
		next.ServeHTTP(lrw, r)
		log.Printf("%s %s %d id=%s dur=%s", r.Method, r.URL.Path, lrw.code, id, time.Since(start))
	})
}

type logResp struct {
	http.ResponseWriter
	code int
}

func (l *logResp) WriteHeader(code int) { l.code = code; l.ResponseWriter.WriteHeader(code) }

func genReqID() string {
	const a = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = a[rand.IntN(len(a))]
	}
	return string(b)
}

func clientIP(r *http.Request) string {
	// X-Forwarded-For support (first ip)
	if xf := r.Header.Get("X-Forwarded-For"); xf != "" {
		parts := strings.Split(xf, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// --- naive limiter (good enough for this demo) ---
type limiter struct {
	mu     sync.Mutex
	tokens map[string]int
	last   time.Time
}

func newLimiter() *limiter {
	return &limiter{tokens: make(map[string]int), last: time.Now()}
}

// refill every second up to 5 tokens per ip; new ip starts with 5
func (l *limiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	if now.Sub(l.last) > time.Second {
		for k := range l.tokens {
			l.tokens[k]++
			if l.tokens[k] > 5 {
				l.tokens[k] = 5
			}
		}
		l.last = now
	}
	if _, ok := l.tokens[ip]; !ok {
		l.tokens[ip] = 5
	}
	if l.tokens[ip] <= 0 {
		return false
	}
	l.tokens[ip]--
	return true
}
