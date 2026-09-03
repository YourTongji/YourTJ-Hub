package api

import (
	"fmt"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/urlutil"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
)

// admin URL 字段校验（issue #409）：四个页面配置保存接口在落库前对全部
// 管理员可配置 URL 字段做 canonical 校验并按字段策略归一化（trim）。
// 校验失败返回首个违规字段名（空串 = 全部通过），由保存接口转为稳定错误。

func validateSiteSettingsURLs(settings *pageConfig.SiteSettingsConfig) string {
	if url, ok := urlutil.Canonicalize(urlutil.External, settings.SiteUrl); !ok {
		return "settings.siteUrl"
	} else {
		settings.SiteUrl = url
	}
	if url, ok := urlutil.Canonicalize(urlutil.Image, settings.SiteLogo); !ok {
		return "settings.siteLogo"
	} else {
		settings.SiteLogo = url
	}
	return ""
}

func validateFriendLinksURLs(groups []pageConfig.FriendLinksGroup) string {
	for i := range groups {
		for j := range groups[i].Links {
			link := &groups[i].Links[j]
			if url, ok := urlutil.Canonicalize(urlutil.External, link.Url); !ok {
				return fmt.Sprintf("linksInfo[%d].links[%d].url", i, j)
			} else {
				link.Url = url
			}
			if url, ok := urlutil.Canonicalize(urlutil.Image, link.LogoUrl); !ok {
				return fmt.Sprintf("linksInfo[%d].links[%d].logoUrl", i, j)
			} else {
				link.LogoUrl = url
			}
		}
	}
	return ""
}

func validateSponsorsURLs(config *pageConfig.SponsorsConfig) string {
	levels := [][]pageConfig.SponsorItem{
		config.Sponsors.Level0,
		config.Sponsors.Level1,
		config.Sponsors.Level2,
		config.Sponsors.Level3,
	}
	for level, items := range levels {
		for j := range items {
			item := &items[j]
			if url, ok := urlutil.Canonicalize(urlutil.External, item.Link); !ok {
				return fmt.Sprintf("sponsorsInfo.sponsors.level%d[%d].link", level, j)
			} else {
				item.Link = url
			}
			if url, ok := urlutil.Canonicalize(urlutil.Image, item.AvatarUrl); !ok {
				return fmt.Sprintf("sponsorsInfo.sponsors.level%d[%d].avatarUrl", level, j)
			} else {
				item.AvatarUrl = url
			}
		}
	}
	if url, ok := urlutil.Canonicalize(urlutil.Contact, config.Contact.ButtonLink); !ok {
		return "sponsorsInfo.contact.buttonLink"
	} else {
		config.Contact.ButtonLink = url
	}
	return ""
}

func validateSiteChromeURLs(config *pageConfig.SiteChromeConfig) string {
	itemLists := [][]pageConfig.ChromeItem{
		config.Header,
		config.MainMenu,
		config.Resources,
	}
	for _, group := range config.SidebarGroups {
		itemLists = append(itemLists, group.Items)
	}
	for listIndex, items := range itemLists {
		for j := range items {
			if url, ok := urlutil.Canonicalize(urlutil.SiteLink, items[j].URL); !ok {
				return fmt.Sprintf("settings.%d[%d].url", listIndex, j)
			} else {
				items[j].URL = url
			}
		}
	}
	for i := range config.FooterInfo.List {
		if url, ok := urlutil.Canonicalize(urlutil.SiteLink, config.FooterInfo.List[i].Url); !ok {
			return fmt.Sprintf("settings.footerInfo.list[%d].url", i)
		} else {
			config.FooterInfo.List[i].Url = url
		}
	}
	if url, ok := urlutil.Canonicalize(urlutil.Image, config.BrandImage); !ok {
		return "settings.brandImage"
	} else {
		config.BrandImage = url
	}
	return ""
}
