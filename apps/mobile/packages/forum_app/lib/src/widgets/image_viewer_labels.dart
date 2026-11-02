import 'package:ui_kit/ui_kit.dart';
import '../../l10n/app_localizations.dart';

/// One host mapping for every gallery, including single-image sticker previews.
extension ImageViewerLocalizations on AppLocalizations {
  GfImageViewerLabels get imageViewerLabels => GfImageViewerLabels(
    thumbnails: imageViewerThumbnails,
    originalSize: imageViewerOriginalSize,
    fitPreview: imageViewerFitPreview,
    imagePosition: imageViewPosition,
  );
}
