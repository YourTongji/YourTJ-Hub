/// @mention 候选面板（issue #565）。
///
/// 位于回复 composer 上方、软键盘之上（宿主 Positioned + SafeArea + Scaffold
/// resize 区域内），遵循 maxWidth: 560；列表约 4.5 行封顶可滚动；毛玻璃表面
/// 与滚动隔离（候选滚动不带动底层话题列表）；Semantics 提供 listbox/option
/// 语义与身份标签；active 行不只依赖颜色（加粗 + 高亮底色）。选中经
/// [onSelect] 由宿主写入编辑器，面板不自持业务状态。
library;

import 'dart:ui';

import 'package:flutter/material.dart';

import 'package:ui_kit/ui_kit.dart';

import 'mention_session.dart';

/// 候选行固定高度（≥48dp 触控目标，键盘滚动按此步进）。
const double kMentionRowHeight = 52;

/// 面板文案（宿主从 AppLocalizations 注入，组件保持 l10n 无关）。
@immutable
class MentionPanelMessages {
  const MentionPanelMessages({
    required this.listboxLabel,
    required this.loading,
    required this.noResults,
    required this.searchFailed,
    required this.keepTyping,
    required this.tagReplyTarget,
    required this.tagTopicAuthor,
    required this.tagParticipant,
  });

  final String listboxLabel;
  final String loading;
  final String noResults;
  final String searchFailed;
  final String keepTyping;
  final String tagReplyTarget;
  final String tagTopicAuthor;
  final String tagParticipant;

  String tagFor(MentionTag? tag) => switch (tag) {
    MentionTag.replyTarget => tagReplyTarget,
    MentionTag.topicAuthor => tagTopicAuthor,
    MentionTag.participant => tagParticipant,
    null => '',
  };
}

/// 候选面板：跟随会话开合，选中候选交还宿主写入。
class MentionCandidatesPanel extends StatelessWidget {
  const MentionCandidatesPanel({
    super.key,
    required this.session,
    required this.messages,
    required this.onSelect,
  });

  final MentionSessionController session;
  final MentionPanelMessages messages;
  final void Function(MentionToken token, MentionUser user) onSelect;

  /// 候选为空时的提示：仅输入 @ 无本地上下文 → 继续输入（Web 同语义）；
  /// 有 query → 失败提示 / 搜索中 / 无匹配。
  String _hintMessage(bool queryEmpty) {
    if (queryEmpty) return messages.keepTyping;
    if (session.failed) return messages.searchFailed;
    if (session.loading) return messages.loading;
    return messages.noResults;
  }

  @override
  Widget build(BuildContext context) {
    return ListenableBuilder(
      listenable: session,
      builder: (context, _) {
        if (!session.open) return const SizedBox.shrink();
        final token = session.token;
        if (token == null) return const SizedBox.shrink();
        final colors = GfTheme.colorsOf(context);
        final radii = GfTheme.radiiOf(context);
        final candidates = session.candidates;
        final queryEmpty = token.query.trim().isEmpty;

        return TextFieldTapRegion(
          child: Padding(
            padding: const EdgeInsets.only(bottom: 8),
            child: ClipRRect(
              borderRadius: BorderRadius.vertical(
                top: Radius.circular(radii.field),
              ),
              child: BackdropFilter(
                filter: ImageFilter.blur(sigmaX: 18, sigmaY: 18),
                child: Container(
                  constraints: const BoxConstraints(maxHeight: 236),
                  decoration: BoxDecoration(
                    color: colors.base100.withValues(alpha: 0.92),
                    border: Border(
                      top: BorderSide(
                        color: colors.line.withValues(alpha: 0.65),
                      ),
                    ),
                  ),
                  // ≈4.5 行封顶（52 * 4.5 + 上下留白），超出滚动。
                  child: Semantics(
                    label: messages.listboxLabel,
                    child: candidates.isEmpty
                        ? _MentionHintRow(message: _hintMessage(queryEmpty))
                        : _MentionCandidateList(
                            session: session,
                            messages: messages,
                            onSelect: onSelect,
                          ),
                  ),
                ),
              ),
            ),
          ),
        );
      },
    );
  }
}

