import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';

/// Shared avatar/body geometry. Live avatars may own profile navigation;
/// snapshot avatars are display-only and do not identify a source account.
class ChatMessageRow extends StatelessWidget {
  const ChatMessageRow({
    super.key,
    required this.mine,
    required this.avatar,
    required this.child,
    this.senderName,
  });

  final bool mine;
  final Widget avatar;
  final Widget child;
  final String? senderName;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(vertical: 5),
    child: Row(
      mainAxisAlignment: mine ? MainAxisAlignment.end : MainAxisAlignment.start,
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (!mine) avatar,
        Flexible(
          child: senderName == null
              ? child
              : Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Padding(
                      padding: const EdgeInsets.only(bottom: 5),
                      child: Text(
                        senderName!,
                        style: TextStyle(
                          fontSize: 12,
                          fontWeight: FontWeight.w500,
                          color: GfTheme.colorsOf(context).iconMuted,
                        ),
                      ),
                    ),
                    child,
                  ],
                ),
        ),
        if (mine) ...[const SizedBox(width: 8), avatar],
      ],
    ),
  );
}
