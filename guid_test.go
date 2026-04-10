package apkgwriter

import (
	"testing"
)

func TestGuidFor(t *testing.T) {
	t.Run("deterministic", func(t *testing.T) {
		a := GuidFor("front", "back")
		b := GuidFor("front", "back")
		if a != b {
			t.Fatalf("same inputs: %q vs %q", a, b)
		}
		want := "O!Dlg?gp^V"
		if a != want {
			t.Fatalf("golden: got %q want %q", a, want)
		}
		if GuidFor("other", "pair") == a {
			t.Fatal("different inputs should not collide with golden vector")
		}
	})
	t.Run("zero_hash_branch", func(t *testing.T) {
		if got := guidFromHashInt(0); got != "a" {
			t.Fatalf("guidFromHashInt(0): got %q want a", got)
		}
	})
}
