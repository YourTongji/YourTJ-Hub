package forum

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/anonymousidentityservice"
	"strconv"
)

func personaAuthor(uid string) TopicAuthorPayload {
	p := anonymousidentityservice.Lookup([]string{uid})[uid]
	return personaAuthorPayload(uid, p)
}
func personaAuthorPayload(uid string, p anonymousidentityservice.PublicPersona) TopicAuthorPayload {
	return TopicAuthorPayload{Kind: "persona", PublicUID: uid, ProfileURL: p.ProfileURL, Username: p.Name, AvatarURL: p.AvatarURL}
}

func publicAuthorProfilePath(author TopicAuthorPayload) string {
	if author.ProfileURL != "" {
		return author.ProfileURL
	}
	if author.ID != 0 {
		return "/u/" + strconv.FormatUint(author.ID, 10)
	}
	return ""
}
