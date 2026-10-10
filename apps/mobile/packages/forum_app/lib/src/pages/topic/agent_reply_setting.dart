import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';
import '../../providers.dart';
import '../../server_messages.dart';

/// Topic bot-reply control: the author gets an on/off row, readers only see
/// a quiet notice while replies are turned off.
class AgentReplySetting extends ConsumerStatefulWidget {
  const AgentReplySetting({
    super.key,
    required this.topicId,
    required this.disabled,
    required this.canManage,
    required this.onChanged,
  });
  final int topicId;
  final bool disabled;
  final bool canManage;
  final Future<void> Function() onChanged;

  @override
  ConsumerState<AgentReplySetting> createState() => _AgentReplySettingState();
}

class _AgentReplySettingState extends ConsumerState<AgentReplySetting> {
  bool _saving = false;
  Object? _error;

  // The switch stays on the confirmed server value; it flips after a reload.
  Future<void> _setAllowed(bool allowed) async {
    if (_saving) return;
    final epoch = ref.read(offlineCacheEpochProvider);
    final topicId = widget.topicId;
    setState(() {
      _saving = true;
      _error = null;
    });
    try {
      await ref
          .read(topicRepositoryProvider)
          .updateAgentReplies(topicId: topicId, disabled: !allowed);
      if (mounted &&
          topicId == widget.topicId &&
          epoch == ref.read(offlineCacheEpochProvider)) {
        await widget.onChanged();
      }
    } catch (error) {
      if (mounted && topicId == widget.topicId) setState(() => _error = error);
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context);
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    if (!widget.canManage) {
      if (!widget.disabled) return const SizedBox.shrink();
      return Padding(
        key: const Key('agent-replies-notice'),
        padding: const EdgeInsets.fromLTRB(16, 12, 16, 4),
        child: Text(
          l.agentRepliesDisabled,
          style: type.caption.copyWith(color: colors.iconMuted),
        ),
      );
    }
    return Padding(
      key: const Key('agent-replies-manage'),
      padding: const EdgeInsets.fromLTRB(12, 12, 12, 4),
      child: DecoratedBox(
        decoration: BoxDecoration(
          color: colors.base200.withValues(alpha: .6),
          borderRadius: BorderRadius.circular(14),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            IgnorePointer(
              ignoring: _saving,
              child: GfSwitchRow(
                title: l.agentRepliesAllow,
                description: l.agentRepliesHelp,
                value: !widget.disabled,
                onChanged: _setAllowed,
              ),
            ),
            if (_error != null)
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 0, 16, 12),
                child: Semantics(
                  liveRegion: true,
                  child: Text(
                    resolveErrorMessage(l, _error!),
                    style: type.caption.copyWith(color: colors.error),
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }
}
