package report

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Report struct {
	CVURL     string `json:"cv_url"`
	Hash      string `json:"hash"`
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Timestamp string `json:"timestamp"`
}

type Builder struct{}

func NewBuilder() *Builder { return &Builder{} }

// Build computes SHA256 over cvURL and creates userID: first 8 chars + "-" + 4 random [a-z0-9]
func (b *Builder) Build(cvURL, email string) (Report, error) {
	h := sha256.Sum256([]byte(cvURL))
	hash := hex.EncodeToString(h[:])
	id := fmt.Sprintf("%s-%s", hash[:8], randSuffix(4))
	return Report{
		CVURL:     cvURL,
		Hash:      hash,
		UserID:    id,
		Email:     email,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// BuildAndPersist writes report_<user_id>.json and creates source_code.zip via provided zipper.
func (b *Builder) BuildAndPersist(cvURL, email, srcDir string, zipFunc func(dir, dst string) error) (Report, string, string, error) {
	rep, err := b.Build(cvURL, email)
	if err != nil {
		return Report{}, "", "", err
	}

	reportName := fmt.Sprintf("report_%s.json", rep.UserID)
	if err := writePrettyJSON(reportName, rep); err != nil {
		return Report{}, "", "", err
	}

	zipName := "source_code.zip"
	if err := zipFunc(srcDir, zipName); err != nil {
		return Report{}, "", "", err
	}

	return rep, reportName, zipName, nil
}

func writePrettyJSON(path string, v any) error {
	_ = os.MkdirAll(filepath.Dir(filepath.Clean(path)), 0o755)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func randSuffix(n int) string {
	const alpha = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	_, _ = rand.Read(b) // best-effort; fine for this use
	for i := range b {
		b[i] = alpha[int(b[i])%len(alpha)]
	}
	return string(b)
}
