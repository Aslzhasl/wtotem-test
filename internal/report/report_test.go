package report

import "testing"

func TestBuild_IDShape(t *testing.T) {
	b := NewBuilder()
	rep, err := b.Build("https://example.com/cv", "a@b.c")
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.UserID) != 13 || rep.UserID[8] != '-' {
		t.Fatalf("bad id %q", rep.UserID)
	}
}
