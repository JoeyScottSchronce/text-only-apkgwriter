# text-only-apkgwriter

Go library that writes a **text-only** Anki **`.apkg`**: a zip containing **`collection.anki2`** (SQLite) and a minimal **`media`** manifest.

```go
import "github.com/JoeyScottSchronce/text-only-apkgwriter"

err := apkgwriter.WriteApkg(w, "My deck", []apkgwriter.Card{
    {Front: "Q", Back: "A"},
})
```

## Resource use and large decks

**Memory model**

1. All notes are written into a **temporary SQLite file** on disk (`os.CreateTemp`).
2. After the DB is closed, the library **`os.ReadFile`s the entire file** into a `[]byte`.
3. That blob is written into **`collection.anki2`** inside a **`zip.Writer`** on the caller’s `io.Writer`.

Work is **O(n)** in **card count** and **front/back sizes**. Peak memory is driven by the **full collection file size** plus whatever the **`archive/zip`** path and your `io.Writer` hold (callers that use a `bytes.Buffer` also retain the full `.apkg` in memory).

There is **no maximum deck size, card count, or field length** inside this package so it stays usable from CLIs and tests. **Callers** must cap workload if they care about OOM risk.

**Recommended limits (reference consumer)**

The primary consumer, **[ankideck-backend](https://github.com/JoeyScottSchronce/ankideck-backend)**, validates each card with **`domain.FilterValid`** ( **`MaxFieldRunes`** per question/answer). Generate responses also cap at **`MaxCardsPerDeck`**; the export HTTP handler does **not** apply that same numeric card ceiling—deck size is mainly bounded by the **~8 MiB** JSON body limit (`io.LimitReader` in **`internal/httpexport`**). Those policies live in the API layer and are **not duplicated** in this library.

**Streaming / chunking**

The current API is **not** a streaming pipeline from SQLite into the zip. Supporting **lower peak memory** or **chunked** export would require **new exported functions** or signatures; that is **out of scope** until implemented.

## Tests

**Locally:** `go vet ./...` and `go test ./...` (from the module root).

**CI:** [`.github/workflows/ci.yml`](.github/workflows/ci.yml) runs the same commands on **push** and **pull_request** to **`main`** (Go version from **`go.mod`**).
