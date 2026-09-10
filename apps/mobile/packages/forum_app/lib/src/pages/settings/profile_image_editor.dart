import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../../l10n/app_localizations.dart';
import '../../server_messages.dart';
import '../../widgets/profile_image_crop.dart';

/// Upload failures stay in the editor and preserve the selected crop for retry.
class ProfileImageEditor extends StatefulWidget {
  const ProfileImageEditor({
    super.key,
    required this.source,
    required this.cover,
    required this.onSave,
  });
  final ProfileCropSource source;
  final bool cover;
  final Future<void> Function(Uint8List bytes) onSave;
  @override
  State<ProfileImageEditor> createState() => _ProfileImageEditorState();
}

class _ProfileImageEditorState extends State<ProfileImageEditor> {
  double _zoom = 1, _startZoom = 1;
  Offset _offset = Offset.zero,
      _startOffset = Offset.zero,
      _startFocal = Offset.zero;
  Size _viewport = Size.zero;
  bool _saving = false;
  Object? _error;
  ProfileCropGeometry get _geometry => ProfileCropGeometry(
    source: Size(
      widget.source.width.toDouble(),
      widget.source.height.toDouble(),
    ),
    viewport: _viewport,
    zoom: _zoom,
    offset: _offset,
  );

  Future<void> _save() async {
    if (_saving || _viewport.isEmpty) return;
    setState(() {
      _saving = true;
      _error = null;
    });
    try {
      final bytes = await compute(
        exportProfileCrop,
        ProfileCropExport(
          widget.source,
          _geometry.selection,
          cover: widget.cover,
        ),
      );
      if (!mounted) return;
      await widget.onSave(bytes);
      if (mounted) {
        setState(() => _saving = false);
        WidgetsBinding.instance.addPostFrameCallback((_) {
          if (mounted) Navigator.pop(context, true);
        });
      }
    } catch (error) {
      if (mounted) setState(() => _error = error);
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return PopScope(
      canPop: !_saving,
      child: Scaffold(
        backgroundColor: GfTheme.colorsOf(context).base100,
        appBar: GfAppBar(
          title: Text(widget.cover ? l10n.settingsCover : l10n.settingsAvatar),
          leading: IconButton(
            icon: const Icon(Icons.close),
            tooltip: l10n.commonCancel,
            onPressed: _saving ? null : () => Navigator.pop(context),
          ),
          actions: [
            TextButton(
              onPressed: _saving ? null : _save,
              child: Text(
                _saving ? l10n.settingsAvatarUploading : l10n.commonSave,
              ),
            ),
          ],
        ),
        body: SafeArea(
          top: false,
          child: LayoutBuilder(
            builder: (context, constraints) {
              return SingleChildScrollView(
                padding: const EdgeInsets.all(24),
                child: Center(
                  child: ConstrainedBox(
                    constraints: const BoxConstraints(maxWidth: 560),
                    child: Column(
                      children: [
                        const SizedBox(height: 24),
                        Text(
                          l10n.settingsCropHint,
                          textAlign: TextAlign.center,
                        ),
                        const SizedBox(height: 24),
                        LayoutBuilder(
                          builder: (context, constraints) {
                            final width = constraints.maxWidth;
                            _viewport = Size(
                              width,
                              width / (widget.cover ? 5 : 1),
                            );
                            final geometry = _geometry;
                            final rect = geometry.destination;
                            return Semantics(
                              label: l10n.settingsCropPreview,
                              image: true,
                              child: GestureDetector(
                                key: const Key('profile-image-crop'),
                                onScaleStart: _saving
                                    ? null
                                    : (details) {
                                        _startZoom = _zoom;
                                        _startOffset = geometry.offset;
                                        _startFocal =
                                            details.localFocalPoint -
                                            _viewport.center(Offset.zero);
                                      },
                                onScaleUpdate: _saving
                                    ? null
                                    : (details) => setState(() {
                                        _zoom = (_startZoom * details.scale)
                                            .clamp(1, 4);
                                        final focal =
                                            details.localFocalPoint -
                                            _viewport.center(Offset.zero);
                                        _offset =
                                            (_startOffset - _startFocal) *
                                                (_zoom / _startZoom) +
                                            focal;
                                        _offset = _geometry.offset;
                                      }),
                                child: ClipRect(
                                  child: SizedBox.fromSize(
                                    size: _viewport,
                                    child: Stack(
                                      children: [
                                        Positioned.fromRect(
                                          rect: rect,
                                          child: Image.memory(
                                            widget.source.bytes,
                                            fit: BoxFit.fill,
                                            gaplessPlayback: true,
                                            excludeFromSemantics: true,
                                          ),
                                        ),
                                        if (widget.cover)
                                          Positioned.fill(
                                            child: IgnorePointer(
                                              child: Row(
                                                children: [
                                                  Expanded(
                                                    child: Container(
                                                      color: Colors.black26,
                                                    ),
                                                  ),
                                                  const Expanded(
                                                    flex: 3,
                                                    child: SizedBox.expand(),
                                                  ),
                                                  Expanded(
                                                    child: Container(
                                                      color: Colors.black26,
                                                    ),
                                                  ),
                                                ],
                                              ),
                                            ),
                                          ),
                                      ],
                                    ),
                                  ),
                                ),
                              ),
                            );
                          },
                        ),
                        if (widget.cover)
                          Padding(
                            padding: const EdgeInsets.only(top: 12),
                            child: Text(
                              l10n.settingsCoverSafeArea,
                              textAlign: TextAlign.center,
                            ),
                          ),
                        const SizedBox(height: 24),
                        Row(
                          children: [
                            const Icon(Icons.zoom_out),
                            Expanded(
                              child: Slider(
                                value: _zoom,
                                min: 1,
                                max: 4,
                                label: '${_zoom.toStringAsFixed(1)}×',
                                semanticFormatterCallback: (value) =>
                                    '${l10n.settingsCropZoom} ${value.toStringAsFixed(1)}×',
                                onChanged: _saving
                                    ? null
                                    : (value) => setState(() {
                                        _offset =
                                            _geometry.offset * (value / _zoom);
                                        _zoom = value;
                                        _offset = _geometry.offset;
                                      }),
                              ),
                            ),
                            const Icon(Icons.zoom_in),
                          ],
                        ),
                        TextButton.icon(
                          onPressed: _saving
                              ? null
                              : () => setState(() {
                                  _zoom = 1;
                                  _offset = Offset.zero;
                                }),
                          icon: const Icon(Icons.restart_alt),
                          label: Text(l10n.settingsCropReset),
                        ),
                        if (_saving) const LinearProgressIndicator(),
                        if (_error != null)
                          Padding(
                            padding: const EdgeInsets.only(top: 16),
                            child: Text(
                              resolveErrorMessage(l10n, _error!),
                              style: TextStyle(
                                color: Theme.of(context).colorScheme.error,
                              ),
                              textAlign: TextAlign.center,
                            ),
                          ),
                      ],
                    ),
                  ),
                ),
              );
            },
          ),
        ),
      ),
    );
  }
}
