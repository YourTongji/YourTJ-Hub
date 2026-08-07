# 双端像素级比对记录(web 窄屏 375px vs 移动端)

日期:2026-08-08
方法:web 以 headless 浏览器 375×812 移动视口截取 `http://127.0.0.1:5234`(本地后端 + 最新 `resource/static/dist`);移动端以 Flutter golden 基线(390×844, Roboto + Noto CJK 打包字体)与 iOS/Android 模拟器真实截图对照。截图见 `screenshots/web/*.png` 与 `screenshots/android_*.png` / `screenshots/page4_mobile_home_ios.png`。

## 逐页比对结论

### 1. 首页(home)
- **web 截图**:`screenshots/web/home.png`
- **一致项**:话题行结构(标题 15px 加粗 + 分类 chip(色点+名称) + 单行描述截断 + 元信息行(参与者头像/相对时间/回复图标+数))、行间 hairline 分隔、排序 tabs(latest/hot/popular, 选中态深色 pill)、header 64px 品牌区。
- **差异项与接受理由**:
  - web 顶部有「+ New topic」全宽按钮,移动端由底部 publish tab + FAB 承担 → 移动端形态差异,已接受(FAB 为移动端入口,用户锁定保留)。
  - web 显示作者名,mobile fixture 无作者名(测试数据) → 数据差异,非样式差异。
  - 移动端元信息时间「3 小时前」为动态渲染,截图时刻不同 → 时间文案漂移,golden 用固定 past 时间戳(2025-01-15)保证确定性。

