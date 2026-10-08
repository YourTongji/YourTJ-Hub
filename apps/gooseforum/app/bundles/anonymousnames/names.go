// Package anonymousnames generates six-character persona names from curated components.
package anonymousnames

import (
	"crypto/rand"
	_ "embed"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"unicode"
	"unicode/utf8"
)

const Version = "phrase6-v1"

//go:embed data/phrases.json
var phrasesJSON []byte

type namePool struct {
	Actions []string `json:"actions"`
	Objects []string `json:"objects"`
	Animals []string `json:"animals"`
}

func (p namePool) size() int { return len(p.Actions) * len(p.Objects) * len(p.Animals) }
func (p namePool) name(index int) string {
	animal := index % len(p.Animals)
	index /= len(p.Animals)
	object := index % len(p.Objects)
	action := index / len(p.Objects)
	return p.Actions[action] + p.Objects[object] + "的" + p.Animals[animal]
}

var load = sync.OnceValues(func() (namePool, error) {
	var pool namePool
	if err := json.Unmarshal(phrasesJSON, &pool); err != nil {
		return namePool{}, err
	}
	for _, component := range []struct {
		words  []string
		length int
	}{
		{pool.Actions, 2}, {pool.Objects, 2}, {pool.Animals, 1},
	} {
		if len(component.words) == 0 {
			return namePool{}, fmt.Errorf("empty anonymous name component")
		}
		seen := make(map[string]bool)
		for _, word := range component.words {
			if seen[word] || utf8.RuneCountInString(word) != component.length {
				return namePool{}, fmt.Errorf("invalid anonymous name component: %q", word)
			}
			for _, r := range word {
				if !unicode.Is(unicode.Han, r) {
					return namePool{}, fmt.Errorf("non-Han component: %q", word)
				}
			}
			seen[word] = true
		}
	}
	if pool.size() < 10 {
		return namePool{}, fmt.Errorf("anonymous name pool has fewer than ten names")
	}
	return pool, nil
})

// Batch uniformly draws ten distinct combinations with cryptographic randomness.
// Components combine freely, including playful phrases. Existing persisted names
// and batches are not regenerated; separate new batches may repeat a name.
func Batch() ([]string, error) {
	pool, err := load()
	if err != nil {
		return nil, err
	}
	selected := make(map[int]bool, 10)
	result := make([]string, 0, 10)
	for len(result) < 10 {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(pool.size())))
		if err != nil {
			return nil, err
		}
		index := int(n.Int64())
		if !selected[index] {
			selected[index] = true
			result = append(result, pool.name(index))
		}
	}
	return result, nil
}
