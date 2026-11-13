import { describe, it, expect } from '@ohos/hypium';
import { imageExtension } from '../../../main/ets/system/GalleryPlugin';

export default function galleryTest(): void {
  describe('GalleryEncoding', () => {
    it('usesImageBytesInsteadOfTheSuggestedFilename', 0, () => {
      expect(imageExtension(new Uint8Array([137,80,78,71,13,10,26,10,0,0,0,0]))).assertEqual('png');
      expect(imageExtension(new Uint8Array([255,216,255,224,0,0,0,0,0,0,0,0]))).assertEqual('jpg');
      expect(imageExtension(new Uint8Array([71,73,70,56,57,97,0,0,0,0,0,0]))).assertEqual('gif');
      expect(imageExtension(new Uint8Array([82,73,70,70,0,0,0,0,87,69,66,80]))).assertEqual('webp');
      expect(imageExtension(new Uint8Array(12))).assertNull();
      expect(imageExtension(new Uint8Array(0))).assertNull();
    });
  });
}