### 2. 话题详情(topic)
- **web 截图**:`screenshots/web/topic.png`
- **一致项**:标题区(大标题+作者行(头像+昵称+时钟时间)+分类点+统计)、楼层结构(左侧头像列+作者+#postNo+时间+正文+操作按钮行 Like/Bookmark/Watch/Report)、底部浮动「1 / 1」楼号按钮(未登录态仅楼号)。
- **差异项与接受理由**:
  - web 未登录时浮动条仅楼号按钮;移动端登录后显示完整胶囊(楼号+赞/藏+参与讨论) → 登录态差异,移动端 Android 截图已证实完整胶囊渲染(见 `screenshots/android_topic.png`)。
  - web 回复区在无回复时显示「All replies shown」+ 相关话题列表;移动端同数据下 GfMessageBubble 空列表 → 空态语义一致,结构差异可接受。

### 3. 登录(login)
- **web 截图**:`screenshots/web/login.png`
- **一致项**:品牌 logo + 标题/副标题、**segmented 切换(Log in/Sign up,选中白底蓝字)** ← 移动端 GfSegmented 完全对应、表单(用户名/密码带 icon + 圆角边框)、全宽蓝色主按钮、社交登录区(表单下方 2 列网格)、右上角语言切换。
- **差异项与接受理由**:
  - web 有验证码字段(后端开启时);移动端登录页按 AuthController 状态条件渲染验证码 → 行为一致,视觉由条件触发。
  - 移动端无「OR CONTINUE WITH」文案的社交区? → 移动端保留社交按钮区,文案按移动端 i18n 简化为图标+名称,形态一致。

### 4. 分类页(category)
- **web 截图**:`screenshots/web/category.png`
- **一致项**:PageHeader(分类名+色点+CATEGORY 徽章+描述)、排序 tabs、话题列表行(标题+摘要+元信息行+分隔线)。
- **差异项与接受理由**:web 有「+ New topic」按钮;移动端分类页保留 GfTabBar + 列表,发布入口统一走底部 publish → 移动端形态差异。

### 5. 搜索(search)
- **web 截图**:`screenshots/web/search.png`
- **一致项**:全宽搜索框(h-10 圆角+Search 按钮)、空态(图标+标题+描述)与移动端 GfEmpty 语义一致。
- **差异项与接受理由**:web scope tabs 在输入关键词后才出现,截图为空态未显示;移动端 GfTabBar scope 常驻 → 移动端更直接,可接受。

### 6. 通知(notifications)
- **web 截图**:`screenshots/web/notifications.png`
- **一致项**:PageHeader(标题+Mark all read 按钮)、All/Unread 分段 pill tabs、空态(铃铛图标+No notifications)。
- **差异项与接受理由**:web 未读数 0 时 Unread tab 无计数徽章;移动端 GfNotificationRow 有未读竖条+蓝点样式 → 行级视觉以 golden 覆盖(notifications_page.png,含未读行),空态语义一致。

### 7. 消息(messages)
- **web 截图**:`screenshots/web/messages.png`
- **一致项**:标题+新建按钮、搜索框、空态(图标+No conversations+New message CTA)与移动端 GfEmpty 语义一致。
- **差异项与接受理由**:web 空态 CTA 蓝色实心;移动端空态 GfEmpty 无按钮 → 移动端简化,可接受。会话行视觉由 golden(messages_page.png)与 GfConversationRow 组件基线覆盖。

### 8. 设置(settings)
- **web 截图**:`screenshots/web/settings.png`
- **一致项**:下划线式可横滚 tab 栏(Profile/Account/Privacy/Bindings/Security)← 移动端 GfTabBar 5 tab 对应、Profile tab 表单行(Username/Email 只读+Edit、Display name、Language、Bio/Signature textarea、Social profiles 6 行)。
- **差异项与接受理由**:web 为标签在上/控件在下纵向表单,移动端为 GfSettingRow 单行结构(icon+title+trailing) → 移动端形态差异,已接受(移动端列表式更符合习惯)。

### 9. 草稿(drafts)
- **web 截图**:`screenshots/web/drafts.png`
- **一致项**:PageHeader(标题+0 drafts 徽章+New draft)、空态(图标+No drafts yet+说明+CTA)与移动端 GfEmpty 语义一致。
- **差异项与接受理由**:web 顶部 outline New draft + 空态内 primary New draft 双入口;移动端草稿页无新建入口(移动端在 publish 页存草稿) → 移动端简化,可接受。行视觉由 GfDraftRow 组件基线覆盖。

### 10. 发布(publish)
- **web 截图**:`screenshots/web/publish.png`
- **一致项**:PageHeader(Publish topic+副标题)、Title 输入、Category chip(色点+名称+Up to 3)、Body 编辑器(工具栏+Editor/Markdown 切换+Preview)、底部 Cancel/Save draft/Publish topic 三按钮。
- **差异项与接受理由**:web 为 Quill 富文本+Markdown 双模式;移动端为 Quill 编辑器+工具栏(无 Markdown 源码模式) → 移动端简化,已接受(Blueprint 非目标:不实现 VisualMarkdownEditor)。

### 11. 用户主页(profile)
- **web 截图**:`screenshots/web/profile.png`
- **一致项**:封面+头像上叠+昵称+Admin/Online 徽章+@名+bio、操作行(Edit profile)、4 列统计网格(2 行 8 项)、下划线 tab 栏(Summary/Activity/Badges/Bookmarks)。
- **差异项与接受理由**:web 统计 4×2=8 项在 tab 内容区;移动端 GfUserCard 统计为单行 4 项 → 移动端形态差异,已接受。

## 全局接受理由

- **FAB vs web 无 FAB**:web 移动端发帖入口为列表顶部按钮;移动端保留 FAB(publish tab 冗余入口)——用户锁定项,已在 Blueprint 决策记录。
- **移动端单行列表式 vs web 纵向表单**:GfSettingRow/GfConversationRow 等采用移动端标准列表式,语义字段一一对应。
- **空态语义一致**:web 与移动端均使用 图标+标题+描述 空态(GfEmpty),仅 CTA 按钮有无差异。
- **时间文案**:web 用浏览器本地时区显示;移动端 format.dart 已改为时区无关的字符串字段提取,与后端展示字符串一致。
