import { describe, it, expect } from '@ohos/hypium';
import { common } from '@kit.AbilityKit';
import { Any, MethodCall, MethodResult } from '@ohos/flutter_ohos';
import ClipboardPlugin, { isGif } from '../../../main/ets/system/ClipboardPlugin';

class ClipboardResult implements MethodResult {
  code: string = '';
  private finish: () => void = () => {};
  completed: Promise<void> = new Promise<void>(resolve => { this.finish = resolve; });
  success(_value: Any): void { this.finish(); }
  error(code: string, _message: string, _details: Any): void { this.code = code; this.finish(); }
  notImplemented(): void { this.code = 'not_implemented'; this.finish(); }
}

export default function clipboardTest(context: common.UIAbilityContext): void {
  describe('ClipboardBoundaries', () => {
    it('recognizesBothGifHeadersAndRejectsOtherData', 0, () => {
      expect(isGif(new Uint8Array([71,73,70,56,55,97]))).assertTrue();
      expect(isGif(new Uint8Array([71,73,70,56,57,97]))).assertTrue();
      expect(isGif(new Uint8Array([71,73,70,56,57,0]))).assertFalse();
      expect(isGif(new Uint8Array(0))).assertFalse();
    });
    it('rejectsEmptyWritesBeforeChangingTheClipboard', 0, async () => {
      const plugin = new ClipboardPlugin(context);
      const html = new ClipboardResult();
      plugin.onMethodCall(new MethodCall('copyHtml', ''), html);
      await html.completed;
      expect(html.code).assertEqual('clipboard_failed');
      const image = new ClipboardResult();
      plugin.onMethodCall(new MethodCall('copyImage', new Uint8Array(0)), image);
      await image.completed;
      expect(image.code).assertEqual('clipboard_failed');
    });
  });
}
