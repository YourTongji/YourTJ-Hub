# Route → Contract 覆盖矩阵

<!-- 本文件由 `pnpm run check:coverage`（scripts/check-route-coverage.mjs）生成，请勿手改。 -->

路由快照来自 `TestRoutesSnapshot`（`fixtures/routes-snapshot.json`，默认配置装配，不含 OIDC `/api/oauth/*` 端点——OIDC 另有专项）。

- 快照路由总数：270
- /api JSON 路由：214，已入契约：212（99%），已知未覆盖：0
- 非 API 排除路由：58

## 已覆盖（212）

| Method | Path | operationId |
| --- | --- | --- |
| DELETE | `/api/forum/course-reviews/:reviewId` | `deleteCourseReview` |
| DELETE | `/api/forum/course-reviews/:reviewId/dislike` | `unmarkReviewDislike` |
| DELETE | `/api/forum/course-reviews/:reviewId/helpful` | `unmarkReviewHelpful` |
| GET | `/api/admin/ai-summary-settings` | `adminGetAiSummarySettings` |
| GET | `/api/admin/announcement` | `adminGetAnnouncement` |
| GET | `/api/admin/badges` | `adminListBadges` |
| GET | `/api/admin/data/export/download/:taskId` | `adminDownloadExportTask` |
| GET | `/api/admin/data/export/tasks` | `adminListExportTasks` |
| GET | `/api/admin/friend-links` | `adminGetFriendLinks` |
| GET | `/api/admin/get-all-role-item` | `adminGetAllRoleItem` |
| GET | `/api/admin/http-notify-settings` | `adminGetHttpNotifySettings` |
| GET | `/api/admin/mail-settings` | `adminGetMailSettings` |
| GET | `/api/admin/mcp-settings` | `adminGetMcpSettings` |
| GET | `/api/admin/onesystem-settings` | `adminGetOnesystemSettings` |
| GET | `/api/admin/pk/sync-status` | `adminGetPkSyncStatus` |
| GET | `/api/admin/posting-settings` | `adminGetPostingSettings` |
| GET | `/api/admin/privacy-policy` | `adminGetPrivacyPolicy` |
| GET | `/api/admin/rate-limit-settings` | `adminGetRateLimitSettings` |
| GET | `/api/admin/schedule-settings` | `adminGetScheduleSettings` |
| GET | `/api/admin/security-settings` | `adminGetSecuritySettings` |
| GET | `/api/admin/server-version` | `adminGetServerVersion` |
| GET | `/api/admin/site-chrome` | `adminGetSiteChrome` |
| GET | `/api/admin/site-settings` | `adminGetSiteSettings` |
| GET | `/api/admin/site-theme` | `adminGetSiteTheme` |
| GET | `/api/admin/sponsors` | `adminGetSponsors` |
| GET | `/api/admin/storage-migrate-tasks` | `adminListStorageMigrateTasks` |
| GET | `/api/admin/storage-settings` | `adminGetStorageSettings` |
| GET | `/api/admin/terms-of-service` | `adminGetTermsOfService` |
| GET | `/api/admin/wiki/sync/cdn` | `getWikiAssetCDN` |
| GET | `/api/admin/wiki/sync/runs` | `listWikiSyncRuns` |
| GET | `/api/admin/wiki/sync/status` | `getWikiSyncStatus` |
| GET | `/api/admin/wiki/sync/webhook-secret` | `getWikiWebhookSecret` |
| GET | `/api/admin/wiki/tree` | `getAdminWikiTree` |
| GET | `/api/forum/courses` | `listCourses` |
| GET | `/api/forum/courses/:courseId` | `getCourse` |
| GET | `/api/forum/courses/:courseId/related` | `getCourseRelated` |
| GET | `/api/forum/courses/:courseId/reviews` | `listCourseReviews` |
| GET | `/api/forum/courses/:courseId/summary` | `getCourseSummary` |
| GET | `/api/forum/get-site-statistics` | `getSiteStatistics` |
| GET | `/api/forum/notifications` | `getNotifications` |
| GET | `/api/forum/posts/revisions` | `getPostRevisions` |
| GET | `/api/forum/posts/window` | `getPostWindow` |
| GET | `/api/forum/search` | `searchForum` |
| GET | `/api/forum/unread-status` | `getUnreadStatus` |
| GET | `/api/forum/user/deleted-content` | `deletedContentList` |
| GET | `/api/forum/user/my-content` | `myContentList` |
| GET | `/api/get-captcha` | `getCaptcha` |
| GET | `/api/login-public-key` | `getLoginPublicKey` |
| GET | `/api/oauth/bindings` | `getOAuthBindings` |
| GET | `/api/pk/calendars` | `pkListCalendars` |
| GET | `/api/pk/campuses` | `pkListCampuses` |
| GET | `/api/pk/course-review-brief` | `pkGetCourseReviewBrief` |
| GET | `/api/pk/faculties` | `pkListFaculties` |
| GET | `/api/pk/latest-update` | `pkGetLatestUpdate` |
| GET | `/api/user-card` | `getUserCard` |
| GET | `/api/user/sessions` | `listSessions` |
| GET | `/api/user/totp/status` | `getTotpStatus` |
| GET | `/api/v1/agent/me` | `agentMe` |
| GET | `/api/v1/agent/search` | `agentSearch` |
| GET | `/api/v1/agent/topics` | `agentTopicList` |
| GET | `/api/v1/agent/topics/:topicId/posts` | `agentPostList` |
| GET | `/api/wiki/home` | `getWikiHome` |
| GET | `/api/wiki/namespaces` | `listWikiNamespaces` |
| GET | `/api/wiki/search` | `searchWikiSearch` |
| GET | `/api/wiki/tree` | `getWikiTree` |
| PATCH | `/api/forum/course-reviews/:reviewId` | `updateCourseReview` |
| POST | `/api/admin/agent-create` | `adminAgentCreate` |
| POST | `/api/admin/agent-disable` | `adminAgentDisable` |
| POST | `/api/admin/agent-list` | `adminAgentList` |
| POST | `/api/admin/agent-rotate-token` | `adminAgentRotateToken` |
| POST | `/api/admin/agent-update` | `adminAgentUpdate` |
| POST | `/api/admin/ai-summary-models` | `adminListAiSummaryModels` |
| POST | `/api/admin/badge-delete` | `adminDeleteBadge` |
| POST | `/api/admin/badge-save` | `adminSaveBadge` |
| POST | `/api/admin/category-delete` | `adminCategoryDelete` |
| POST | `/api/admin/category-list` | `adminCategoryList` |
| POST | `/api/admin/category-moderator-add` | `adminCategoryModeratorAdd` |
| POST | `/api/admin/category-moderator-delete` | `adminCategoryModeratorDelete` |
| POST | `/api/admin/category-save` | `adminCategorySave` |
| POST | `/api/admin/data/export` | `adminCreateExportTask` |
| POST | `/api/admin/data/import` | `adminImportData` |
| POST | `/api/admin/file-resources` | `adminListFileResources` |
| POST | `/api/admin/get-permission-list` | `adminGetPermissionList` |
| POST | `/api/admin/global-moderator-add` | `adminGlobalModeratorAdd` |
| POST | `/api/admin/global-moderator-delete` | `adminGlobalModeratorDelete` |
| POST | `/api/admin/global-moderator-list` | `adminGlobalModeratorList` |
| POST | `/api/admin/img-upload` | `adminUploadImage` |
| POST | `/api/admin/opt-record-page` | `adminOptRecordPage` |
| POST | `/api/admin/pk/sync-calendar` | `adminSyncPkCalendar` |
| POST | `/api/admin/posts/delete` | `adminDeletePost` |
| POST | `/api/admin/publish-site-theme` | `adminPublishSiteTheme` |
| POST | `/api/admin/review-action` | `adminReviewAction` |
| POST | `/api/admin/review-queue` | `adminListReviewQueue` |
| POST | `/api/admin/role-delete` | `adminRoleDelete` |
| POST | `/api/admin/role-list` | `adminRoleList` |
| POST | `/api/admin/role-save` | `adminRoleSave` |
| POST | `/api/admin/save-ai-summary-settings` | `adminSaveAiSummarySettings` |
| POST | `/api/admin/save-announcement` | `adminSaveAnnouncement` |
| POST | `/api/admin/save-friend-links` | `adminSaveFriendLinks` |
| POST | `/api/admin/save-http-notify-settings` | `adminSaveHttpNotifySettings` |
| POST | `/api/admin/save-mail-settings` | `adminSaveMailSettings` |
| POST | `/api/admin/save-mcp-settings` | `adminSaveMcpSettings` |
| POST | `/api/admin/save-onesystem-settings` | `adminSaveOnesystemSettings` |
| POST | `/api/admin/save-posting-settings` | `adminSavePostingSettings` |
| POST | `/api/admin/save-privacy-policy` | `adminSavePrivacyPolicy` |
| POST | `/api/admin/save-rate-limit-settings` | `adminSaveRateLimitSettings` |
| POST | `/api/admin/save-schedule-settings` | `adminSaveScheduleSettings` |
| POST | `/api/admin/save-security-settings` | `adminSaveSecuritySettings` |
| POST | `/api/admin/save-site-chrome` | `adminSaveSiteChrome` |
| POST | `/api/admin/save-site-settings` | `adminSaveSiteSettings` |
| POST | `/api/admin/save-site-theme` | `adminSaveSiteTheme` |
| POST | `/api/admin/save-sponsors` | `adminSaveSponsors` |
| POST | `/api/admin/save-storage-settings` | `adminSaveStorageSettings` |
| POST | `/api/admin/save-terms-of-service` | `adminSaveTermsOfService` |
| POST | `/api/admin/save-user-badges` | `adminSaveUserBadges` |
| POST | `/api/admin/storage-migrate-task` | `adminCreateStorageMigrateTask` |
| POST | `/api/admin/test-mail-connection` | `adminTestMailConnection` |
| POST | `/api/admin/test-storage-connection` | `adminTestStorageConnection` |
| POST | `/api/admin/topics/categories-edit` | `adminEditTopicCategories` |
| POST | `/api/admin/topics/delete` | `adminDeleteTopic` |
| POST | `/api/admin/topics/edit` | `adminEditTopic` |
| POST | `/api/admin/topics/list` | `adminListTopics` |
| POST | `/api/admin/topics/pin-edit` | `adminEditTopicPin` |
| POST | `/api/admin/topics/restore` | `adminRestoreTopic` |
| POST | `/api/admin/topics/source` | `adminGetTopicSource` |
| POST | `/api/admin/traffic-overview` | `adminTrafficOverview` |
| POST | `/api/admin/user-badge-options` | `adminUserBadgeOptions` |
| POST | `/api/admin/user-edit` | `adminEditUser` |
| POST | `/api/admin/user-list` | `adminUserList` |
| POST | `/api/admin/wiki/sync` | `runWikiSync` |
| POST | `/api/admin/wiki/sync/cdn` | `saveWikiAssetCDN` |
| POST | `/api/admin/wiki/sync/webhook-secret` | `saveWikiWebhookSecret` |
| POST | `/api/auth/:provider/unbind` | `unbindOAuth` |
| POST | `/api/auth/oidc/exchange` | `exchangeMobileOidcCode` |
| POST | `/api/auth/totp/verify` | `verifyTotpLogin` |
| POST | `/api/change-password` | `changePassword` |
| POST | `/api/forgot-password` | `forgotPassword` |
| POST | `/api/forum/chat/mark-read` | `markChatRead` |
| POST | `/api/forum/chat/messages` | `getChatMessages` |
| POST | `/api/forum/chat/send` | `sendChatMessage` |
| POST | `/api/forum/course-reviews` | `createCourseReview` |
| POST | `/api/forum/course-reviews/:reviewId/reports` | `reportCourseReview` |
| POST | `/api/forum/courses/bookmark` | `bookmarkCourse` |
| POST | `/api/forum/follow-user` | `followUser` |
| POST | `/api/forum/moderation/course-create` | `adminCourseCreate` |
| POST | `/api/forum/moderation/course-delete` | `adminCourseDelete` |
| POST | `/api/forum/moderation/course-list` | `adminCourseList` |
| POST | `/api/forum/moderation/course-review-delete` | `adminReviewDelete` |
| POST | `/api/forum/moderation/course-review-edit` | `adminReviewUpdate` |
| POST | `/api/forum/moderation/course-review-list` | `adminReviewList` |
| POST | `/api/forum/moderation/course-review-reports` | `moderationCourseReviewReportList` |
| POST | `/api/forum/moderation/course-review-reveal` | `moderationCourseReviewReveal` |
| POST | `/api/forum/moderation/course-review-status` | `moderationCourseReviewStatus` |
| POST | `/api/forum/moderation/course-stats-rebuild` | `adminCourseStatsRebuild` |
| POST | `/api/forum/moderation/course-update` | `adminCourseUpdate` |
| POST | `/api/forum/moderation/logs` | `listModerationLogs` |
| POST | `/api/forum/moderation/post-status` | `moderationUpdatePostStatus` |
| POST | `/api/forum/moderation/report-status` | `moderationUpdateReportStatus` |
| POST | `/api/forum/moderation/reports` | `listModerationReports` |
| POST | `/api/forum/moderation/topic-status` | `moderationUpdateTopicStatus` |
| POST | `/api/forum/moderation/view-deleted-content` | `viewDeletedContent` |
| POST | `/api/forum/notification/mark-all-read` | `markAllNotificationsRead` |
| POST | `/api/forum/notification/mark-read` | `markNotificationRead` |
| POST | `/api/forum/posts/bookmark` | `bookmarkPost` |
| POST | `/api/forum/posts/create` | `createPost` |
| POST | `/api/forum/posts/delete` | `deletePost` |
| POST | `/api/forum/posts/like` | `likePost` |
| POST | `/api/forum/posts/update` | `updatePost` |
| POST | `/api/forum/report` | `createReport` |
| POST | `/api/forum/topics/bookmark` | `bookmarkTopic` |
| POST | `/api/forum/topics/delete` | `deleteTopic` |
| POST | `/api/forum/topics/like` | `likeTopic` |
| POST | `/api/forum/topics/status` | `updateTopicStatus` |
| POST | `/api/forum/topics/watch` | `watchTopic` |
| POST | `/api/forum/topics/write` | `writeTopic` |
| POST | `/api/forum/user/account-close` | `closeAccount` |
| POST | `/api/forum/user/content-batch-delete` | `batchDeleteContent` |
| POST | `/api/forum/user/content-event` | `reportContentEvent` |
| POST | `/api/forum/user/content-privacy-erase` | `privacyEraseContent` |
| POST | `/api/forum/user/content-purge` | `purgeContent` |
| POST | `/api/forum/user/content-restore` | `restoreContent` |
| POST | `/api/login` | `login` |
| POST | `/api/logout` | `logout` |
| POST | `/api/pk/course-details` | `pkFindCourseDetails` |
| POST | `/api/pk/course-info-sync` | `pkSyncCourseInfo` |
| POST | `/api/pk/course-search` | `pkSearchCourses` |
| POST | `/api/pk/courses-by-major` | `pkFindCoursesByMajor` |
| POST | `/api/pk/courses-by-nature` | `pkFindCoursesByNature` |
| POST | `/api/pk/courses-by-time` | `pkFindCoursesByTime` |
| POST | `/api/pk/grades` | `pkFindGrades` |
| POST | `/api/pk/majors` | `pkFindMajors` |
| POST | `/api/pk/optional-types` | `pkFindOptionalTypes` |
| POST | `/api/register` | `register` |
| POST | `/api/resend-activation-email` | `resendActivationEmail` |
| POST | `/api/reset-password` | `resetPassword` |
| POST | `/api/set-preset-avatar` | `setPresetAvatar` |
| POST | `/api/set-user-email` | `setUserEmail` |
| POST | `/api/set-user-info` | `setUserInfo` |
| POST | `/api/set-user-name` | `setUserName` |
| POST | `/api/set-user-profile-cover` | `setUserProfileCover` |
| POST | `/api/upload-avatar` | `uploadAvatar` |
| POST | `/api/user/sessions/revoke` | `revokeSession` |
| POST | `/api/user/sessions/revoke-all` | `revokeAllSessions` |
| POST | `/api/user/totp/disable` | `disableTotp` |
| POST | `/api/user/totp/enable` | `enableTotp` |
| POST | `/api/user/totp/setup` | `setupTotp` |
| POST | `/api/v1/agent/topics` | `agentWriteTopic` |
| POST | `/api/v1/agent/topics/:topicId/posts` | `agentCreatePost` |
| POST | `/api/wear-badge` | `wearBadge` |
| POST | `/api/wiki/webhook` | `wikiWebhook` |
| PUT | `/api/forum/course-reviews/:reviewId/dislike` | `markReviewDislike` |
| PUT | `/api/forum/course-reviews/:reviewId/helpful` | `markReviewHelpful` |

