import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../l10n/app_localizations.dart';
import 'release_notes.dart';
import 'release_notes_view.dart';

/// Update prompt content for [showGfBottomSheet]. Notes scroll; status and
/// actions stay pinned at the bottom edge, within thumb reach.
class UpdatePromptSheet extends StatefulWidget {
  const UpdatePromptSheet({
    super.key,
    required this.version,
    this.installedVersion,
    this.sizeBytes,
    required this.notes,
    required this.historyComplete,
    this.working = false,
    this.failed = false,
    this.needsPermission = false,
    this.ready = false,
    this.receivedBytes = 0,
    this.failureText,
    required this.primaryLabel,
    required this.onPrimary,
    this.onCancel,
    required this.onLater,
    this.onSkip,
  });

  final String version;

  /// Shown as `installed → target` so a multi-version jump is explicit.
  final String? installedVersion;
  final int? sizeBytes;

  /// All notes for the update; the prompt shows required notes and the first
  /// five others, and expands in place to the rest.
  final List<ReleaseNote> notes;
  final bool historyComplete;
  final bool working;
  final bool failed;
  final bool needsPermission;
  final bool ready;
  final int receivedBytes;
  final String? failureText;
  final String primaryLabel;
  final VoidCallback onPrimary;
  final VoidCallback? onCancel;
  final VoidCallback onLater;
  final VoidCallback? onSkip;

  @override
  State<UpdatePromptSheet> createState() => _UpdatePromptSheetState();
}

class _UpdatePromptSheetState extends State<UpdatePromptSheet> {
  bool _expanded = false;
  bool _moreBelow = false;

  static String _megabytes(int bytes) =>
      '${(bytes / 1024 / 1024).toStringAsFixed(1)} MB';

