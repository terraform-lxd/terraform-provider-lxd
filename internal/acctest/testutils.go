package acctest

import (
	"math/rand"

	petname "github.com/dustinkirkland/golang-petname"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// generateString generates a random string of the given length.
func generateString(length int) string {
	s := make([]byte, length)
	for i := range s {
		s[i] = charset[rand.Intn(len(charset))]
	}
	return string(s)
}

// GenerateName generates a petname with a random string suffix.
// If requested number of words is 1 or less, just petname is returned.
func GenerateName(words int, separator string) string {
	if words <= 1 {
		return petname.Name()
	}

	return petname.Generate(words-1, separator) + separator + generateString(6)
}
