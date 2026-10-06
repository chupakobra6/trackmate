package monthlyreport

import (
	"strings"
	"testing"
	"time"
)

func TestMonth(t *testing.T) {
	now := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	p, err := Month("2026-12", "Asia/Tokyo", now)
	if err != nil {
		t.Fatal(err)
	}
	if p.Start.UTC() != time.Date(2026, 11, 30, 15, 0, 0, 0, time.UTC) || p.End.UTC() != time.Date(2026, 12, 31, 15, 0, 0, 0, time.UTC) {
		t.Fatalf("%+v", p)
	}
	for _, month := range []string{"2027-01", "2027-02", "2026-1", "bad"} {
		if _, err = Month(month, "Asia/Tokyo", now); err == nil {
			t.Fatalf("accepted %s", month)
		}
	}
	if _, err = Month("2026-12", "Asia/Tokyo", p.End.Add(-time.Nanosecond)); err == nil {
		t.Fatal("accepted incomplete month")
	}
	if _, err = Month("2026-12", "Asia/Tokyo", p.End); err != nil {
		t.Fatal("exclusive boundary rejected", err)
	}
}
func TestPrepare(t *testing.T) {
	escaped, hash, err := Prepare("<&> 😀")
	if err != nil || escaped != "&lt;&amp;&gt; 😀" || len(hash) != 64 {
		t.Fatalf("%s %s %v", escaped, hash, err)
	}
	for _, text := range []string{"", " \n", strings.Repeat("a", 4097), strings.Repeat("😀", 2049), string([]byte{255})} {
		if _, _, err := Prepare(text); err == nil {
			t.Fatal("invalid accepted")
		}
	}
	for _, text := range []string{strings.Repeat("a", 4096), strings.Repeat("😀", 2048)} {
		if _, _, err := Prepare(text); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPrepareCanonicalWhitespace(t *testing.T) {
	original := " \t\n<report> 😀\n\n  internal spaces  \nnext\n\u00a0"
	expected := "<report> 😀\n\n  internal spaces  \nnext"
	got, err := prepare(original)
	if err != nil {
		t.Fatal(err)
	}
	escaped, hash, err := Prepare(expected)
	if err != nil {
		t.Fatal(err)
	}
	if got.text != expected || got.html != escaped || got.hash != hash {
		t.Fatalf("canonical mismatch: %+v", got)
	}
	if _, _, err = Prepare(" \n" + strings.Repeat("😀", 2048) + "\n"); err != nil {
		t.Fatal("limit measured before canonicalization", err)
	}
	if _, _, err = Prepare(" \n" + string([]byte{255}) + " \n"); err == nil {
		t.Fatal("invalid UTF-8 normalized away")
	}
}
