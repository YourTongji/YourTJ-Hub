package anonymousidentityservice

import (
	"context"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	identity "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/anonymousIdentity"
)

// Lookup reads only public persona rows. Missing identities stay anonymous.
func Lookup(uids []string) map[string]PublicPersona {
	result := make(map[string]PublicPersona, len(uids))
	for _, uid := range uids {
		if uid != "" {
			result[uid] = PublicPersona{Kind: "persona", PublicUID: uid, Name: "匿名同学", ProfileURL: "/a/" + uid}
		}
	}
	rows, err := identity.GetMap(db.ConnectContext(context.Background()), uids)
	if err != nil {
		return result
	}
	for uid, row := range rows {
		result[uid] = Public(row)
	}
	return result
}
