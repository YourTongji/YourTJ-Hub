import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';

import 'mock.dart';

// ---------------------------------------------------------------------------
// Foundations
// ---------------------------------------------------------------------------

class FoundationsPage extends StatelessWidget {
  const FoundationsPage({super.key});

  @override
  Widget build(BuildContext context) {
    final c = GfTheme.colorsOf(context);
    final t = GfTheme.typographyOf(context);
    Widget swatch(
      String name,
      String role,
      Color color, {
      bool outline = false,
    }) => SizedBox(
      width: (MediaQuery.sizeOf(context).width - 32 - 24) / 3,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            height: 44,
            decoration: BoxDecoration(
              color: color,
              borderRadius: BorderRadius.circular(GfRadius.sm),
              border: outline
                  ? Border.all(color: c.separator, width: gfHairline(context))
                  : null,
            ),
          ),
          const SizedBox(height: 6),
          GfText(name, style: t.meta),
          GfText(
            role,
            style: t.label.copyWith(
              color: c.textSecondary,
              fontWeight: FontWeight.w400,
            ),
          ),
        ],
      ),
    );
    final ramp = <(String, String, TextStyle)>[
      ('display', '28/36 · 700', t.display),
      ('title1', '22/30 · 700', t.title1),
      ('title2', '20/28 · 600', t.title2),
      ('title3', '17/24 · 600', t.title3),
      ('heading', '16/24 · 600', t.heading),
      ('body', '16/24 · 400', t.body),
      ('small', '15/22 · 400', t.small),
      ('subhead', '14/20 · 400', t.subhead),
      ('caption', '13/18 · 400', t.caption),
      ('meta', '12/16 · 500', t.meta),
      ('label', '11/14 · 600', t.label),
    ];
    return CatalogScaffold(
      title: '基础',
      children: [
        Specimen(
          title: '颜色角色',
          note: '派生自 Web 色板，不改 tokens',
          child: Wrap(
            spacing: 12,
            runSpacing: 14,
            children: [
              swatch('主色', 'primary', c.primary),
              swatch('主色浅底', 'primarySoft', c.primarySoft),
              swatch('反色', 'neutral', c.neutral),
              swatch('主文字', 'textPrimary', c.textPrimary),
              swatch('次文字', 'textSecondary', c.textSecondary),
              swatch('占位', 'textPlaceholder', c.textPlaceholder),
              swatch('画布', 'canvas', c.canvas, outline: true),
              swatch('分组画布', 'canvasGrouped', c.canvasGrouped, outline: true),
              swatch('分组卡片', 'surfaceGrouped', c.surfaceGrouped, outline: true),
              swatch('填充', 'fill', c.fill, outline: true),
              swatch('强填充', 'fillStrong', c.fillStrong),
              swatch('分隔线', 'separator', c.separator),
              swatch('成功', 'success', c.success),
              swatch('警告', 'warning', c.warning),
              swatch('错误', 'error', c.error),
            ],
          ),
        ),
        Specimen(
          title: '字号阶梯',
          note: '12 个角色，替代 15 种散落字号',
          child: Column(
            children: [
              for (final (name, spec, style) in ramp)
                Padding(
                  padding: const EdgeInsets.symmetric(vertical: 5),
                  child: Row(
                    crossAxisAlignment: CrossAxisAlignment.center,
                    children: [
                      SizedBox(
                        width: 84,
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            GfText(name, style: t.meta),
                            GfText(
                              spec,
                              style: t.label.copyWith(
                                color: c.textSecondary,
                                fontWeight: FontWeight.w400,
                              ),
                            ),
                          ],
                        ),
                      ),
                      Expanded(
                        child: GfText(
                          '同济论坛 YourTJ',
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: style,
                        ),
                      ),
                    ],
                  ),
                ),
            ],
          ),
        ),
        Specimen(
          title: '圆角',
          note: '可交互=胶囊，静态标签=小方角',
          child: Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              for (final (name, r) in [
                ('xs 6', GfRadius.xs),
                ('sm 10', GfRadius.sm),
                ('md 14', GfRadius.md),
                ('lg 20', GfRadius.lg),
                ('xl 28', GfRadius.xl),
                ('full', GfRadius.full),
              ])
                Column(
                  children: [
                    Container(
                      width: 50,
                      height: 50,
                      decoration: BoxDecoration(
                        color: c.primarySoft,
                        border: Border.all(
                          color: c.primary.withValues(alpha: 0.5),
                          width: 1,
                        ),
                        borderRadius: BorderRadius.circular(r),
                      ),
                    ),
                    const SizedBox(height: 6),
                    Caption(name),
                  ],
                ),
            ],
          ),
        ),
        Specimen(
          title: '间距',
          note: '4pt 栅格；页边距 16，分组间距 24',
          child: Column(
            children: [
              for (final s in [
                GfSpace.xs,
                GfSpace.sm,
                GfSpace.md,
                GfSpace.lg,
                GfSpace.xl,
                GfSpace.xxl,
                GfSpace.xxxl,
              ])
                Padding(
                  padding: const EdgeInsets.symmetric(vertical: 3),
                  child: Row(
                    children: [
                      SizedBox(
                        width: 36,
                        child: Caption('${s.toInt()}', align: TextAlign.start),
                      ),
                      Container(
                        width: s * 6,
                        height: 10,
                        decoration: BoxDecoration(
                          color: c.primary.withValues(alpha: 0.75),
                          borderRadius: BorderRadius.circular(3),
                        ),
                      ),
                    ],
                  ),
                ),
            ],
          ),
        ),
        Specimen(
          title: '层级',
          note: '深色模式改用浅色面 + 发丝线',
          padded: false,
          child: GroupedBand(
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: GfSpace.gutter),
              child: Row(
                children: [
                  for (final (name, level) in [
                    ('low', GfElevation.low),
                    ('mid', GfElevation.mid),
                    ('high', GfElevation.high),
                  ]) ...[
                    if (name != 'low') const SizedBox(width: 12),
                    Expanded(
                      child: Container(
                        height: 72,
                        alignment: Alignment.center,
                        decoration: BoxDecoration(
                          color: c.surfaceRaised,
                          borderRadius: BorderRadius.circular(GfRadius.lg),
                          boxShadow: c.elevation(level),
                          border: c.isDark
                              ? Border.all(color: c.separator, width: 0.5)
                              : null,
                        ),
                        child: GfText(
                          name,
                          style: t.subhead.copyWith(color: c.textSecondary),
                        ),
                      ),
                    ),
                  ],
                ],
              ),
            ),
          ),
        ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Buttons
