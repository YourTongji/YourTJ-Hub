// Package profileprivacyservice decides which profile aggregates a viewer may
// see. It hides profile summaries only: topics, replies and relationships stay
// reachable through their ordinary forum routes.
package profileprivacyservice

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userActivities"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userFollow"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
)

// Profile sections, named after their settings keys.
const (
	SectionTopics    = "profileTopics"
	SectionActivity  = "profileActivity"
	SectionLikes     = "profileLikes"
	SectionFollowing = "followingList"
	SectionFollowers = "followersList"
)

// Reasons tell a viewer what would reveal a hidden section. Blocks and storage
// failures use the generic private reason so neither is disclosed.
const (
	ReasonSignIn    = "signin"
	ReasonFollowers = "followers"
	ReasonMutual    = "mutual"
	ReasonPrivate   = "private"
)

type Decision struct {
	Visible bool
	// Reason is set only when the section is hidden.
	Reason string
	// Audience is the configured audience, exposed to the owner only.
	Audience string
}

type View struct {
	decisions map[string]Decision
}

func (v View) Section(section string) Decision { return v.decisions[section] }
func (v View) Visible(section string) bool     { return v.decisions[section].Visible }

// TimelineActions lists the activity types this viewer may see. Posts, likes
// and follows each also obey their own section so the timeline cannot reveal
// a hidden list indirectly.
func (v View) TimelineActions() []int {
	if !v.Visible(SectionActivity) {
		return nil
	}
	actions := []int{int(userActivities.ActionSignUp), int(userActivities.ActionComment)}
	if v.Visible(SectionTopics) {
		actions = append(actions, int(userActivities.ActionPost))
	}
	if v.Visible(SectionLikes) {
		actions = append(actions, int(userActivities.ActionLike))
	}
	if v.Visible(SectionFollowing) {
		actions = append(actions, int(userActivities.ActionFollow))
	}
	return actions
}

var sections = []string{SectionTopics, SectionActivity, SectionLikes, SectionFollowing, SectionFollowers}

func audienceOf(settings users.PrivacySettingsEntity, section string) string {
	switch section {
	case SectionTopics:
		return settings.ProfileTopics
	case SectionActivity:
		return settings.ProfileActivity
	case SectionLikes:
		return settings.ProfileLikes
	case SectionFollowing:
		return settings.FollowingList
	default:
		return settings.FollowersList
	}
}

// Evaluate resolves every section for one viewer before any profile query
// runs. The owner always sees everything; read failures hide every section.
func Evaluate(ownerID, viewerID uint64) View {
	view := View{decisions: make(map[string]Decision, len(sections))}
	settings, err := users.GetPrivacySettings(ownerID)
	if viewerID != 0 && viewerID == ownerID {
		for _, section := range sections {
			decision := Decision{Visible: true}
			if err == nil {
				decision.Audience = audienceOf(settings, section)
			}
			view.decisions[section] = decision
		}
		return view
	}
	if err != nil {
		for _, section := range sections {
			view.decisions[section] = Decision{Reason: ReasonPrivate}
		}
		return view
	}
	restricted := false
	for _, section := range sections {
		restricted = restricted || audienceOf(settings, section) != users.AudienceEveryone
	}
	var blocked, viewerFollows, ownerFollows bool
	if restricted && viewerID != 0 {
		var blockErr, followErr error
		blocked, blockErr = users.InteractionBlocked(viewerID, ownerID)
		viewerFollows, ownerFollows, followErr = userFollow.MutualState(viewerID, ownerID)
		// Fail closed: an unknown relationship never grants follower access.
		blocked = blocked || blockErr != nil || followErr != nil
	}
	for _, section := range sections {
		view.decisions[section] = decide(audienceOf(settings, section), viewerID != 0, blocked, viewerFollows, ownerFollows)
	}
	return view
}

func decide(audience string, signedIn, blocked, viewerFollows, ownerFollows bool) Decision {
	switch {
	case audience == users.AudienceEveryone:
		return Decision{Visible: true}
	case blocked || audience == users.AudienceSelf:
		return Decision{Reason: ReasonPrivate}
	case audience == users.AudienceMembers:
		if signedIn {
			return Decision{Visible: true}
		}
		return Decision{Reason: ReasonSignIn}
	case audience == users.AudienceFollowers:
		if viewerFollows {
			return Decision{Visible: true}
		}
		return Decision{Reason: ReasonFollowers}
	case audience == users.AudienceMutual:
		if viewerFollows && ownerFollows {
			return Decision{Visible: true}
		}
		return Decision{Reason: ReasonMutual}
	}
	// Unknown stored values fail closed.
	return Decision{Reason: ReasonPrivate}
}

func Get(userID uint64) (users.PrivacySettingsEntity, error) {
	return users.GetPrivacySettings(userID)
}

func Update(userID uint64, update users.PrivacySettingsUpdate) (users.PrivacySettingsEntity, error) {
	return users.UpdatePrivacySettings(userID, update)
}
