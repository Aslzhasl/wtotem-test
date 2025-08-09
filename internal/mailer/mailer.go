package mailer

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"mime/quotedprintable"
	"net/smtp"
	"net/textproto"
	"os"
	"path/filepath"
)

type Sender interface {
	Send(ctx context.Context, from, to, subject, body string, attachments []string) error
}

type SMTP struct {
	Login, Password, Server, Port string
}

type Mailer struct{ cfg SMTP }

func New(s SMTP) *Mailer { return &Mailer{cfg: s} }

func (m *Mailer) Send(ctx context.Context, from, to, subject, body string, attachments []string) error {
	addr := m.cfg.Server + ":" + m.cfg.Port
	auth := smtp.PlainAuth("", m.cfg.Login, m.cfg.Password, m.cfg.Server)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	// headers
	fmt.Fprintf(&buf, "From: %s\r\n", from)
	fmt.Fprintf(&buf, "To: %s\r\n", to)
	fmt.Fprintf(&buf, "Subject: %s\r\n", subject)
	fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&buf, "Content-Type: multipart/mixed; boundary=%q\r\n\r\n", w.Boundary())

	// text part
	part, _ := w.CreatePart(textproto.MIMEHeader{
		"Content-Type":              {"text/plain; charset=utf-8"},
		"Content-Transfer-Encoding": {"quoted-printable"},
	})
	qpw := quotedprintable.NewWriter(part)
	_, _ = qpw.Write([]byte(body))
	_ = qpw.Close()

	// attachments
	for _, p := range attachments {
		if err := attach(w, p); err != nil {
			return fmt.Errorf("attach %s: %w", p, err)
		}
	}
	_ = w.Close()

	done := make(chan error, 1)
	go func() { done <- smtp.SendMail(addr, auth, from, []string{to}, buf.Bytes()) }()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}

func attach(w *multipart.Writer, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	_, name := filepath.Split(path)

	h := textproto.MIMEHeader{}
	h.Set("Content-Type", detectContentType(name))
	h.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))
	h.Set("Content-Transfer-Encoding", "base64")

	part, err := w.CreatePart(h)
	if err != nil {
		return err
	}

	enc := base64.NewEncoder(base64.StdEncoding, part)
	if _, err := enc.Write(data); err != nil {
		return err
	}
	return enc.Close()
}

func detectContentType(name string) string {
	switch {
	case hasSuffix(name, ".json"):
		return "application/json"
	case hasSuffix(name, ".zip"):
		return "application/zip"
	default:
		return "application/octet-stream"
	}
}

func hasSuffix(s, suf string) bool { return len(s) >= len(suf) && s[len(s)-len(suf):] == suf }
