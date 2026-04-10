package apkgwriter

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"os"
	"sort"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestSanitizeDeckTitle(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", "Untitled_Deck"},
		{"   ", "Untitled_Deck"},
		{"My Deck", "My_Deck"},
		{"café", "caf_"},
		{"a/b\\c:d", "a_b_c_d"},
		{"###", "___"},
		{"...", "___"},
		{"\u3042\u3044", "__"},
		{"---", "---"},
		{"ok-123_ABC", "ok-123_ABC"},
		{strings.Repeat("x", 500), strings.Repeat("x", 500)},
	}
	for _, tt := range tests {
		got := SanitizeDeckTitle(tt.in)
		if got != tt.want {
			t.Errorf("SanitizeDeckTitle(%q) = %q; want %q", tt.in, got, tt.want)
		}
	}
}

func TestWriteApkg(t *testing.T) {
	t.Run("happy_path", func(t *testing.T) {
		var buf bytes.Buffer
		err := WriteApkg(&buf, "Test Deck", []Card{
			{Front: "Q1", Back: "A1"},
			{Front: "Q2", Back: "A2"},
		})
		if err != nil {
			t.Fatal(err)
		}

		zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
		if err != nil {
			t.Fatal(err)
		}
		names := make([]string, 0, len(zr.File))
		for _, f := range zr.File {
			names = append(names, f.Name)
		}
		sort.Strings(names)
		wantNames := []string{"collection.anki2", "media"}
		if len(names) != len(wantNames) {
			t.Fatalf("zip entries: got %v want %v", names, wantNames)
		}
		for i := range wantNames {
			if names[i] != wantNames[i] {
				t.Fatalf("zip entries: got %v want %v", names, wantNames)
			}
		}

		var col []byte
		for _, f := range zr.File {
			if f.Name == "collection.anki2" {
				rc, err := f.Open()
				if err != nil {
					t.Fatal(err)
				}
				col, err = io.ReadAll(rc)
				_ = rc.Close()
				if err != nil {
					t.Fatal(err)
				}
				break
			}
		}
		if len(col) == 0 {
			t.Fatal("missing collection.anki2")
		}

		tmp := t.TempDir() + "/col.anki2"
		if err := os.WriteFile(tmp, col, 0o600); err != nil {
			t.Fatal(err)
		}

		db, err := sql.Open("sqlite", "file:"+tmp+"?mode=ro")
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM notes`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 2 {
			t.Fatalf("notes: got %d want 2", n)
		}
		if err := db.QueryRow(`SELECT COUNT(*) FROM cards`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 2 {
			t.Fatalf("cards: got %d want 2", n)
		}

		var flds string
		if err := db.QueryRow(`SELECT flds FROM notes ORDER BY id LIMIT 1`).Scan(&flds); err != nil {
			t.Fatal(err)
		}
		if flds != "Q1\x1fA1" {
			t.Fatalf("flds: %q", flds)
		}

		var modelsJSON string
		if err := db.QueryRow(`SELECT models FROM col`).Scan(&modelsJSON); err != nil {
			t.Fatal(err)
		}
		var models map[string]map[string]any
		if err := json.Unmarshal([]byte(modelsJSON), &models); err != nil {
			t.Fatal(err)
		}
		wantName := "Test Deck"
		var gotName string
		for _, m := range models {
			if n, ok := m["name"].(string); ok {
				gotName = n
				break
			}
		}
		if gotName != wantName {
			t.Fatalf("note type name: got %q want %q (models JSON uses deck title, not Basic)", gotName, wantName)
		}
	})

	t.Run("no_cards", func(t *testing.T) {
		var buf bytes.Buffer
		err := WriteApkg(&buf, "X", nil)
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "no cards") {
			t.Fatalf("error %q should mention no cards", err.Error())
		}
	})
}