  bool _onMetrics(ScrollMetrics metrics) {
    final moreBelow = metrics.extentAfter > 1;
    if (moreBelow != _moreBelow) setState(() => _moreBelow = moreBelow);
    return false;
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    final secondary = colors.baseContent.withValues(alpha: 0.6);
    final statusText = widget.working
        ? (widget.receivedBytes == 0
              ? l10n.updatePreparing
              : l10n.updateDownloading)
        : widget.failed
        ? widget.failureText ?? l10n.updateFailed
        : widget.needsPermission
        ? l10n.updatePermission
        : widget.ready
        ? l10n.updateReady
        : null;
    final remaining = widget.notes.where((note) => !note.required).length - 5;
    final shown = _expanded ? widget.notes : promptReleaseNotes(widget.notes);
    final maxHeight = MediaQuery.sizeOf(context).height * .88;
    final installed = widget.installedVersion;
    return ConstrainedBox(
      constraints: BoxConstraints(maxHeight: maxHeight),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Flexible(
            child: NotificationListener<ScrollMetricsNotification>(
              onNotification: (notification) =>
                  _onMetrics(notification.metrics),
              child: NotificationListener<ScrollNotification>(
                onNotification: (notification) =>
                    _onMetrics(notification.metrics),
                // A short fade marks notes that continue below the fold.
                child: ShaderMask(
                  blendMode: BlendMode.dstIn,
                  shaderCallback: (bounds) => LinearGradient(
                    begin: Alignment.topCenter,
                    end: Alignment.bottomCenter,
                    colors: [
                      Colors.black,
                      _moreBelow ? Colors.transparent : Colors.black,
                    ],
                    stops: [
                      bounds.height <= 32 ? 0 : 1 - 32 / bounds.height,
                      1,
                    ],
                  ).createShader(bounds),
                  child: SingleChildScrollView(
                    padding: const EdgeInsets.fromLTRB(24, 24, 24, 8),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        Row(
                          children: [
                            ExcludeSemantics(
                              child: Container(
                                width: 48,
                                height: 48,
                                decoration: BoxDecoration(
                                  color: colors.base300,
                                  borderRadius: BorderRadius.circular(14),
                                ),
                                alignment: Alignment.center,
                                child: const GfLogo(size: 30),
                              ),
                            ),
                            const SizedBox(width: 14),
                            Expanded(
                              child: Column(
                                crossAxisAlignment: CrossAxisAlignment.start,
                                children: [
                                  Semantics(
                                    header: true,
                                    child: Text(
                                      l10n.updateAvailable,
                                      style: type.title2,
                                    ),
                                  ),
                                  const SizedBox(height: 2),
                                  Wrap(
                                    crossAxisAlignment:
                                        WrapCrossAlignment.center,
                                    spacing: 6,
                                    children: [
                                      if (installed != null &&
                                          installed != widget.version) ...[
                                        Text(
                                          installed,
                                          style: type.caption.copyWith(
                                            color: secondary,
                                          ),
                                        ),
                                        ExcludeSemantics(
                                          child: GfSymbol(
                                            'arrow-right',
                                            size: 14,
                                            color: secondary,
                                          ),
                                        ),
                                      ],
                                      Text(
                                        widget.version,
                                        style: type.caption.copyWith(
                                          color: colors.baseContent,
                                          fontWeight: FontWeight.w600,
                                        ),
                                      ),
                                      if (widget.sizeBytes != null)
                                        Text(
                                          '· ${_megabytes(widget.sizeBytes!)}',
                                          style: type.caption.copyWith(
                                            color: secondary,
                                          ),
                                        ),
                                    ],
                                  ),
                                ],
                              ),
                            ),
                          ],
                        ),
                        if (shown.isNotEmpty || !widget.historyComplete) ...[
                          const SizedBox(height: 24),
                          ReleaseNotesView(
                            notes: shown,
                            historyComplete: widget.historyComplete,
                          ),
                        ],
                        if (remaining > 0)
                          Align(
                            alignment: AlignmentDirectional.centerStart,
                            child: Padding(
                              padding: const EdgeInsets.only(top: 8),
                              child: GfButton(
                                label: _expanded
                                    ? l10n.releaseNotesFewer
                                    : l10n.releaseNotesShowRemaining(remaining),
                                icon: GfSymbol(
                                  _expanded ? 'chevron-up' : 'chevron-down',
                                  size: 16,
                                ),
                                onPressed: () =>
                                    setState(() => _expanded = !_expanded),
                                variant: GfButtonVariant.link,
                                size: GfButtonSize.small,
                              ),
                            ),
                          ),
                      ],
                    ),
                  ),
                ),
              ),
            ),
          ),
          // Actions stay pinned. With very large text this region scrolls
          // within 60% of the sheet; status and the primary action come first,
          // so only the secondary actions can move below the fold.
          ConstrainedBox(
            constraints: BoxConstraints(maxHeight: maxHeight * .6),
            child: SingleChildScrollView(
              padding: const EdgeInsets.fromLTRB(24, 8, 24, 12),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  if (statusText != null) ...[
                    Row(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Expanded(
                          child: Semantics(
                            key: const ValueKey('update-status'),
                            container: true,
                            liveRegion: true,
                            label: statusText,
                            excludeSemantics: true,
                            child: Text(
                              statusText,
                              style: type.caption.copyWith(
                                color: widget.failed ? colors.error : secondary,
                              ),
                            ),
                          ),
                        ),
                        if (widget.working &&
                            widget.receivedBytes > 0 &&
                            widget.sizeBytes != null)
                          Padding(
                            padding: const EdgeInsetsDirectional.only(
                              start: 12,
                            ),
                            child: Text(
                              '${_megabytes(widget.receivedBytes)} / ${_megabytes(widget.sizeBytes!)}',
                              style: type.caption.copyWith(color: secondary),
                            ),
                          ),
                      ],
                    ),
                    const SizedBox(height: 8),
                  ],
                  if (widget.working) ...[
                    ExcludeSemantics(
                      child: LinearProgressIndicator(
                        minHeight: 4,
                        borderRadius: BorderRadius.circular(2),
                        backgroundColor: colors.base300,
                        color: colors.primary,
                        value: widget.receivedBytes == 0
                            ? (GfMotion.reducedOf(context) ? 0 : null)
                            : widget.receivedBytes / (widget.sizeBytes ?? 1),
                      ),
                    ),
                    const SizedBox(height: 16),
                    GfButton(
                      label: l10n.updateCancelDownload,
                      onPressed: widget.onCancel,
                      variant: GfButtonVariant.secondary,
                      size: GfButtonSize.large,
                      expanded: true,
                    ),
                  ] else ...[
                    GfButton(
                      label: widget.primaryLabel,
                      onPressed: widget.onPrimary,
                      size: GfButtonSize.large,
                      expanded: true,
                    ),
                    const SizedBox(height: 4),
                    Row(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        if (widget.onSkip != null)
                          Expanded(
                            child: GfButton(
                              label: l10n.updateSkip,
                              onPressed: widget.onSkip,
                              variant: GfButtonVariant.muted,
                              expanded: true,
                            ),
                          ),
                        Expanded(
                          child: GfButton(
                            label: l10n.updateLater,
                            onPressed: widget.onLater,
                            variant: GfButtonVariant.muted,
                            expanded: true,
                          ),
                        ),
                      ],
                    ),
                  ],
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}
