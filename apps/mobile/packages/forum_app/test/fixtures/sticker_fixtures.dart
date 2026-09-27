import 'package:core/core.dart';

const composerSticker = StickerItemPayload(
  name: 'smile',
  url: '/smile.png',
  displayName: 'Smile',
);

class ComposerStickerRepository extends StickerRepository {
  ComposerStickerRepository(super.client);

  @override
  Future<List<StickerItemPayload>> list() async => [composerSticker];

  @override
  Future<List<StickerItemPayload>> mine() async => [];

  @override
  Future<List<StickerItemPayload>> resolve(List<String> names) async => [];
}
