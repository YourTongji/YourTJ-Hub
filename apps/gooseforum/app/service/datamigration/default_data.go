package datamigration

import (
	"log/slog"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jsonopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/defaultconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
)

func EnsureDefaultData() {
	defaultCategory := category.Get(1)
	if defaultCategory.Id == 0 {
		defaultCategory.Name = "YourTJHub"
		defaultCategory.Desc = "🦢 大鹅栖息地 | 自由漫谈的江湖茶馆"
		category.SaveOrCreateById(&defaultCategory)
		slog.Info("created default category")
	}

	configEntity := pageConfig.GetByPageType(pageConfig.FriendShipLinks)
	if configEntity.Id == 0 {
		configEntity.PageType = pageConfig.FriendShipLinks
		configEntity.Config = jsonopt.Encode(defaultconfig.GetDefaultFriendLinksConfig())
		pageConfig.CreateOrSave(&configEntity)
	}
}
