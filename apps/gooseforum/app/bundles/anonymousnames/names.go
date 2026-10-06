// Package anonymousnames embeds the unfiltered THUOCL lexicon for persona names.
package anonymousnames

import (
	"bufio"
	"crypto/rand"
	"embed"
	"fmt"
	"math/big"
	"strings"
	"sync"
)

const Version = "THUOCL-a30ce79d895d01ab5132a5c74c29703ff7efb4cc"

//go:embed data/*.txt LICENSE README.md
var source embed.FS

var load = sync.OnceValues(func() ([]string, error) {
	files, err := source.ReadDir("data")
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	words := make([]string, 0, 156289)
	for _, file := range files {
		data, err := source.ReadFile("data/" + file.Name())
		if err != nil {
			return nil, err
		}
		// The place-name file uses bare CR; three files pad the tab fields.
		text := strings.ReplaceAll(strings.TrimPrefix(string(data), "\ufeff"), "\r", "\n")
		scanner := bufio.NewScanner(strings.NewReader(text))
		for scanner.Scan() {
			word, _, _ := strings.Cut(scanner.Text(), "\t")
			word = strings.TrimSpace(word)
			if word == "" || seen[word] {
				continue
			}
			seen[word] = true
			words = append(words, word)
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}
	if len(words) < 10 {
		return nil, fmt.Errorf("anonymous lexicon has fewer than ten words")
	}
	return words, nil
})

// Batch draws ten distinct words uniformly. Separate batches may repeat words.
func Batch() ([]string, error) {
	words, err := load()
	if err != nil {
		return nil, err
	}
	selected := make(map[int]bool, 10)
	result := make([]string, 0, 10)
	for len(result) < 10 {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(words))))
		if err != nil {
			return nil, err
		}
		index := int(n.Int64())
		if !selected[index] {
			selected[index] = true
			result = append(result, words[index])
		}
	}
	return result, nil
}
