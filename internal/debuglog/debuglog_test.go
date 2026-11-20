package debuglog

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddfAndEntries(t *testing.T) {
	Addf("halo %s", "dunia")
	got := Entries()
	assert.Contains(t, got[len(got)-1], "halo dunia")

	for i := 0; i < maxEntries+10; i++ {
		Addf("baris %d", i)
	}
	got = Entries()
	assert.Len(t, got, maxEntries)
	assert.Contains(t, got[len(got)-1], fmt.Sprintf("baris %d", maxEntries+9))
}