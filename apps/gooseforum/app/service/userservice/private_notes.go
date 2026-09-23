package userservice

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
)

var ErrInvalidPrivateNote = errors.New("invalid private note")

func SetPrivateNote(ownerID, targetID uint64, text string) error {
	text = strings.TrimSpace(text)
	if !utf8.ValidString(text) || utf8.RuneCountInString(text) > 64 {
		return ErrInvalidPrivateNote
	}
	for _, r := range text {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return ErrInvalidPrivateNote
		}
	}
	return users.SetPrivateNote(ownerID, targetID, text)
}
func ListPrivateNotes(ownerID uint64) ([]users.PrivateNote, error) {
	return users.ListPrivateNotes(ownerID)
}
