// Package debuglog menyimpan jejak singkat (in-memory) untuk diagnosis sementara.
// HAPUS paket ini (dan pemakaiannya) setelah masalah admin selesai.
package debuglog

import (
	"fmt"
	"sync"
	"time"
)

const maxEntries = 500

var (
	mu      sync.Mutex
	entries []string
)

// Addf menambahkan satu baris log (password TIDAK PERNAH dicatat).
func Addf(format string, args ...interface{}) {
	line := time.Now().Format("15:04:05") + " " + fmt.Sprintf(format, args...)
	mu.Lock()
	defer mu.Unlock()
	entries = append(entries, line)
	if len(entries) > maxEntries {
		entries = entries[len(entries)-maxEntries:]
	}
}

// Entries mengembalikan salinan seluruh baris log.
func Entries() []string {
	mu.Lock()
	defer mu.Unlock()
	out := make([]string, len(entries))
	copy(out, entries)
	return out
}