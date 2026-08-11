package searchservice

import (
	"testing"

	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"gorm.io/gorm"
)

func TestShouldIndexUser(t *testing.T) {
	tests := []struct {
		name string
		user *users.EntityComplete
		want bool
	}{
		{name: "human", user: &users.EntityComplete{Id: 1, ActorType: users.ActorTypeHuman}, want: true},
		{name: "bot", user: &users.EntityComplete{Id: 2, ActorType: users.ActorTypeBot}, want: false},
		{name: "deleted", user: &users.EntityComplete{Id: 3, DeletedAt: gorm.DeletedAt{Valid: true}}, want: false},
		{name: "missing id", user: &users.EntityComplete{}, want: false},
		{name: "nil", user: nil, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldIndexUser(tt.user); got != tt.want {
				t.Fatalf("shouldIndexUser() = %v, want %v", got, tt.want)
			}
		})
	}
}