## 已知未覆盖（0）

| Method | Path | 归属切片 |
| --- | --- | --- |

## 排除（非 JSON API，58）

| Method | Path | 原因 |
| --- | --- | --- |
| CONNECT | `/mcp` | MCP streamable HTTP 端点（Any 展开多方法），走 MCP 自有协议契约 |
| DELETE | `/mcp` | MCP streamable HTTP 端点（Any 展开多方法），走 MCP 自有协议契约 |
| GET | `/` | SSR 页面（GoHTML 三模渲染），非 JSON API |
| GET | `/activate` | SSR 页面（GoHTML 三模渲染），非 JSON API |
| GET | `/admin` | SSR 页面（GoHTML 三模渲染），非 JSON API |
| GET | `/admin/*path` | SSR 页面（GoHTML 三模渲染），非 JSON API |
| GET | `/api/auth/:provider` | goth 浏览器 302 重定向流程（HTML/重定向，非 JSON API）；OAuth/OIDC 协议面由专项契约轨道跟进 |
| GET | `/api/auth/:provider/callback` | goth 浏览器 302 重定向流程（HTML 错误页/重定向，非 JSON API）；OAuth/OIDC 协议面由专项契约轨道跟进 |
| GET | `/assets/*filepath` | go:embed 静态资源（StaticFS 展开 GET+HEAD） |
| GET | `/c/:slug/:id` | SSR 页面（GoHTML 三模渲染），非 JSON API |
| GET | `/c/:slug/:id/l/:sort` | SSR 页面（GoHTML 三模渲染），非 JSON API |
| GET | `/courses` | SSR 页面（GoHTML 三模渲染），非 JSON API |
| GET | `/courses/:courseId` | SSR 页面（GoHTML 三模渲染），非 JSON API |
| GET | `/drafts` | SSR 页面（GoHTML 三模渲染），非 JSON API |
| GET | `/file/img/*filename` | 上传文件读取服务，非 JSON API |
| GET | `/health` | 健康检查探针，infra 端点非 JSON API |
| GET | `/links` | SSR 页面（GoHTML 三模渲染），非 JSON API |
| GET | `/llms-full.txt` | SEO/机器可读文本输出，非 JSON API |
| GET | `/llms.txt` | SEO/机器可读文本输出，非 JSON API |
| GET | `/login` | SSR 页面（GoHTML 三模渲染），非 JSON API |
| GET | `/mcp` | MCP streamable HTTP 端点（Any 展开多方法），走 MCP 自有协议契约 |
| GET | `/messages` | SSR 页面（GoHTML 三模渲染），非 JSON API |
| GET | `/moderation` | SSR 页面（GoHTML 三模渲染），非 JSON API |
| GET | `/moderation/course-reviews` | SSR 页面（GoHTML 三模渲染），非 JSON API |
| GET | `/moderation/courses` | SSR 页面（GoHTML 三模渲染），非 JSON API |
| GET | `/notifications` | SSR 页面（GoHTML 三模渲染），非 JSON API |
| GET | `/p/post/:id` | SSR 页面（GoHTML 三模渲染），非 JSON API |
| GET | `/p/post/:id/:postNo` | SSR 页面（GoHTML 三模渲染），非 JSON API |
| GET | `/p/posts/:document` | SSR 页面（GoHTML 三模渲染），非 JSON API |
| GET | `/privacy` | SSR 页面（GoHTML 三模渲染），非 JSON API |
| GET | `/publish` | SSR 页面（GoHTML 三模渲染），非 JSON API |
| GET | `/reload` | 开发期模板热重载端点，非 JSON API |
| GET | `/reset-password` | SSR 页面（GoHTML 三模渲染），非 JSON API |
| GET | `/robots.txt` | SEO/机器可读文本输出，非 JSON API |
| GET | `/rss.xml` | SEO/机器可读文本输出，非 JSON API |
| GET | `/schedule` | SSR 页面（GoHTML 三模渲染），非 JSON API |
| GET | `/search` | SSR 页面（GoHTML 三模渲染），非 JSON API |
| GET | `/settings` | SSR 页面（GoHTML 三模渲染），非 JSON API |
| GET | `/site-theme.css` | 动态主题 CSS 输出，非 JSON API |
| GET | `/sitemap.xml` | SEO/机器可读文本输出，非 JSON API |
| GET | `/sponsors` | SSR 页面（GoHTML 三模渲染），非 JSON API |
| GET | `/static/*filepath` | 静态资源（StaticFS 展开 GET+HEAD） |
| GET | `/terms` | SSR 页面（GoHTML 三模渲染），非 JSON API |
| GET | `/theme-preview` | SSR 页面（GoHTML 三模渲染），非 JSON API |
| GET | `/u/:userId` | SSR 页面（GoHTML 三模渲染），非 JSON API |
| GET | `/u/:userId/:section` | SSR 页面（GoHTML 三模渲染），非 JSON API |
| GET | `/u/:userId/:section/:subsection` | SSR 页面（GoHTML 三模渲染），非 JSON API |
| GET | `/wiki` | SSR 页面（GoHTML 三模渲染），非 JSON API |
| GET | `/wiki/*path` | SSR 页面（GoHTML 三模渲染），非 JSON API |
| HEAD | `/assets/*filepath` | go:embed 静态资源（StaticFS 展开 GET+HEAD） |
| HEAD | `/mcp` | MCP streamable HTTP 端点（Any 展开多方法），走 MCP 自有协议契约 |
| HEAD | `/static/*filepath` | 静态资源（StaticFS 展开 GET+HEAD） |
| OPTIONS | `/mcp` | MCP streamable HTTP 端点（Any 展开多方法），走 MCP 自有协议契约 |
| PATCH | `/mcp` | MCP streamable HTTP 端点（Any 展开多方法），走 MCP 自有协议契约 |
| POST | `/file/img-upload` | multipart 文件上传端点，不纳入 JSON API 契约 |
| POST | `/mcp` | MCP streamable HTTP 端点（Any 展开多方法），走 MCP 自有协议契约 |
| PUT | `/mcp` | MCP streamable HTTP 端点（Any 展开多方法），走 MCP 自有协议契约 |
| TRACE | `/mcp` | MCP streamable HTTP 端点（Any 展开多方法），走 MCP 自有协议契约 |
