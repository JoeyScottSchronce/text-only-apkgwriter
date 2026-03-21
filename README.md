# text-only-apkgwriter

Go library that builds a **text-only** Anki `.apkg` package (zip with SQLite collection).

**Canonical repository:** [github.com/JoeyScottSchronce/text-only-apkgwriter](https://github.com/JoeyScottSchronce/text-only-apkgwriter)

## Module

```text
module github.com/JoeyScottSchronce/text-only-apkgwriter
```

Consumers (e.g. AnkiDeck’s export service):

```bash
go get github.com/JoeyScottSchronce/text-only-apkgwriter@v0.1.0
```

Use a **semver tag** (`v0.1.0`, …) after the first release so `go get` resolves without `replace`.

## Publish this repo (initial)

If this tree is the first content for the GitHub repo (empty remote):

```bash
cd apkgwriter   # or the folder containing this README + go.mod
git init
git add .
git commit -m "Initial import: text-only apkg writer"
git branch -M main
git remote add origin https://github.com/JoeyScottSchronce/text-only-apkgwriter.git
git push -u origin main
git tag v0.1.0
git push origin v0.1.0
```

Then in **AnkiDeck** (or any consumer), remove the `replace` line in `backend/go.mod`, run:

```bash
go get github.com/JoeyScottSchronce/text-only-apkgwriter@v0.1.0
```

and delete the local `apkgwriter/` copy from the monorepo if you no longer want a submodule.

## API

- `apkgwriter.Card` — `Front` / `Back` strings.
- `apkgwriter.WriteApkg(w io.Writer, deckTitle string, cards []Card) error`

No HTTP server and **no** dependency on app frontends or API servers.

## Tests

```bash
go test ./...
```

## Scope

v1 is **text-only**; the package can grow to support media later without breaking callers.