class _MentionCandidateList extends StatefulWidget {
  const _MentionCandidateList({
    required this.session,
    required this.messages,
    required this.onSelect,
  });

  final MentionSessionController session;
  final MentionPanelMessages messages;
  final void Function(MentionToken token, MentionUser user) onSelect;

  @override
  State<_MentionCandidateList> createState() => _MentionCandidateListState();
}

class _MentionCandidateListState extends State<_MentionCandidateList> {
  final ScrollController _scroll = ScrollController();

  @override
  void didUpdateWidget(covariant _MentionCandidateList oldWidget) {
    super.didUpdateWidget(oldWidget);
    // 键盘移动 active 行时保证可见（固定行高步进）。
    if (!_scroll.hasClients) return;
    final double target = widget.session.activeIndex * kMentionRowHeight;
    final double offset = _scroll.offset;
    final double viewport = _scroll.position.viewportDimension;
    if (target < offset) {
      _scroll.jumpTo(target);
    } else if (target + kMentionRowHeight > offset + viewport) {
      _scroll.jumpTo(target + kMentionRowHeight - viewport);
    }
  }

  @override
  void dispose() {
    _scroll.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final session = widget.session;
    return ListView.builder(
      controller: _scroll,
      padding: const EdgeInsets.symmetric(vertical: 4),
      itemCount: session.candidates.length,
      itemBuilder: (context, index) {
        final user = session.candidates[index];
        return _MentionCandidateRow(
          user: user,
          active: index == session.activeIndex,
          tagLabel: widget.messages.tagFor(user.tag),
          onTap: () {
            final token = session.token;
            if (token != null) widget.onSelect(token, user);
          },
        );
      },
    );
  }
}

class _MentionCandidateRow extends StatelessWidget {
  const _MentionCandidateRow({
    required this.user,
    required this.active,
    required this.tagLabel,
    required this.onTap,
  });

  final MentionUser user;
  final bool active;
  final String tagLabel;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    // Semantics 至少包含昵称、@username 与上下文身份。
    final semanticsLabel = tagLabel.isEmpty
        ? '${user.displayName}，@${user.username}'
        : '${user.displayName}，@${user.username}，$tagLabel';
    return Semantics(
      selected: active,
      label: semanticsLabel,
      // 读屏激活：ExcludeSemantics 会剥掉 InkWell 的 tap 语义，动作挂在外层节点。
      onTap: onTap,
      child: ExcludeSemantics(
        child: InkWell(
          onTap: onTap,
          child: Container(
            height: kMentionRowHeight,
            padding: const EdgeInsets.symmetric(horizontal: 14),
            color: active
                ? colors.primary.withValues(alpha: 0.10)
                : Colors.transparent,
            child: Row(
              children: [
                GfAvatar(src: user.avatarUrl, size: 30),
                const SizedBox(width: 10),
                Expanded(
                  child: Text(
                    user.displayName,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: GfTheme.typographyOf(context).body.copyWith(
                      fontWeight: active ? FontWeight.w600 : FontWeight.w400,
                    ),
                  ),
                ),
                Flexible(
                  child: Text(
                    '@${user.username}',
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    textAlign: TextAlign.right,
                    style: GfTheme.typographyOf(
                      context,
                    ).caption.copyWith(color: colors.iconMuted),
                  ),
                ),
                if (tagLabel.isNotEmpty) ...[
                  const SizedBox(width: 8),
                  Flexible(
                    child: Container(
                      padding: const EdgeInsets.symmetric(
                        horizontal: 8,
                        vertical: 2,
                      ),
                      decoration: BoxDecoration(
                        color: colors.base200.withValues(alpha: 0.7),
                        borderRadius: BorderRadius.circular(999),
                      ),
                      child: Text(
                        tagLabel,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: GfTheme.typographyOf(
                          context,
                        ).caption.copyWith(color: colors.iconMuted),
                      ),
                    ),
                  ),
                ],
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _MentionHintRow extends StatelessWidget {
  const _MentionHintRow({required this.message});

  final String message;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    return Semantics(
      label: message,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
        child: Text(
          message,
          maxLines: 2,
          overflow: TextOverflow.ellipsis,
          style: GfTheme.typographyOf(
            context,
          ).caption.copyWith(color: colors.iconMuted),
        ),
      ),
    );
  }
}
