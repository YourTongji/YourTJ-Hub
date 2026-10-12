package profileprivacyservice

import (
	"slices"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userActivities"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
)

func TestDecideAudienceMatrix(t *testing.T) {
	type viewer struct{ signedIn, blocked, follows, followedBack bool }
	guest := viewer{}
	stranger := viewer{signedIn: true}
	follower := viewer{signedIn: true, follows: true}
	followedOnly := viewer{signedIn: true, followedBack: true}
	mutual := viewer{signedIn: true, follows: true, followedBack: true}
	blockedMutual := viewer{signedIn: true, blocked: true, follows: true, followedBack: true}
	cases := []struct {
		audience string
		viewer   viewer
		visible  bool
		reason   string
	}{
		{users.AudienceEveryone, guest, true, ""},
		{users.AudienceEveryone, blockedMutual, true, ""},
		{users.AudienceMembers, guest, false, ReasonSignIn},
		{users.AudienceMembers, stranger, true, ""},
		{users.AudienceMembers, blockedMutual, false, ReasonPrivate},
		{users.AudienceFollowers, guest, false, ReasonFollowers},
		{users.AudienceFollowers, stranger, false, ReasonFollowers},
		{users.AudienceFollowers, followedOnly, false, ReasonFollowers},
		{users.AudienceFollowers, follower, true, ""},
		{users.AudienceFollowers, blockedMutual, false, ReasonPrivate},
		{users.AudienceMutual, follower, false, ReasonMutual},
		{users.AudienceMutual, followedOnly, false, ReasonMutual},
		{users.AudienceMutual, mutual, true, ""},
		{users.AudienceMutual, blockedMutual, false, ReasonPrivate},
		{users.AudienceSelf, mutual, false, ReasonPrivate},
		{"unknown", mutual, false, ReasonPrivate},
	}
	for _, tc := range cases {
		got := decide(tc.audience, tc.viewer.signedIn, tc.viewer.blocked, tc.viewer.follows, tc.viewer.followedBack)
		if got.Visible != tc.visible || got.Reason != tc.reason || got.Audience != "" {
			t.Errorf("decide(%s, %+v) = %+v, want visible=%v reason=%q", tc.audience, tc.viewer, got, tc.visible, tc.reason)
		}
	}
}

func TestTimelineActionsFollowEachSection(t *testing.T) {
	view := func(hidden ...string) View {
		v := View{decisions: map[string]Decision{}}
		for _, section := range sections {
			v.decisions[section] = Decision{Visible: !slices.Contains(hidden, section)}
		}
		return v
	}
	post, like, follow := int(userActivities.ActionPost), int(userActivities.ActionLike), int(userActivities.ActionFollow)
	if got := view(SectionActivity).TimelineActions(); len(got) != 0 {
		t.Fatalf("hidden activity still lists %v", got)
	}
	all := view().TimelineActions()
	for _, action := range []int{post, like, follow, int(userActivities.ActionComment), int(userActivities.ActionSignUp)} {
		if !slices.Contains(all, action) {
			t.Fatalf("public timeline misses action %d", action)
		}
	}
	for section, action := range map[string]int{SectionTopics: post, SectionLikes: like, SectionFollowing: follow} {
		if slices.Contains(view(section).TimelineActions(), action) {
			t.Fatalf("hidden %s still lists action %d", section, action)
		}
	}
}