// ---------------------------------------------------------------------------

class ButtonsPage extends StatelessWidget {
  const ButtonsPage({super.key});

  @override
  Widget build(BuildContext context) {
    final c = GfTheme.colorsOf(context);
    void tap() {}
    return CatalogScaffold(
      title: '按钮与操作',
      children: [
        Specimen(
          title: '变体',
          note: '同一胶囊形状，只换颜色',
          child: Wrap(
            spacing: 10,
            runSpacing: 12,
            children: [
              GfButton(
                label: '发布',
                icon: const GfSymbol('square-pen'),
                onPressed: tap,
              ),
              GfButton(
                label: '关注',
                variant: GfButtonVariant.tonal,
                icon: const GfSymbol('user-round-plus'),
                onPressed: tap,
              ),
              GfButton(
                label: '收藏',
                variant: GfButtonVariant.secondary,
                icon: const GfSymbol('bookmark'),
                onPressed: tap,
              ),
              GfButton(
                label: '分享',
                variant: GfButtonVariant.outline,
                icon: const GfSymbol('share-2'),
                onPressed: tap,
              ),
              GfButton(
                label: '反色',
                variant: GfButtonVariant.neutral,
                onPressed: tap,
              ),
              GfButton(
                label: '删除',
                variant: GfButtonVariant.danger,
                icon: const GfSymbol('trash-2'),
                onPressed: tap,
              ),
              GfButton(
                label: '幽灵',
                variant: GfButtonVariant.ghost,
                onPressed: tap,
              ),
              GfButton(
                label: '弱化',
                variant: GfButtonVariant.muted,
                onPressed: tap,
              ),
              GfButton(
                label: '查看全部',
                variant: GfButtonVariant.link,
                onPressed: tap,
              ),
            ],
          ),
        ),
        Specimen(
          title: '尺寸',
          note: '32 / 40 / 48 / 52（与输入框同高）',
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Wrap(
                spacing: 10,
                crossAxisAlignment: WrapCrossAlignment.center,
                children: [
                  GfButton(
                    label: '小号',
                    size: GfButtonSize.small,
                    onPressed: tap,
                  ),
                  GfButton(label: '中号', onPressed: tap),
                  GfButton(
                    label: '大号',
                    size: GfButtonSize.large,
                    onPressed: tap,
                  ),
                ],
              ),
              const SizedBox(height: 12),
              GfButton(
                label: '登录',
                size: GfButtonSize.extraLarge,
                expanded: true,
                onPressed: tap,
              ),
            ],
          ),
        ),
        Specimen(
          title: '状态',
          child: Wrap(
            spacing: 10,
            runSpacing: 12,
            children: [
              GfButton(label: '保存中', loading: true, onPressed: tap),
              const GfButton(label: '不可用', onPressed: null),
              const GfButton(
                label: '不可用',
                variant: GfButtonVariant.secondary,
                onPressed: null,
              ),
              const GfButton(
                label: '不可用',
                variant: GfButtonVariant.outline,
                onPressed: null,
              ),
            ],
          ),
        ),
        Specimen(
          title: '图标按钮',
          note: '纯图标 · 浅底 · 实底，触控区恒为 44',
          child: Row(
            children: [
              GfIconButton(symbol: 'heart', tooltip: '喜欢', onPressed: tap),
              GfIconButton(symbol: 'bookmark', tooltip: '收藏', onPressed: tap),
              GfIconButton(symbol: 'share-2', tooltip: '分享', onPressed: tap),
              const SizedBox(width: 12),
              GfIconButton(
                symbol: 'x',
                tooltip: '关闭',
                variant: GfIconButtonVariant.tonal,
                iconSize: 18,
                onPressed: tap,
              ),
              GfIconButton(
                symbol: 'ellipsis',
                tooltip: '更多',
                variant: GfIconButtonVariant.tonal,
                onPressed: tap,
              ),
              const SizedBox(width: 12),
              GfIconButton(
                symbol: 'arrow-up',
                tooltip: '发送',
                variant: GfIconButtonVariant.filled,
                onPressed: tap,
              ),
              const GfIconButton(
                symbol: 'arrow-up',
                tooltip: '发送',
                variant: GfIconButtonVariant.filled,
                onPressed: null,
              ),
            ],
          ),
        ),
        Specimen(
          title: '浮动按钮',
          child: SizedBox(
            height: 72,
            child: Row(
              children: [
                SizedBox(
                  width: 72,
                  child: GfFloatingAction(
                    onPressed: tap,
                    symbol: 'plus',
                    bottomInset: 4,
                  ),
                ),
                const SizedBox(width: 16),
                Expanded(
                  child: GfFloatingAction(
                    onPressed: tap,
                    symbol: 'square-pen',
                    label: '写帖子',
                    bottomInset: 4,
                  ),
                ),
              ],
            ),
          ),
        ),
        Specimen(
          title: '导航栏操作',
          note: '取消 / 标题 / 保存',
          padded: false,
          child: Container(
            decoration: BoxDecoration(
              border: Border.symmetric(
                horizontal: BorderSide(
                  color: c.separator,
                  width: gfHairline(context),
                ),
              ),
            ),
            child: SizedBox(
              height: 56,
              child: NavigationToolbar(
                leading: GfNavTextAction(label: '取消', onPressed: tap),
                middle: GfText(
                  '编辑资料',
                  style: GfTheme.typographyOf(context).title3,
                ),
                trailing: Padding(
                  padding: const EdgeInsets.only(right: GfSpace.md),
                  child: GfButton(
                    label: '保存',
                    size: GfButtonSize.small,
                    onPressed: tap,
                  ),
                ),
              ),
            ),
          ),
        ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Inputs
// ---------------------------------------------------------------------------

class InputsPage extends StatefulWidget {
  const InputsPage({super.key});

  @override
  State<InputsPage> createState() => _InputsPageState();
}

class _InputsPageState extends State<InputsPage> {
  String _segment = 'latest';
  String _mode = 'card';
  final Set<String> _tags = {'院系', '只看有评价'};
  bool _notify = true;
  bool _check = true;
  int _radio = 1;
  double _slider = 0.6;
  final _search = TextEditingController(text: '高等数学');
  final _bio = TextEditingController(text: '土木工程 · 2023 级，喜欢摄影和骑行。');

  @override
  void dispose() {
    _search.dispose();
    _bio.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final c = GfTheme.colorsOf(context);
    return CatalogScaffold(
      title: '表单与选择',
      children: [
        Specimen(
          title: '输入框',
          note: '填充底 · 圆角 14 · 高 52',
          child: Column(
            children: [
              const GfInput(
                hintText: '学号或邮箱',
                prefixIcon: Padding(
                  padding: EdgeInsets.all(14),
                  child: GfSymbol('user-round', size: 20),
                ),
              ),
              const SizedBox(height: 12),
              GfInput(
                controller: TextEditingController(text: 'yourtj@tongji.edu.cn'),
                labelText: '邮箱',
              ),
              const SizedBox(height: 12),
              GfInput(
                controller: TextEditingController(text: '12'),
                decoration: const InputDecoration(
                  labelText: '密码',
                  errorText: '密码至少需要 8 位',
                ),
                obscureText: true,
              ),
              const SizedBox(height: 12),
              const GfInput(hintText: '不可编辑', enabled: false),
            ],
          ),
        ),
        Specimen(
          title: '搜索',
          note: '胶囊 · 高 44',
          child: Column(
            children: [
              const GfSearchField(hintText: '搜索帖子、课程、用户', clearLabel: '清除'),
              const SizedBox(height: 12),
              GfSearchField(
                hintText: '搜索',
                clearLabel: '清除',
                controller: _search,
              ),
            ],
          ),
        ),
        Specimen(
          title: '分段选择',
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              GfSegmented<String>(
                segments: const [
                  ('最新', 'latest'),
                  ('热门', 'hot'),
                  ('关注', 'following'),
                ],
                selected: _segment,
                onSelected: (v) => setState(() => _segment = v),
              ),
              const SizedBox(height: 12),
              GfPillSwitch<String>(
                options: const [
                  GfPillOption(
                    label: '卡片',
                    value: 'card',
                    symbol: 'layout-grid',
                  ),
                  GfPillOption(label: '列表', value: 'list', symbol: 'list'),
                ],
                selected: _mode,
                onSelected: (v) => setState(() => _mode = v),
              ),
            ],
          ),
        ),
        Specimen(
          title: '筛选标签',
          note: '选中=主色浅底',
          child: Wrap(
            spacing: 8,
            runSpacing: 4,
            children: [
              for (final tag in ['院系', '学期', '校区', '教师', '只看有评价'])
                GfSelectTag(
                  label: tag,
                  selected: _tags.contains(tag),
                  onChanged: (v) =>
                      setState(() => v ? _tags.add(tag) : _tags.remove(tag)),
                ),
            ],
          ),
        ),
        Specimen(
          title: '开关 · 勾选 · 单选 · 滑块',
          padded: false,
          child: GroupedBand(
            child: GfListSection(
              children:
                  [
                        GfSwitchRow(
                          title: '新回复提醒',
                          value: _notify,
                          onChanged: (v) => setState(() => _notify = v),
                        ),
                        GfSettingRow(
                          title: '同步到日历',
                          trailing: Checkbox(
                            value: _check,
                            onChanged: (v) =>
                                setState(() => _check = v ?? false),
                          ),
                          onTap: () => setState(() => _check = !_check),
                        ),
                        for (final (i, label) in [(0, '跟随系统'), (1, '浅色')])
                          GfSettingRow(
                            title: label,
                            trailing: Radio<int>(value: i),
                            onTap: () => setState(() => _radio = i),
                          ),
                        Padding(
                          padding: const EdgeInsets.fromLTRB(16, 8, 16, 8),
                          child: Row(
                            children: [
                              GfSymbol(
                                'type',
                                size: 16,
                                color: c.textSecondary,
                              ),
                              Expanded(
                                child: Slider(
                                  value: _slider,
                                  onChanged: (v) => setState(() => _slider = v),
                                ),
                              ),
                              GfSymbol(
                                'type',
                                size: 24,
                                color: c.textSecondary,
                              ),
                            ],
                          ),
                        ),
                      ]
                      .map(
                        (w) => RadioGroup<int>(
                          groupValue: _radio,
                          onChanged: (v) => setState(() => _radio = v ?? 0),
                          child: w,
                        ),
                      )
                      .toList(),
            ),
          ),
        ),
        Specimen(
          title: '分组表单行',
          note: '用于编辑资料等页面',
          padded: false,
          child: GroupedBand(
            child: GfListSection(
              children: [
                GfFieldRow(
                  label: '昵称',
                  controller: TextEditingController(text: '小济同学'),
                  maxLength: 20,
                ),
                GfFieldRow(
                  label: '简介',
                  controller: _bio,
                  maxLines: 3,
                  maxLength: 120,
                ),
                const GfFieldRow(
                  label: '个人网站',
                  hintText: 'https://',
                  keyboardType: TextInputType.url,
                ),
              ],
            ),
          ),
        ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Tags & feedback
// ---------------------------------------------------------------------------

class FeedbackPage extends StatefulWidget {
  const FeedbackPage({super.key});

  @override
  State<FeedbackPage> createState() => _FeedbackPageState();
}

class _FeedbackPageState extends State<FeedbackPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) showGfToast(context, '已保存到草稿箱');
    });
  }

  @override
  Widget build(BuildContext context) {
    final c = GfTheme.colorsOf(context);
    final t = GfTheme.typographyOf(context);
    return CatalogScaffold(
      title: '标签与反馈',
      children: [
        Specimen(
          title: '状态徽标',
          note: '静态 · 圆角 6',
          child: const Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              GfBadge(label: '草稿'),
              GfBadge(
                label: '置顶',
                variant: GfBadgeVariant.info,
                icon: GfSymbol('pin-filled'),
              ),
              GfBadge(
                label: '已解决',
                variant: GfBadgeVariant.success,
                icon: GfSymbol('circle-check'),
              ),
              GfBadge(
                label: '待审核',
                variant: GfBadgeVariant.warning,
                icon: GfSymbol('clock'),
              ),
              GfBadge(label: '已屏蔽', variant: GfBadgeVariant.error),
              GfBadge(label: '26 秋'),
              GfBadge(label: '4 学分'),
            ],
          ),
        ),
        Specimen(
          title: '分类与计数',
          note: '分类=胶囊，未读计数=红色',
          child: Row(
            children: [
              const GfChip(label: '校园生活', color: catCampus),
              const SizedBox(width: 8),
              const GfChip(label: '学习交流', color: catStudy),
              const SizedBox(width: 8),
              const GfChip(label: '二手', color: catTrade),
              const Spacer(),
              const GfCountBadge(count: 3),
              const SizedBox(width: 6),
              const GfCountBadge(count: 28),
              const SizedBox(width: 6),
              const GfCountBadge(count: 120),
            ],
          ),
        ),
        Specimen(
          title: '提示条',
          note: '中性 · 主色 · 成功 · 警告 · 错误',
          child: Column(
            children: [
              const GfAlert(symbol: 'info', child: GfText('评价会在审核通过后公开显示。')),
              const SizedBox(height: 8),
              const GfAlert(
                tone: GfTone.primary,
                symbol: 'sparkles',
                child: GfText('新功能：可以把课程加入排课器对比。'),
              ),
              const SizedBox(height: 8),
              const GfAlert(
                tone: GfTone.warning,
                symbol: 'circle-alert',
                child: GfText('邮箱尚未验证，部分功能受限。'),
              ),
              const SizedBox(height: 8),
              const GfStatusMessage(
                message: '资料已更新',
                variant: GfStatusMessageVariant.success,
              ),
              const SizedBox(height: 8),
              const GfStatusMessage(message: '网络连接失败，请稍后重试'),
            ],
          ),
        ),
        Specimen(
          title: '加载与骨架',
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const GfSkeleton(width: 44, height: 44, radius: GfRadius.full),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: const [
                    GfSkeleton(width: 120, height: 14),
                    SizedBox(height: 8),
                    GfSkeleton(height: 14),
                    SizedBox(height: 8),
                    GfSkeleton(width: 200, height: 14),
                  ],
                ),
              ),
              const SizedBox(width: 16),
              const SizedBox(
                width: 28,
                height: 28,
                child: GfProgressIndicator(strokeWidth: 3),
              ),
            ],
          ),
        ),
        Specimen(
          title: '空状态',
          child: Container(
            height: 300,
            decoration: BoxDecoration(
              border: Border.all(
                color: c.separator,
                width: gfHairline(context),
              ),
              borderRadius: BorderRadius.circular(GfRadius.lg),
            ),
            child: GfEmpty(
              symbol: 'bell',
              message: '还没有通知',
              description: '有人回复、点赞或提到你时，会在这里出现。',
              action: GfButton(
                label: '去首页逛逛',
                variant: GfButtonVariant.secondary,
                onPressed: () {},
              ),
            ),
          ),
        ),
        const SizedBox(height: 8),
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16),
          child: GfText(
            '顶部为反色轻提示（Toast）。',
            style: t.caption.copyWith(color: c.textSecondary),
          ),
        ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Lists
