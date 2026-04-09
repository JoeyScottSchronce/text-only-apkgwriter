package apkgwriter

import (
	"archive/zip"
	"crypto/rand"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Card is a single text front/back pair (Basic note).
type Card struct {
	Front string
	Back  string
}

// WriteApkg writes a text-only .apkg (zip with collection.anki2 + media) to w.
// Media support may be added later without breaking this signature.
func WriteApkg(w io.Writer, deckTitle string, cards []Card) error {
	if len(cards) == 0 {
		return fmt.Errorf("apkgwriter: no cards")
	}
	tmp, err := os.CreateTemp("", "anki-col-*.anki2")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(tmpPath)

	dsn := "file:" + tmpPath + "?_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	if _, err := db.Exec(APKGSchema); err != nil {
		return fmt.Errorf("schema: %w", err)
	}

	now := time.Now()
	tsSec := int(now.Unix())
	tsMs := now.UnixMilli()

	deckID := randomDeckID()
	modelID := tsMs // epoch ms, matches Anki model id convention

	displayName := strings.TrimSpace(deckTitle)
	if displayName == "" {
		displayName = "Untitled Deck"
	}
	if err := insertCol(db, deckID, modelID, displayName, tsSec, int(tsMs)); err != nil {
		return err
	}

	idCounter := tsMs
	nextID := func() int64 {
		idCounter++
		return idCounter
	}

	const sep = "\x1f"
	for i, c := range cards {
		nid := nextID()
		guid := GuidFor(c.Front, c.Back)
		flds := c.Front + sep + c.Back
		tags := " "
		_, err := db.Exec(`INSERT INTO notes VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
			nid, guid, modelID, tsSec, -1, tags, flds, 0, 0, 0, "",
		)
		if err != nil {
			return fmt.Errorf("note %d: %w", i, err)
		}

		cid := nextID()
		due := i + 1 // new card order, 1-based
		_, err = db.Exec(`INSERT INTO cards VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			cid, nid, deckID, 0, tsSec, -1,
			0, 0, // type, queue: new
			due,
			0, 0, 0, 0, 0, 0, 0, 0, "",
		)
		if err != nil {
			return fmt.Errorf("card %d: %w", i, err)
		}
	}

	if err := db.Close(); err != nil {
		return err
	}

	raw, err := os.ReadFile(tmpPath)
	if err != nil {
		return err
	}

	zw := zip.NewWriter(w)
	f, err := zw.Create("collection.anki2")
	if err != nil {
		return err
	}
	if _, err := f.Write(raw); err != nil {
		_ = zw.Close()
		return err
	}
	mf, err := zw.Create("media")
	if err != nil {
		_ = zw.Close()
		return err
	}
	if _, err := mf.Write([]byte("{}")); err != nil {
		_ = zw.Close()
		return err
	}
	return zw.Close()
}

func randomDeckID() int64 {
	var b [8]byte
	_, _ = rand.Read(b[:])
	v := binary.BigEndian.Uint64(b[:])
	return int64(1<<30 + v%(1<<30))
}

