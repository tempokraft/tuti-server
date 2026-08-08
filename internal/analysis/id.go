package analysis

import (
	"crypto/rand"
	"encoding/hex"
)

// randSuffix generates a short random hex id suffix for model-detected
// problems/mistakes, mirroring internal/storage/localfs's id scheme.
func randSuffix() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "0"
	}
	return hex.EncodeToString(b)
}
