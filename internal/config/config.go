package config

import (
	"fmt"
	"os"
)

type SMTP struct {
	Login, Password, Server, Port string
}

type Config struct {
	CVURL       string
	Email       string
	SMTP        SMTP
	HTTPAddr    string
	SourceDir   string
	TargetEmail string
}

func FromFlagsOrEnv(cvURL, email, login, pass, server, port, addr, src, to string) Config {
	if cvURL == "" {
		cvURL = os.Getenv("CV_URL")
	}
	if email == "" {
		email = os.Getenv("EMAIL")
	}
	if login == "" {
		login = os.Getenv("SMTP_LOGIN")
	}
	if pass == "" {
		pass = os.Getenv("SMTP_PASSWORD")
	}
	if server == "" {
		server = envOr("SMTP_SERVER", "smtp.gmail.com")
	}
	if port == "" {
		port = envOr("SMTP_PORT", "587")
	}
	if addr == "" {
		addr = envOr("ADDR", ":8080")
	}
	if src == "" {
		src = envOr("PROJECT_DIR", ".")
	}
	if to == "" {
		to = envOr("TARGET_EMAIL", "szhaisan@wtotem.com")
	}

	return Config{
		CVURL: cvURL, Email: email,
		SMTP:        SMTP{Login: login, Password: pass, Server: server, Port: port},
		HTTPAddr:    addr,
		SourceDir:   src,
		TargetEmail: to,
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func (c Config) Validate() error {
	// HTTP-only: cv_url/email come in request body; validate env/SMTP basics.
	if c.SMTP.Login == "" || c.SMTP.Password == "" {
		return fmt.Errorf("SMTP_LOGIN and SMTP_PASSWORD are required")
	}
	if c.SMTP.Server == "" || c.SMTP.Port == "" {
		return fmt.Errorf("smtp server/port are required")
	}
	if c.TargetEmail == "" {
		return fmt.Errorf("target email is required")
	}
	return nil
}