// ---------------------------------------------------------------------------

class ListsPage extends StatefulWidget {
  const ListsPage({super.key});

  @override
  State<ListsPage> createState() => _ListsPageState();
}

class _ListsPageState extends State<ListsPage> {
  bool _push = true;

  @override
  Widget build(BuildContext context) {
    final c = GfTheme.colorsOf(context);
    return CatalogScaffold(
      title: '列表与分组',
      children: [
        Specimen(
          title: '分组列表',
          note: '分组画布 + 圆角 20 卡片',
          padded: false,
          child: GroupedBand(
            child: GfListSection(
              header: '通用',
              footer: '分隔线从标题处开始，与图标对齐。',
              dividerIndent: GfListSection.iconRowIndent,
              children: [
                GfSettingRow(
                  symbol: 'languages',
                  title: '语言',
                  value: '简体中文',
                  onTap: () {},
                ),
                GfSettingRow(
                  symbol: 'type',
                  title: '字体大小',
                  value: '标准',
                  onTap: () {},
                ),
                GfSwitchRow(
                  symbol: 'bell',
                  title: '推送通知',
                  description: '回复、私信与系统通知',
                  value: _push,
                  onChanged: (v) => setState(() => _push = v),
                ),
                GfSettingRow(
                  symbol: 'log-out',
                  title: '退出登录',
                  destructive: true,
                  onTap: () {},
                ),
              ],
            ),
          ),
        ),
        Specimen(
          title: '通知行',
          note: '头像 + 类型角标；无头像时用色调圆',
          padded: false,
          child: Column(
            children: [
              GfNotificationRow(
                symbol: 'heart-filled',
                tone: GfNotificationTone.like,
                title: '林知夏 赞了你的帖子',
                actorName: '林知夏',
                avatarUrl: avatar(47),
                subtitle: '四平路校区哪个食堂的早餐最值得推荐？',
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
                subtitle: '西苑的豆浆油条，七点前去不用排队。',
                time: '1 小时前',
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
            ],
          ),
        ),
        Specimen(
          title: '会话行',
          padded: false,
          child: Column(
            children: [
              GfConversationRow(
                avatarUrl: avatar(32),
                name: '陈一鸣',
                lastMessage: '明天下午的小组讨论改到图书馆三楼啦',
                time: '14:08',
                unreadCount: 2,
                unreadLabel: '未读',
                onTap: () {},
              ),
              Divider(
                indent: 76,
                height: 1,
                thickness: gfHairline(context),
                color: c.separator,
              ),
              GfConversationRow(
                avatarUrl: avatar(5),
                name: '学生会 · 学术部',
                lastMessage: '[图片] 讲座海报终稿',
                time: '昨天',
                unreadCount: 0,
                unreadLabel: '未读',
                onTap: () {},
              ),
            ],
          ),
        ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Overlays
// ---------------------------------------------------------------------------

class OverlaysPage extends StatelessWidget {
  const OverlaysPage({super.key});

  @override
  Widget build(BuildContext context) {
    final c = GfTheme.colorsOf(context);
    return CatalogScaffold(
      title: '浮层',
      children: [
        Specimen(
          title: '对话框',
          note: '圆角 28 · 双按钮等分',
          padded: false,
          child: Container(
            color: c.scrim,
            padding: const EdgeInsets.symmetric(horizontal: 32, vertical: 28),
            child: Material(
              color: c.surfaceRaised,
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(GfRadius.xl),
              ),
              child: GfAlertDialog(
                title: const GfText('删除这条评论？'),
                content: const GfText('删除后无法恢复，相关的点赞与回复也会一起移除。'),
                actions: [
                  GfDialogAction(onPressed: () {}, child: const GfText('取消')),
                  GfDialogAction(
                    onPressed: () {},
                    filled: true,
                    destructive: true,
                    child: const GfText('删除'),
                  ),
                ],
              ),
            ),
          ),
        ),
        Specimen(
          title: '菜单',
          note: '圆角 20 · 项圆角 14（同心）',
          child: Align(
            alignment: Alignment.centerRight,
            child: SizedBox(
              width: 220,
              child: GfMenuSurface(
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    GfMenuItem(label: '分享', symbol: 'share-2', onTap: () {}),
                    GfMenuItem(label: '复制链接', symbol: 'link', onTap: () {}),
                    GfMenuItem(label: '举报', symbol: 'flag', onTap: () {}),
                    GfMenuItem(
                      label: '删除',
                      symbol: 'trash-2',
                      variant: GfMenuItemVariant.danger,
                      onTap: () {},
                    ),
                  ],
                ),
              ),
            ),
          ),
        ),
        Specimen(
          title: '底部面板',
          note: '顶部圆角 28 · 统一标题行',
          padded: false,
          child: Container(
            color: c.scrim,
            padding: const EdgeInsets.only(top: 24),
            child: Material(
              color: c.surfaceRaised,
              shape: const RoundedRectangleBorder(
                borderRadius: BorderRadius.vertical(
                  top: Radius.circular(GfRadius.xl),
                ),
              ),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  SizedBox(
                    height: 28,
                    child: Center(
                      child: Container(
                        width: 36,
                        height: 5,
                        decoration: BoxDecoration(
                          color: c.fillStrong,
                          borderRadius: BorderRadius.circular(GfRadius.full),
                        ),
                      ),
                    ),
                  ),
                  GfSheetHeader(
                    title: '选择学期',
                    description: '课程目录将按学期筛选',
                    onClose: () {},
                  ),
                  for (final (term, selected) in [
                    ('2026 秋季', true),
                    ('2026 春季', false),
                    ('2025 秋季', false),
                  ])
                    GfSettingRow(
                      title: term,
                      trailing: selected
                          ? GfSymbol('check', size: 20, color: c.primary)
                          : const SizedBox(width: 20),
                      onTap: () {},
                    ),
                  const SizedBox(height: 12),
                ],
              ),
            ),
          ),
        ),
        Specimen(
          title: '真实弹出',
          child: Wrap(
            spacing: 10,
            runSpacing: 10,
            children: [
              GfButton(
                label: '对话框',
                variant: GfButtonVariant.secondary,
                onPressed: () => showConfirmDemo(context),
              ),
              GfButton(
                label: '底部面板',
                variant: GfButtonVariant.secondary,
                onPressed: () => showSheetDemo(context),
              ),
              GfButton(
                label: '操作表',
                variant: GfButtonVariant.secondary,
                onPressed: () => showGfActionSheet<String>(
                  context,
                  title: const GfText('排序方式'),
                  initialValue: 'latest',
                  items: const [
                    GfPopupMenuItem(value: 'latest', child: GfText('最新发布')),
                    GfPopupMenuItem(value: 'reply', child: GfText('最新回复')),
                    GfPopupMenuItem(value: 'hot', child: GfText('最热')),
                  ],
                ),
              ),
              GfButton(
                label: '轻提示',
                variant: GfButtonVariant.secondary,
                onPressed: () => showGfToast(context, '链接已复制'),
              ),
            ],
          ),
        ),
      ],
    );
  }
}

Future<void> showConfirmDemo(BuildContext context) => showGfAlertDialog<void>(
  context,
  barrierDismissible: true,
  builder: (context) => GfAlertDialog(
    title: const GfText('放弃这次编辑？'),
    content: const GfText('你修改的昵称和简介还没有保存，离开后将丢失。'),
    actions: [
      GfDialogAction(
        onPressed: () => Navigator.pop(context),
        child: const GfText('继续编辑'),
      ),
      GfDialogAction(
        onPressed: () => Navigator.pop(context),
        filled: true,
        destructive: true,
        child: const GfText('放弃'),
      ),
    ],
  ),
);

Future<void> showSheetDemo(BuildContext context) => showGfBottomSheet<void>(
  context,
  builder: (context) {
    final c = GfTheme.colorsOf(context);
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        GfSheetHeader(
          title: '发布到',
          description: '选择一个分区，让更多同学看到',
          onClose: () => Navigator.pop(context),
        ),
        for (final (name, color, desc, selected) in [
          ('校园生活', catCampus, '吃喝玩乐、生活经验', true),
          ('学习交流', catStudy, '课程、考试与科研', false),
          ('二手交易', catTrade, '闲置转让与求购', false),
          ('互助问答', catHelp, '遇到问题来这里问', false),
        ])
          GfSettingRow(
            leading: Container(
              width: 36,
              height: 36,
              decoration: BoxDecoration(
                color: color.withValues(alpha: c.isDark ? 0.2 : 0.1),
                borderRadius: BorderRadius.circular(GfRadius.sm),
              ),
              alignment: Alignment.center,
              child: Container(
                width: 10,
                height: 10,
                decoration: BoxDecoration(color: color, shape: BoxShape.circle),
              ),
            ),
            title: name,
            description: desc,
            trailing: selected
                ? GfSymbol('check', size: 20, color: c.primary)
                : const SizedBox(width: 20),
            onTap: () => Navigator.pop(context),
          ),
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 12, 16, 8),
          child: GfButton(
            label: '下一步',
            size: GfButtonSize.extraLarge,
            expanded: true,
            onPressed: () => Navigator.pop(context),
          ),
        ),
      ],
    );
  },
);

// ---------------------------------------------------------------------------
// Content
// ---------------------------------------------------------------------------

class ContentPage extends StatelessWidget {
  const ContentPage({super.key});

  @override
  Widget build(BuildContext context) {
    return CatalogScaffold(
      title: '内容',
      children: [
        Specimen(
          title: '帖子卡片',
          padded: false,
          child: Column(
            children: [
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
                onLike: (target) async => true,
                onBookmark: (target) async => true,
                hot: true,
              ),
              GfTopicCard(
                title: '嘉定校区秋天的银杏大道',
                description: '下课路过随手拍的，今年黄得特别早。',
                authorName: '周屿',
                authorAvatarUrl: avatar(12),
                categories: const [
                  GfTopicCategory(name: '摄影', color: catStudy),
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
        ),
        Specimen(
          title: '消息气泡',
          note: '圆角 20，靠近发送方一角收成 6',
          child: Column(
            children: const [
              GfMessageBubble(text: '晚上一起去图书馆吗？', mine: false, time: '19:02'),
              SizedBox(height: 8),
              GfMessageBubble(
                text: '好呀，我先去占座，三楼靠窗那排 📚',
                mine: true,
                time: '19:03',
              ),
            ],
          ),
        ),
      ],
    );
  }
}
