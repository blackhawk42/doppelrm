package doppelparser

import (
	"fmt"
	"io"
	"iter"
	"regexp"
	"strings"
)

var hashRegex = regexp.MustCompile(`^\w+\s*\n`)
var filenameRegex = regexp.MustCompile(`^\s+.*\n?`)

type DoppelCollisions struct {
	collisionMap map[string][]string
	hashOrder    []string
}

func (dc *DoppelCollisions) Iter() iter.Seq2[string, []string] {
	return func(yield func(string, []string) bool) {
		for _, h := range dc.hashOrder {
			if !yield(h, dc.collisionMap[h]) {
				return
			}
		}
	}
}

func (dc *DoppelCollisions) Hashes() []string {
	return dc.hashOrder
}

func (dc *DoppelCollisions) Filenames() [][]string {
	result := make([][]string, 0, len(dc.collisionMap))
	for _, col := range dc.Iter() {
		result = append(result, col)
	}

	return result
}

func (dc *DoppelCollisions) Len() int {
	return len(dc.collisionMap)
}

func (dc *DoppelCollisions) GetFilenames(hash string) ([]string, error) {
	result, ok := dc.collisionMap[hash]
	if !ok {
		return result, fmt.Errorf("hash %s is not in registered collisions", hash)
	}

	return result, nil
}

func lineCol(text string, index int) (int, int) {
	line := 1
	col := 1
	for i := 0; i < index; i++ {
		if text[i] == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}

	return line, col
}

func parseDoppelCollisions(text string) (*DoppelCollisions, error) {
	result := &DoppelCollisions{
		collisionMap: make(map[string][]string),
		hashOrder:    make([]string, 0),
	}
	originalText := text

	currentIndex := 0

	for text != "" {
		loc := hashRegex.FindStringIndex(text)
		if loc == nil {
			line, col := lineCol(originalText, currentIndex)
			return result, fmt.Errorf("parsing error at line %d, col %d: expected hash", line, col)
		}

		hash := strings.TrimSpace(text[loc[0]:loc[1]])
		text = text[loc[1]:]

		result.hashOrder = append(result.hashOrder, hash)
		collisions, ok := result.collisionMap[hash]
		if ok {
			line, col := lineCol(originalText, currentIndex)
			return result, fmt.Errorf("parsing error at line %d, col %d: hash %s has already appeared before", line, col, hash)
		}

		currentIndex += loc[1]

		for {
			loc = filenameRegex.FindStringIndex(text)
			if loc == nil {
				break
			}

			collisions = append(collisions, strings.TrimSpace(text[loc[0]:loc[1]]))
			text = text[loc[1]:]

			currentIndex += loc[1]
		}

		result.collisionMap[hash] = collisions
	}

	return result, nil
}

// ParseDoppelFile takes the content of a file in the format outputed by doppel
// and gives a parsed structure.
func ParseDoppelFile(r io.Reader) (*DoppelCollisions, error) {
	text, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("while reading input: %w", err)
	}

	result, err := parseDoppelCollisions(string(text))

	return result, err
}
