import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../l10n/app_localizations.dart';
import '../asset_url.dart';
import '../providers.dart';
import '../server_messages.dart';
import 'chat_forwarding.dart';

/// Existing conversations and suggested contacts share the new-chat directory.
/// Selection is local; only the explicit send button creates a delivery batch.
class ForwardMessagesPage extends ConsumerStatefulWidget {
  const ForwardMessagesPage({
    super.key,
    required this.convId,
    required this.messageIds,
    required this.ownerEpoch,
  });
  final int convId;
  final int ownerEpoch;
  final List<int> messageIds;
  @override
  ConsumerState<ForwardMessagesPage> createState() =>
      _ForwardMessagesPageState();
}

class _ForwardMessagesPageState extends ConsumerState<ForwardMessagesPage> {
  late final int _epoch = widget.ownerEpoch;
  List<UserConnectionPayload>? _users;
  Object? _error;
  final _selected = <int>{};
  String _query = '';
  late bool _merged = widget.messageIds.length > 10;
  bool get _current => mounted && _epoch == ref.read(offlineCacheEpochProvider);
  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    if (!_current) return;
    setState(() => _error = null);
    try {
      final page = await ref.read(pageRepositoryProvider).fetch('/messages');
      final props = parsePageProps<MessagesPageProps>(page);
      if (props == null) throw const FormatException('missing messages page');
      final users = <int, UserConnectionPayload>{
        for (final user in props.suggestedUsers) user.id: user,
        for (final conv in props.conversations)
          conv.peerId: UserConnectionPayload(
            id: conv.peerId,
            username: conv.peerUsername,
            nickname: conv.peerNickname ?? '',
            avatarUrl: conv.peerAvatar,
            bio: '',
            url: conv.peerUrl,
          ),
      }..remove(page.layout.viewer.id);
      if (_current) {
        setState(
          () => _users = users.values
              .where((u) => u.id > 0 && !u.isSelf)
              .toList(),
        );
      }
    } catch (error) {
      if (_current) setState(() => _error = error);
    }
  }

  Future<void> _send(ChatForwarding controller) async {
    if (!_current || controller.busy) return;
    if (!controller.unfinished) {
      controller.create(
        messageIds: widget.messageIds,
        merged: _merged,
        recipients: _users!.where((u) => _selected.contains(u.id)).toList(),
      );
    }
    await controller.send();
    if (!mounted || !_current) return;
    if (controller.batch?.complete ?? false) {
      showGfToast(context, AppLocalizations.of(context).messageForwardSuccess);
      Navigator.of(context).pop();
    }
  }

  Future<void> _discard(ChatForwarding controller) async {
    final l10n = AppLocalizations.of(context);
    final discard = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(l10n.messageForwardCancelRemaining),
        content: Text(l10n.messageForwardAbandonNotice),
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
    if (discard == true && mounted && _current) {
      controller.discard();
      Navigator.pop(context);
    }
  }

  @override
  Widget build(BuildContext context) {
    if (ref.watch(offlineCacheEpochProvider) != _epoch) {
      return const SizedBox.shrink();
    }
    final controller = ref.watch(chatForwardingProvider(widget.convId));
    final batch = controller.unfinished ? controller.batch : null;
    final l10n = AppLocalizations.of(context);
    final users = _users
        ?.where(
          (u) => '${u.nickname} ${u.username}'.toLowerCase().contains(
            _query.toLowerCase(),
          ),
        )
        .toList();
    return PopScope(
      canPop: !controller.busy,
      child: Scaffold(
        appBar: GfAppBar(title: Text(l10n.messageForwardTitle)),
        body: SafeArea(
          top: false,
          child: Column(
            children: [
              Expanded(
                child: ListView(
                  keyboardDismissBehavior:
                      ScrollViewKeyboardDismissBehavior.onDrag,
                  children: [
                    Padding(
                      padding: const EdgeInsets.all(16),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.stretch,
                        children: [
                          Text(
                            l10n.messageForwardCount(
                              batch?.messageIds.length ??
                                  widget.messageIds.length,
                            ),
                            style: Theme.of(context).textTheme.titleMedium,
                          ),
                          const SizedBox(height: 8),
                          Text(l10n.messageForwardExplanation),
                          if (widget.messageIds.length > 10 && batch == null)
                            Text(l10n.messageForwardIndividualLimit),
                          if (batch == null) ...[
                            const SizedBox(height: 12),
                            SegmentedButton<bool>(
                              segments: [
                                ButtonSegment(
                                  value: false,
                                  enabled: widget.messageIds.length <= 10,
                                  label: Text(l10n.messageForwardIndividual),
                                ),
                                ButtonSegment(
                                  value: true,
                                  label: Text(l10n.messageForwardMerged),
                                ),
                              ],
                              selected: {_merged},
                              onSelectionChanged: (value) =>
                                  setState(() => _merged = value.single),
                            ),
                            const SizedBox(height: 12),
                            TextField(
                              onChanged: (value) =>
                                  setState(() => _query = value),
                              decoration: InputDecoration(
                                hintText: l10n.messageForwardSearch,
                                prefixIcon: const Icon(Icons.search),
                              ),
                            ),
                          ] else
                            Text(
                              batch.merged
                                  ? l10n.messageForwardMerged
                                  : l10n.messageForwardIndividual,
                            ),
                        ],
                      ),
                    ),
                    batch != null
                        ? Column(
                            children: [
                              for (final recipient in batch.recipients)
                                ListTile(
                                  leading: GfAvatar(
                                    src: resolveApiAssetUrl(
                                      recipient.user.avatarUrl,
                                    ),
                                    size: 40,
                                  ),
                                  title: Text(
                                    recipient.user.nickname.isEmpty
                                        ? recipient.user.username
                                        : recipient.user.nickname,
                                  ),
                                  subtitle: Text(switch (recipient.state) {
                                    ForwardDeliveryState.pending =>
                                      l10n.messageForwardPending,
                                    ForwardDeliveryState.sending =>
                                      l10n.messageForwardSending,
                                    ForwardDeliveryState.sent =>
                                      l10n.messageForwardSuccess,
                                    ForwardDeliveryState.failed =>
                                      resolveErrorMessage(
                                        l10n,
                                        recipient.error!,
                                      ),
                                  }),
                                  trailing:
                                      recipient.state ==
                                          ForwardDeliveryState.sent
                                      ? const Icon(Icons.check_circle_outline)
                                      : recipient.state ==
                                            ForwardDeliveryState.sending
                                      ? const SizedBox(
                                          width: 20,
                                          height: 20,
                                          child: CircularProgressIndicator(
                                            strokeWidth: 2,
                                          ),
                                        )
                                      : null,
                                ),
                            ],
                          )
                        : _error != null
                        ? Center(
                            child: Column(
                              mainAxisSize: MainAxisSize.min,
                              children: [
                                Text(resolveErrorMessage(l10n, _error!)),
                                TextButton(
                                  onPressed: _load,
                                  child: Text(l10n.commonRetry),
                                ),
                              ],
                            ),
                          )
                        : users == null
                        ? const Center(child: CircularProgressIndicator())
                        : users.isEmpty
                        ? Center(
                            child: Text(l10n.messageForwardEmptyRecipients),
                          )
                        : Column(
                            children: users.map((user) {
                              return CheckboxListTile(
                                checkboxShape: const CircleBorder(),
                                secondary: GfAvatar(
                                  src: resolveApiAssetUrl(user.avatarUrl),
                                  size: 40,
                                ),
                                title: Text(
                                  user.nickname.isEmpty
                                      ? user.username
                                      : user.nickname,
                                ),
                                subtitle: Text('@${user.username}'),
                                value: _selected.contains(user.id),
                                onChanged: (selected) {
                                  if (selected == true &&
                                      _selected.length >= 10) {
                                    showGfToast(
                                      context,
                                      l10n.messageForwardRecipientLimit(10),
                                    );
                                    return;
                                  }
                                  setState(() {
                                    if (selected == true) {
                                      _selected.add(user.id);
                                    } else {
                                      _selected.remove(user.id);
                                    }
                                  });
                                },
                              );
                            }).toList(),
                          ),
                  ],
                ),
              ),
              Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    FilledButton(
                      onPressed:
                          controller.busy ||
                              (batch == null && _selected.isEmpty)
                          ? null
                          : () => _send(controller),
                      child: Text(
                        controller.busy
                            ? l10n.messageForwardSending
                            : batch != null
                            ? l10n.messageForwardRetry
                            : '${l10n.messageForwardConfirm} · ${l10n.messageForwardTargets(_selected.length)}',
                      ),
                    ),
                    if (batch != null && !controller.busy)
                      TextButton(
                        onPressed: () => _discard(controller),
                        child: Text(l10n.messageForwardCancelRemaining),
                      ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
