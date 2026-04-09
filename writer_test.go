package apkgwriter

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"os"
	"testing"

	_ "modernc.org/sqlite"
)

func TestWriteApkg(t *testing.T) {
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
}