// insertCol persists the collection row. json.Marshal on these map[string]any blobs only fails for
// values encoding/json cannot represent (e.g. NaN/Inf floats, chan/func); current literals are all safe.
func insertCol(db *sql.DB, deckID, modelID int64, deckTitle string, crt, modMs int) error {
	conf := map[string]any{
		"activeDecks":   []int64{deckID},
		"addToCur":      true,
		"collapseTime":  1200,
		"curDeck":       deckID,
		"curModel":      fmt.Sprintf("%d", modelID),
		"dueCounts":     true,
		"estTimes":      true,
		"newBury":       true,
		"newSpread":     0,
		"nextPos":       len(deckTitle) + 1,
		"sortBackwards": false,
		"sortType":      "noteFld",
		"timeLim":       0,
	}
	confB, err := json.Marshal(conf)
	if err != nil {
		return fmt.Errorf("apkgwriter: marshal col conf: %w", err)
	}

	decks := map[string]any{
		"1": map[string]any{
			"collapsed": false, "conf": 1, "desc": "", "dyn": 0,
			"extendNew": 10, "extendRev": 50, "id": 1,
			"lrnToday": []int{0, 0}, "mod": modMs / 1000, "name": "Default",
			"newToday": []int{0, 0}, "revToday": []int{0, 0}, "timeToday": []int{0, 0}, "usn": 0,
		},
		fmt.Sprintf("%d", deckID): map[string]any{
			"collapsed": false, "conf": 1, "desc": "", "dyn": 0,
			"extendNew": 10, "extendRev": 50, "id": deckID,
			"lrnToday": []int{0, 0}, "mod": modMs / 1000, "name": deckTitle,
			"newToday": []int{0, 0}, "revToday": []int{0, 0}, "timeToday": []int{0, 0}, "usn": -1,
		},
	}
	decksB, err := json.Marshal(decks)
	if err != nil {
		return fmt.Errorf("apkgwriter: marshal col decks: %w", err)
	}

	model := buildModelJSON(modelID, deckID, modMs/1000, deckTitle)
	models := map[string]any{
		fmt.Sprintf("%d", modelID): model,
	}
	modelsB, err := json.Marshal(models)
	if err != nil {
		return fmt.Errorf("apkgwriter: marshal col models: %w", err)
	}

	dconf := map[string]any{
		"1": map[string]any{
			"autoplay": true, "id": 1,
			"lapse": map[string]any{
				"delays": []int{10}, "leechAction": 0, "leechFails": 8, "minInt": 1, "mult": 0,
			},
			"maxTaken": 60, "mod": 0, "name": "Default",
			"new": map[string]any{
				"bury": true, "delays": []int{1, 10}, "initialFactor": 2500,
				"ints": []int{1, 4, 7}, "order": 1, "perDay": 20, "separate": true,
			},
			"replayq": true,
			"rev": map[string]any{
				"bury": true, "ease4": 1.3, "fuzz": 0.05, "ivlFct": 1, "maxIvl": 36500, "minSpace": 1, "perDay": 100,
			},
			"timer": 0, "usn": 0,
		},
	}
	dconfB, err := json.Marshal(dconf)
	if err != nil {
		return fmt.Errorf("apkgwriter: marshal col dconf: %w", err)
	}

	_, err = db.Exec(`INSERT INTO col VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		1, crt, modMs, modMs, 11, 0, 0, 0,
		string(confB), string(modelsB), string(decksB), string(dconfB), "{}",
	)
	return err
}

func buildModelJSON(modelID, deckID int64, modSec int, noteTypeName string) map[string]any {
	flds := []any{
		map[string]any{"name": "Front", "ord": 0, "font": "Liberation Sans", "media": []any{}, "rtl": false, "size": 20, "sticky": false},
		map[string]any{"name": "Back", "ord": 1, "font": "Liberation Sans", "media": []any{}, "rtl": false, "size": 20, "sticky": false},
	}
	tmpls := []any{
		map[string]any{
			"name": "Card 1", "ord": 0,
			"qfmt":  "{{Front}}",
			"afmt":  "{{FrontSide}}<hr id=answer>{{Back}}",
			"bqfmt": "", "bafmt": "", "bfont": "", "bsize": 0, "did": nil,
		},
	}
	return map[string]any{
		"css":       ".card { font-family: arial; font-size: 20px; text-align: center; color: black; background-color: white; }",
		"did":       deckID,
		"flds":      flds,
		"id":        fmt.Sprintf("%d", modelID),
		"latexPost": "\\end{document}",
		"latexPre":  "\\documentclass[12pt]{article}\n\\usepackage[utf8]{inputenc}\n\\begin{document}\n",
		"latexsvg":  false,
		"mod":       modSec,
		"name":      noteTypeName,
		"req":       []any{[]any{0, "all", []any{0}}},
		"sortf":     0,
		"tags":      []any{},
		"tmpls":     tmpls,
		"type":      0,
		"usn":       -1,
		"vers":      []any{},
	}
}

// SanitizeDeckTitle returns a filename-safe stem (extension added by caller).
func SanitizeDeckTitle(title string) string {
	s := strings.TrimSpace(title)
	if s == "" {
		s = "Untitled Deck"
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == ' ':
			b.WriteRune('_')
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	out := b.String()
	if out == "" {
		return "Untitled_Deck"
	}
	return out
}
