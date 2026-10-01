import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:share_plus/share_plus.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';
import '../../providers.dart';
import '../../server_messages.dart';
import 'post_history_sheet.dart';

class TopicActions extends ConsumerStatefulWidget {
  const TopicActions({
    super.key,
    required this.props,
    required this.onChanged,
    this.firstPostId,
    this.onReport,
  });
  final TopicDetailProps props;
  final int? firstPostId;
  final VoidCallback? onReport;
  final Future<void> Function() onChanged;
  @override
  ConsumerState<TopicActions> createState() => _TopicActionsState();
}

class _TopicActionsState extends ConsumerState<TopicActions> {
  bool _busy = false;
  bool _menuOpen = false;
  final _moreKey = GlobalKey();
  Future<void> _action(String action) async {
    if (_busy) return;
    final l10n = AppLocalizations.of(context);
    final epoch = ref.read(offlineCacheEpochProvider);
    final topic = widget.props.topic;
    if (action == 'report') {
      widget.onReport?.call();
      return;
    }
    if (action == 'edit') {
      await context.push('/publish?id=${topic.id}');
      if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
        await widget.onChanged();
      }
      return;
    }
    if (action == 'history') {
      await showGfBottomSheet<void>(
        context,
        builder: (_) => PostHistorySheet(postId: widget.firstPostId!),
      );
      return;
    }
    if (action == 'share') {
      final url = Uri.parse(
        ref.read(apiClientProvider).baseUrl,
      ).replace(path: '/p/post/${topic.id}', query: null, fragment: null);
      final box = context.findRenderObject() as RenderBox?;
      setState(() => _busy = true);
      try {
        await SharePlus.instance.share(
          ShareParams(
            text: url.toString(),
            sharePositionOrigin: box == null
                ? null
                : box.localToGlobal(Offset.zero) & box.size,
          ),
        );
      } catch (error) {
        if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
          showGfToast(context, resolveErrorMessage(l10n, error), error: true);
        }
      } finally {
        if (mounted) setState(() => _busy = false);
      }
      return;
    }
    final confirmed = await showDialog<bool>(
      context: context,
      animationStyle: GfMotion.dialogStyle(context),
      builder: (context) => AlertDialog(
        title: Text(
          action == 'delete'
              ? l10n.contentDelete
              : action == 'ban'
              ? l10n.topicModerateBan
              : l10n.topicModerateUnban,
        ),
        content: Text(
          action == 'delete'
              ? l10n.topicDeleteConfirm
              : l10n.topicModerateConfirm,
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: Text(l10n.commonCancel),
          ),
          TextButton(
            onPressed: () => Navigator.pop(context, true),
            child: Text(l10n.commonConfirm),
          ),
        ],
      ),
    );
    if (!mounted ||
        confirmed != true ||
        epoch != ref.read(offlineCacheEpochProvider)) {
      return;
    }
    setState(() => _busy = true);
    try {
      final repo = ref.read(topicRepositoryProvider);
      if (action == 'delete') {
        await repo.deleteTopic(topicId: topic.id);
      } else {
        await repo.moderate(topicId: topic.id, ban: action == 'ban');
      }
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      if (action == 'delete') {
        context.go('/');
      } else {
        await widget.onChanged();
      }
    } catch (error) {
      if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
        showGfToast(context, resolveErrorMessage(l10n, error), error: true);
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  List<GfContextAction<String>> _menuActions(AppLocalizations l10n) {
    final props = widget.props;
    final available =
        !props.topic.authorDeleted && !props.topic.moderatorRemoved;
    return [
      if (props.permissions.isOwnTopic && available) ...[
        GfContextAction(
          value: 'edit',
          label: l10n.commonEdit,
          symbol: 'pen-line',
        ),
        GfContextAction(
          value: 'delete',
          label: l10n.contentDelete,
          symbol: 'trash-2',
          destructive: true,
        ),
      ],
      GfContextAction(
        value: 'share',
        label: l10n.topicShare,
        symbol: 'share-2',
      ),
      if (widget.firstPostId != null)
        GfContextAction(
          value: 'history',
          label: l10n.topicHistory,
          symbol: 'clock',
        ),
      if (widget.onReport != null && available && !props.permissions.isOwnTopic)
        GfContextAction(
          value: 'report',
          label: l10n.topicReport,
          symbol: 'flag',
        ),
      if (props.permissions.canModerateTopic && props.topic.processStatus == 0)
        GfContextAction(
          value: 'ban',
          label: l10n.topicModerateBan,
          symbol: 'eye-off',
          destructive: true,
        ),
      if (props.permissions.canModerateTopic && props.topic.processStatus == 1)
        GfContextAction(
          value: 'unban',
          label: l10n.topicModerateUnban,
          symbol: 'eye',
        ),
    ];
  }

  Future<void> _showMore() async {
    if (_busy || _menuOpen) return;
    final epoch = ref.read(offlineCacheEpochProvider);
    final id = widget.props.topic.id;
    final l10n = AppLocalizations.of(context);
    setState(() => _menuOpen = true);
    final title = widget.props.topic.title.trim();
    final action = await showGfActionMenu<String>(
      context,
      sourceRect: gfMenuSourceRectOf(_moreKey.currentContext!),
      semanticLabel: title.isEmpty ? l10n.topicTitle : title,
      actions: _menuActions(l10n),
    );
    if (!mounted) return;
    setState(() => _menuOpen = false);
    if (action == null ||
        epoch != ref.read(offlineCacheEpochProvider) ||
        id != widget.props.topic.id ||
        !_menuActions(l10n).any((item) => item.value == action)) {
      return;
    }
    await _action(action);
  }

  @override
  Widget build(BuildContext context) => IconButton(
    key: _moreKey,
    tooltip: AppLocalizations.of(context).profileMore,
    icon: const GfSymbol('ellipsis', size: 20),
    onPressed: _busy || _menuOpen ? null : _showMore,
  );
}
