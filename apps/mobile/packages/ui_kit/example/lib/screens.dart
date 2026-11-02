import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';

import 'catalog.dart';
import 'mock.dart';

// ---------------------------------------------------------------------------
// 编辑资料
// ---------------------------------------------------------------------------

class ProfileEditScreen extends StatefulWidget {
  const ProfileEditScreen({super.key});

  @override
  State<ProfileEditScreen> createState() => _ProfileEditScreenState();
}

class _ProfileEditScreenState extends State<ProfileEditScreen> {
  late final ScrollController _scroll = ScrollController(
    initialScrollOffset: galleryInitialScroll,
  );
  final _nickname = TextEditingController(text: '小济同学');
  final _bio = TextEditingController(
    text: '土木工程 · 2023 级。喜欢摄影、骑行和在图书馆三楼靠窗的位置写代码。',
  );
  final _signature = TextEditingController(text: '同舟共济，自强不息');
  final _siteName = TextEditingController(text: '我的博客');
  final _site = TextEditingController(text: 'https://xiaoji.dev');
  final _github = TextEditingController(text: 'xiaoji');
  final _bilibili = TextEditingController();
  final _zhihu = TextEditingController();

  @override
  void dispose() {
    for (final c in [
      _nickname,
      _bio,
      _signature,
      _siteName,
      _site,
      _github,
      _bilibili,
      _zhihu,
    ]) {
      c.dispose();
    }
    _scroll.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final c = GfTheme.colorsOf(context);
    final t = GfTheme.typographyOf(context);
    return Scaffold(
      backgroundColor: c.canvasGrouped,
      appBar: GfAppBar(
        backgroundColor: c.canvasGrouped,
        showDivider: false,
        leadingWidth: 72,
        leading: GfNavTextAction(
          label: '取消',
          onPressed: () => Navigator.maybePop(context),
        ),
        title: const GfText('编辑资料'),
        actions: [
          Padding(
            padding: const EdgeInsets.only(right: GfSpace.md),
            child: Center(
              child: GfButton(
                label: '保存',
                size: GfButtonSize.small,
                onPressed: () {},
              ),
            ),
          ),
        ],
      ),
      body: ListView(
        controller: _scroll,
        padding: const EdgeInsets.only(bottom: 40),
        children: [
          // Cover + avatar card.
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 4, 16, 0),
            child: ClipRRect(
              borderRadius: BorderRadius.circular(GfRadius.lg),
              child: ColoredBox(
                color: c.surfaceGrouped,
                child: Column(
                  children: [
                    SizedBox(
                      height: 132,
                      child: Stack(
                        fit: StackFit.expand,
                        children: [
                          GfNetworkImage(
                            'https://picsum.photos/id/1031/1200/500',
                            fit: BoxFit.cover,
                          ),
                          Positioned(
                            right: 12,
                            top: 12,
                            child: GfGlassIconButton(
                              symbol: 'camera',
                              tooltip: '更换封面',
                              onPressed: () {},
                            ),
                          ),
                        ],
                      ),
                    ),
                    // Avatar overlaps the cover by half its height; the
                    // column below reserves only the overlapping half.
                    SizedBox(
                      height: 92,
                      child: Stack(
                        clipBehavior: Clip.none,
                        alignment: Alignment.topCenter,
                        children: [
                          Positioned(
                            top: -46,
                            child: Stack(
                              clipBehavior: Clip.none,
                              children: [
                                Container(
                                  padding: const EdgeInsets.all(4),
                                  decoration: BoxDecoration(
                                    color: c.surfaceGrouped,
                                    shape: BoxShape.circle,
                                  ),
                                  child: GfAvatar(src: avatar(68), size: 84),
                                ),
                                Positioned(
                                  right: 2,
                                  bottom: 2,
                                  child: Container(
                                    width: 30,
                                    height: 30,
                                    decoration: BoxDecoration(
                                      color: c.primary,
                                      shape: BoxShape.circle,
                                      border: Border.all(
                                        color: c.surfaceGrouped,
                                        width: 3,
                                      ),
                                    ),
                                    alignment: Alignment.center,
                                    child: GfSymbol(
                                      'camera',
                                      size: 14,
                                      color: c.primaryContent,
                                    ),
                                  ),
                                ),
                              ],
                            ),
                          ),
                          Positioned(
                            bottom: 16,
                            child: GfText(
                              '更换头像',
                              style: t.subhead.copyWith(
                                color: c.primary,
                                fontWeight: FontWeight.w600,
                              ),
                            ),
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ),
          const SizedBox(height: GfSpace.section),
          GfListSection(
            header: '基本信息',
            children: [
              GfFieldRow(label: '昵称', controller: _nickname, maxLength: 20),
              GfSettingRow(title: '用户名', value: '@xiaoji', onTap: () {}),
              GfFieldRow(
                label: '简介',
                controller: _bio,
                maxLines: 4,
                maxLength: 160,
              ),
              GfFieldRow(label: '个性签名', controller: _signature, maxLength: 40),
            ],
          ),
          const SizedBox(height: GfSpace.section),
          GfListSection(
            header: '个人网站',
            children: [
              GfFieldRow(label: '名称', controller: _siteName),
              GfFieldRow(
                label: '链接',
                controller: _site,
                keyboardType: TextInputType.url,
                autocorrect: false,
              ),
            ],
          ),
          const SizedBox(height: GfSpace.section),
          GfListSection(
            header: '社交账号',
            footer: '只填用户名即可，主页上会显示对应图标。',
            dividerIndent: GfListSection.iconRowIndent,
            children: [
              GfFieldRow(
                label: 'GitHub',
                controller: _github,
                leading: const GfSocialIcon('github', size: 22),
              ),
              GfFieldRow(
                label: '哔哩哔哩',
                controller: _bilibili,
                hintText: '用户名或空间链接',
                leading: const GfSocialIcon('bilibili', size: 22),
              ),
              GfFieldRow(
                label: '知乎',
                controller: _zhihu,
                hintText: '用户名',
                leading: const GfSocialIcon('zhihu', size: 22),
              ),
            ],
          ),
          const SizedBox(height: GfSpace.section),
          GfListSection(
            header: '偏好',
            children: [
              GfSettingRow(title: '资料语言', value: '简体中文', onTap: () {}),
              GfSettingRow(title: '展示徽章', value: '3 枚', onTap: () {}),
            ],
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// 设置
// ---------------------------------------------------------------------------

class SettingsScreen extends StatefulWidget {
  const SettingsScreen({super.key});

  @override
  State<SettingsScreen> createState() => _SettingsScreenState();
}

class _SettingsScreenState extends State<SettingsScreen> {
  late final ScrollController _scroll = ScrollController(
    initialScrollOffset: galleryInitialScroll,
  );
  bool _push = true;
  bool _mention = true;

  @override
  void dispose() {
    _scroll.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final c = GfTheme.colorsOf(context);
    final t = GfTheme.typographyOf(context);
    const indent = GfListSection.iconRowIndent;
    return Scaffold(
      backgroundColor: c.canvasGrouped,
      appBar: GfAppBar(
        title: const GfText('设置'),
        backgroundColor: c.canvasGrouped,
        showDivider: false,
      ),
      body: ListView(
        controller: _scroll,
        padding: const EdgeInsets.only(top: 4, bottom: 40),
        children: [
          GfListSection(
            children: [
              InkWell(
                onTap: () {},
                child: Padding(
                  padding: const EdgeInsets.all(GfSpace.lg),
                  child: Row(
                    children: [
                      GfAvatar(src: avatar(68), size: 56),
                      const SizedBox(width: GfSpace.md),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            GfText('小济同学', style: t.title3),
                            const SizedBox(height: 2),
                            GfText(
                              '@xiaoji · 土木工程学院',
                              style: t.subhead.copyWith(color: c.textSecondary),
                            ),
                          ],
                        ),
                      ),
                      GfSymbol(
                        'chevron-right',
                        size: 16,
                        color: c.textPlaceholder,
                      ),
                    ],
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(height: GfSpace.section),
          GfListSection(
            header: '账号与安全',
            dividerIndent: indent,
            children: [
              GfSettingRow(
                symbol: 'mail',
                title: '邮箱',
                value: 'xiaoji@tongji.edu.cn',
                onTap: () {},
              ),
              GfSettingRow(symbol: 'key-round', title: '密码与通行密钥', onTap: () {}),
              GfSettingRow(
                symbol: 'shield-check',
                title: '两步验证',
                value: '已开启',
                onTap: () {},
              ),
              GfSettingRow(
                symbol: 'smartphone',
                title: '登录设备',
                value: '3 台',
                onTap: () {},
              ),
            ],
          ),
          const SizedBox(height: GfSpace.section),
          GfListSection(
            header: '通知',
            footer: '关闭推送后，仍可以在「通知」页查看全部消息。',
            dividerIndent: indent,
            children: [
              GfSwitchRow(
                symbol: 'bell',
                title: '推送通知',
                value: _push,
                onChanged: (v) => setState(() => _push = v),
              ),
              GfSwitchRow(
                symbol: 'at-sign',
                title: '被提及时提醒',
                value: _mention,
                onChanged: (v) => setState(() => _mention = v),
              ),
            ],
          ),
          const SizedBox(height: GfSpace.section),
          GfListSection(
            header: '通用',
            dividerIndent: indent,
            children: [
              GfSettingRow(
                symbol: 'moon',
                title: '深色模式',
                value: '跟随系统',
                onTap: () {},
              ),
              GfSettingRow(
                symbol: 'type',
                title: '字体大小',
                value: '标准',
                onTap: () {},
              ),
              GfSettingRow(
                symbol: 'languages',
                title: '语言',
                value: '简体中文',
                onTap: () {},
              ),
              GfSettingRow(
                symbol: 'archive',
                title: '存储空间',
                value: '128 MB',
                onTap: () {},
              ),
            ],
          ),
          const SizedBox(height: GfSpace.section),
          GfListSection(
            dividerIndent: indent,
            children: [
              GfSettingRow(symbol: 'circle-help', title: '帮助与反馈', onTap: () {}),
              GfSettingRow(
                symbol: 'info',
                title: '关于 YourTJ',
                value: '1.8.0',
                onTap: () {},
              ),
            ],
          ),
          const SizedBox(height: GfSpace.section),
          GfListSection(
            children: [
              InkWell(
                onTap: () {},
                child: SizedBox(
                  height: GfControl.row,
                  child: Center(
                    child: GfText(
                      '退出登录',
                      style: t.body.copyWith(color: c.error),
                    ),
                  ),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// 个人主页
// ---------------------------------------------------------------------------

class ProfileScreen extends StatefulWidget {
  const ProfileScreen({super.key});

  @override
  State<ProfileScreen> createState() => _ProfileScreenState();
}

class _ProfileScreenState extends State<ProfileScreen> {
  late final ScrollController _scroll = ScrollController(
    initialScrollOffset: galleryInitialScroll,
  );
  Object _tab = 'topics';

  @override
  void dispose() {
    _scroll.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final c = GfTheme.colorsOf(context);
    return Scaffold(
      extendBodyBehindAppBar: true,
      appBar: AppBar(
        backgroundColor: Colors.transparent,
        leading: Padding(
          padding: const EdgeInsets.only(left: 6),
          child: GfGlassIconButton(
            symbol: 'chevron-left',
            tooltip: '返回',
            onPressed: () => Navigator.maybePop(context),
          ),
        ),
        leadingWidth: 56,
        actions: [
          GfGlassIconButton(symbol: 'share-2', tooltip: '分享', onPressed: () {}),
          const SizedBox(width: 6),
          GfGlassIconButton(
            symbol: 'ellipsis',
            tooltip: '更多',
            onPressed: () {},
          ),
          const SizedBox(width: 10),
        ],
      ),
      body: ListView(
        controller: _scroll,
        padding: EdgeInsets.zero,
        children: [
          GfUserCard(
            avatarUrl: avatar(68),
            coverUrl: 'https://picsum.photos/id/1031/1200/500',
            coverHeight: GfUserCard.coverHeightFor(
              MediaQuery.sizeOf(context).width,
              topInset: MediaQuery.paddingOf(context).top,
            ),
            name: '小济同学',
            username: 'xiaoji',
            nameBadges: const [
              GfBadge(label: 'Lv.6', variant: GfBadgeVariant.info),
            ],
            bio: '土木工程 · 2023 级。喜欢摄影、骑行和在图书馆三楼靠窗的位置写代码。',
            signature: '同舟共济，自强不息',
            stats: const [('关注', '128'), ('粉丝', '2.4k'), ('获赞', '9,812')],
            statActions: {0: () {}, 1: () {}},
            actions: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                GfIconButton(
                  symbol: 'mail',
                  tooltip: '私信',
                  variant: GfIconButtonVariant.tonal,
                  onPressed: () {},
                ),
                const SizedBox(width: 4),
                GfButton(
                  label: '关注',
                  icon: const GfSymbol('user-round-plus'),
                  onPressed: () {},
                ),
              ],
            ),
          ),
          const SizedBox(height: 4),
          DecoratedBox(
            decoration: BoxDecoration(
              border: Border(
                bottom: BorderSide(
                  color: c.separator,
                  width: gfHairline(context),
                ),
              ),
            ),
            child: GfTabBar(
              tabs: const [
                GfTab(label: '帖子', value: 'topics'),
                GfTab(label: '回复', value: 'replies'),
                GfTab(label: '收藏', value: 'bookmarks'),
              ],
              selected: _tab,
              distribute: true,
              onSelected: (v) => setState(() => _tab = v),
            ),
          ),
          GfTopicCard(
            title: '整理了一份嘉定校区骑行路线',
            description: '从图书馆出发绕校园一圈大约 6 公里，路过银杏大道和人工湖，傍晚光线最好。',
            authorName: '小济同学',
            authorAvatarUrl: avatar(68),
            categories: const [GfTopicCategory(name: '校园生活', color: catCampus)],
            imageUrls: [photo('cycling')],
            activityText: '2 天前',
            replyCount: 24,
            viewCount: 1830,
            likeCount: 212,
            onLike: (target) async => true,
            onBookmark: (target) async => true,
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// 首页
// ---------------------------------------------------------------------------

class HomeScreen extends StatefulWidget {
  const HomeScreen({super.key});

  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> {
  Object _sort = 'latest';
  String _category = 'all';

  @override
  Widget build(BuildContext context) {
    final c = GfTheme.colorsOf(context);
    final categories = [
      ('all', '全部', null),
      ('campus', '校园生活', catCampus),
      ('study', '学习交流', catStudy),
      ('trade', '二手交易', catTrade),
      ('help', '互助问答', catHelp),
    ];
    return Scaffold(
      appBar: GfAppBar(
        showDivider: false,
        leading: Padding(
          padding: const EdgeInsets.all(10),
          child: GfAvatar(src: avatar(68), size: 32),
        ),
        title: const GfLogo(size: 30),
        actions: [
          GfIconButton(
            symbol: 'search',
            tooltip: '搜索',
            color: c.textPrimary,
            onPressed: () {},
          ),
          const SizedBox(width: 4),
        ],
      ),
      body: Stack(
        children: [
          ListView(
            padding: const EdgeInsets.only(bottom: 120),
            children: [
              GfTabBar(
                tabs: const [
                  GfTab(label: '最新', value: 'latest'),
                  GfTab(label: '热门', value: 'hot'),
                  GfTab(label: '关注', value: 'following'),
                ],
                selected: _sort,
                onSelected: (v) => setState(() => _sort = v),
              ),
              Container(
                height: 48,
                decoration: BoxDecoration(
                  border: Border(
                    top: BorderSide(
                      color: c.separator,
                      width: gfHairline(context),
                    ),
                  ),
                ),
                child: ListView(
                  scrollDirection: Axis.horizontal,
                  padding: const EdgeInsets.symmetric(
                    horizontal: 12,
                    vertical: 8,
                  ),
                  children: [
                    for (final (id, name, color) in categories)
                      Padding(
                        padding: const EdgeInsets.only(right: 8),
                        child: GfSelectTag(
                          label: name,
                          symbol: color == null ? null : null,
                          selected: _category == id,
                          onChanged: (_) => setState(() => _category = id),
                        ),
                      ),
                  ],
                ),
              ),
              GfTopicCard(
                title: '四平路校区哪个食堂的早餐最值得推荐？',
                description: '刚转到本部，想问问大家平时早餐都去哪里吃，最好七点前能吃上的。',
                authorName: '林知夏',
                authorAvatarUrl: avatar(47),
                categories: const [
                  GfTopicCategory(name: '校园生活', color: catCampus),
                ],
                imageUrls: const [],
                activityText: '12 分钟前',
                replyCount: 36,
                viewCount: 1204,
                likeCount: 58,
                hot: true,
                unseen: true,
                onLike: (target) async => true,
                onBookmark: (target) async => true,
              ),
              GfTopicCard(
                title: '嘉定校区秋天的银杏大道',
                description: '下课路过随手拍的，今年黄得特别早。',
                authorName: '周屿',
                authorAvatarUrl: avatar(12),
                categories: const [
                  GfTopicCategory(name: '学习交流', color: catStudy),
                ],
                imageUrls: [photo('ginkgo')],
                activityText: '1 小时前',
                replyCount: 8,
                viewCount: 342,
                likeCount: 96,
                liked: true,
                onLike: (target) async => true,
                onBookmark: (target) async => true,
              ),
            ],
          ),
          Positioned(
            right: 16,
            bottom: 100,
            child: GfFloatingAction(
              onPressed: () {},
              symbol: 'plus',
              bottomInset: 0,
            ),
          ),
          Positioned(
            left: 0,
            right: 0,
            bottom: 0,
            child: GfBottomNavigation(
              currentIndex: 0,
              showLabels: true,
              onSelected: (_) {},
              items: const [
                GfBottomNavigationItem(
                  label: '首页',
                  symbol: 'house',
                  selectedSymbol: 'house-filled',
                ),
                GfBottomNavigationItem(
                  label: '校园',
                  symbol: 'graduation-cap',
                  selectedSymbol: 'graduation-cap-filled',
                ),
                GfBottomNavigationItem(
                  label: '通知',
                  symbol: 'bell',
                  selectedSymbol: 'bell-filled',
                  badge: true,
                ),
                GfBottomNavigationItem(
                  label: '私信',
                  symbol: 'mail',
                  selectedSymbol: 'mail-filled',
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// 通知
// ---------------------------------------------------------------------------

class NotificationsScreen extends StatefulWidget {
  const NotificationsScreen({super.key});

  @override
  State<NotificationsScreen> createState() => _NotificationsScreenState();
}

class _NotificationsScreenState extends State<NotificationsScreen> {
  Object _tab = 'all';

  @override
  Widget build(BuildContext context) {
    final c = GfTheme.colorsOf(context);
    return Scaffold(
      appBar: GfAppBar(
        title: const GfText('通知'),
        showDivider: false,
        actions: [
          GfIconButton(
            symbol: 'check-check',
            tooltip: '全部已读',
            color: c.textPrimary,
            onPressed: () {},
          ),
          const SizedBox(width: 4),
        ],
      ),
      body: ListView(
        children: [
          DecoratedBox(
            decoration: BoxDecoration(
              border: Border(
                bottom: BorderSide(
                  color: c.separator,
                  width: gfHairline(context),
                ),
              ),
            ),
            child: GfTabBar(
              tabs: const [
                GfTab(label: '全部', value: 'all'),
                GfTab(label: '未读', value: 'unread'),
                GfTab(label: '@我', value: 'mention'),
              ],
              selected: _tab,
              distribute: true,
              onSelected: (v) => setState(() => _tab = v),
            ),
          ),
          GfNotificationRow(
            symbol: 'heart-filled',
            tone: GfNotificationTone.like,
            title: '林知夏 赞了你的帖子',
            actorName: '林知夏',
            avatarUrl: avatar(47),
            subtitle: '整理了一份嘉定校区骑行路线',
            time: '3 分钟前',
            unread: true,
            onTap: () {},
          ),
          GfNotificationRow(
            symbol: 'message-circle-filled',
            tone: GfNotificationTone.primary,
            title: '周屿 回复了你',
            actorName: '周屿',
            avatarUrl: avatar(12),
            subtitle: '人工湖那段晚上灯光很好看，推荐反方向骑。',
            time: '28 分钟前',
            unread: true,
            onTap: () {},
          ),
          GfNotificationRow(
            symbol: 'at-sign',
            tone: GfNotificationTone.info,
            title: '陈一鸣 在帖子中提到了你',
            actorName: '陈一鸣',
            avatarUrl: avatar(32),
            subtitle: '@小济同学 你上次说的那家咖啡店叫什么来着？',
            time: '2 小时前',
            unread: false,
            onTap: () {},
          ),
          GfNotificationRow(
            symbol: 'user-round-plus',
            tone: GfNotificationTone.primary,
            title: '苏念 关注了你',
            actorName: '苏念',
            avatarUrl: avatar(25),
            subtitle: '',
            time: '昨天',
            unread: false,
            onTap: () {},
          ),
          GfNotificationRow(
            symbol: 'shield-check',
            tone: GfNotificationTone.success,
            title: '你的课程评价已通过审核',
            subtitle: '工科数学分析 A(I) · 赵恺',
            time: '昨天',
            unread: false,
            onTap: () {},
          ),
          GfNotificationRow(
            symbol: 'award',
            tone: GfNotificationTone.warning,
            title: '获得新徽章「早起的鸟儿」',
            subtitle: '连续 7 天在 8 点前签到',
            time: '周一',
            unread: false,
            onTap: () {},
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// 私信对话
// ---------------------------------------------------------------------------

class ChatScreen extends StatelessWidget {
  const ChatScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final c = GfTheme.colorsOf(context);
    return Scaffold(
      appBar: GfAppBar(
        title: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            GfAvatar(src: avatar(32), size: 28),
            const SizedBox(width: 8),
            const GfText('陈一鸣'),
          ],
        ),
        actions: [
          GfIconButton(
            symbol: 'ellipsis',
            tooltip: '更多',
            color: c.textPrimary,
            onPressed: () {},
          ),
          const SizedBox(width: 4),
        ],
      ),
      body: Column(
        children: [
          Expanded(
            child: ListView(
              padding: const EdgeInsets.symmetric(horizontal: 16),
              children: const [
                _DaySep('昨天 21:40'),
                GfMessageBubble(text: '下周的结构力学小组作业你分到哪部分了？', mine: false),
                SizedBox(height: 6),
                GfMessageBubble(text: '我负责第三章的受力分析，图还没画完 😵', mine: true),
                SizedBox(height: 6),
                GfMessageBubble(text: '那我先把计算部分整理出来，你画完了发我一下', mine: false),
                _DaySep('今天 14:05'),
                GfMessageBubble(text: '明天下午的小组讨论改到图书馆三楼啦', mine: false),
                SizedBox(height: 6),
                GfMessageBubble(text: '收到！我会带上打印好的图纸', mine: true),
                SizedBox(height: 6),
                GfMessageBubble(
                  text: '顺便帮你占个靠窗的位置 ☀️',
                  mine: true,
                  time: '14:08 已读',
                ),
              ],
            ),
          ),
          GfChatInput(
            onSend: (_) {},
            hintText: '发消息…',
            onAttach: () {},
            attachLabel: '添加图片',
          ),
        ],
      ),
    );
  }
}

class _DaySep extends StatelessWidget {
  const _DaySep(this.label);
  final String label;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(vertical: 14),
    child: Center(child: Caption(label)),
  );
}

// ---------------------------------------------------------------------------
// 登录
// ---------------------------------------------------------------------------

class LoginScreen extends StatefulWidget {
  const LoginScreen({super.key});

  @override
  State<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen> {
  String _mode = 'password';

  @override
  Widget build(BuildContext context) {
    final c = GfTheme.colorsOf(context);
    final t = GfTheme.typographyOf(context);
    return Scaffold(
      appBar: GfAppBar(
        title: const SizedBox.shrink(),
        showDivider: false,
        actions: [
          GfNavTextAction(label: '随便看看', onPressed: () {}),
          const SizedBox(width: 4),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(24, 8, 24, 32),
        children: [
          const Align(alignment: Alignment.centerLeft, child: GfLogo(size: 52)),
          const SizedBox(height: 20),
          GfText('欢迎回到 YourTJ', style: t.display),
          const SizedBox(height: 6),
          GfText(
            '同济人的校园社区，课程评价、排课和讨论都在这里。',
            style: t.small.copyWith(color: c.textSecondary),
          ),
          const SizedBox(height: 28),
          GfSegmented<String>(
            segments: const [('账号密码', 'password'), ('同济统一身份', 'tongji')],
            selected: _mode,
            onSelected: (v) => setState(() => _mode = v),
          ),
          const SizedBox(height: 20),
          const GfInput(
            labelText: '用户名或邮箱',
            prefixIcon: Padding(
              padding: EdgeInsets.all(14),
              child: GfSymbol('user-round', size: 20),
            ),
          ),
          const SizedBox(height: 12),
          const GfInput(
            labelText: '密码',
            obscureText: true,
            prefixIcon: Padding(
              padding: EdgeInsets.all(14),
              child: GfSymbol('lock-filled', size: 20),
            ),
            suffixIcon: Padding(
              padding: EdgeInsets.all(14),
              child: GfSymbol('eye-off', size: 20),
            ),
          ),
          Align(
            alignment: Alignment.centerRight,
            child: GfButton(
              label: '忘记密码？',
              variant: GfButtonVariant.link,
              size: GfButtonSize.small,
              onPressed: () {},
            ),
          ),
          const SizedBox(height: 12),
          GfButton(
            label: '登录',
            size: GfButtonSize.extraLarge,
            expanded: true,
            onPressed: () {},
          ),
          const SizedBox(height: 12),
          GfButton(
            label: '注册新账号',
            size: GfButtonSize.extraLarge,
            variant: GfButtonVariant.secondary,
            expanded: true,
            onPressed: () {},
          ),
          const SizedBox(height: 28),
          Row(
            children: [
              Expanded(
                child: Divider(
                  color: c.separator,
                  thickness: gfHairline(context),
                ),
              ),
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 12),
                child: Caption('其他方式'),
              ),
              Expanded(
                child: Divider(
                  color: c.separator,
                  thickness: gfHairline(context),
                ),
              ),
            ],
          ),
          const SizedBox(height: 16),
          Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              GfIconButton(
                symbol: 'github',
                tooltip: 'GitHub',
                variant: GfIconButtonVariant.tonal,
                size: 52,
                iconSize: 22,
                color: c.textPrimary,
                onPressed: () {},
              ),
              const SizedBox(width: 16),
              GfIconButton(
                symbol: 'key-round',
                tooltip: '通行密钥',
                variant: GfIconButtonVariant.tonal,
                size: 52,
                iconSize: 22,
                color: c.textPrimary,
                onPressed: () {},
              ),
            ],
          ),
          const SizedBox(height: 24),
          GfText(
            '登录即表示同意《用户协议》与《隐私政策》',
            textAlign: TextAlign.center,
            style: t.caption.copyWith(color: c.textSecondary),
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Overlay demos: real routes opened over a sample page.
// ---------------------------------------------------------------------------

class DialogDemoScreen extends StatefulWidget {
  const DialogDemoScreen({super.key});

  @override
  State<DialogDemoScreen> createState() => _DialogDemoScreenState();
}

class _DialogDemoScreenState extends State<DialogDemoScreen> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) showConfirmDemo(context);
    });
  }

  @override
  Widget build(BuildContext context) => const ProfileEditScreen();
}

class SheetDemoScreen extends StatefulWidget {
  const SheetDemoScreen({super.key});

  @override
  State<SheetDemoScreen> createState() => _SheetDemoScreenState();
}

class _SheetDemoScreenState extends State<SheetDemoScreen> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) showSheetDemo(context);
    });
  }

  @override
  Widget build(BuildContext context) => const HomeScreen();
}
