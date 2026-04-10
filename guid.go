package apkgwriter

import (
	"crypto/sha256"
	"strings"
)

// base91Table matches genanki's util.BASE91_TABLE.
var base91Table = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!#$%&()*+,-./:;<=>?@[]^_`{|}~")

// GuidFor derives a stable Anki-style guid from field values (same algorithm as genanki).
func GuidFor(values ...string) string {
	hashStr := strings.Join(values, "__")
	sum := sha256.Sum256([]byte(hashStr))
	var hashInt uint64
	for _, b := range sum[:8] {
		hashInt = (hashInt << 8) | uint64(b)
	}
	return guidFromHashInt(hashInt)
}

func guidFromHashInt(hashInt uint64) string {
	if hashInt == 0 {
		return "a"
	}
	var rev []byte
	for hashInt > 0 {
		rev = append(rev, base91Table[hashInt%uint64(len(base91Table))])
		hashInt /= uint64(len(base91Table))
	}
	for i, j := 0, len(rev)-1; i < j; i, j = i+1, j-1 {
		rev[i], rev[j] = rev[j], rev[i]
	}
	return string(rev)
}
