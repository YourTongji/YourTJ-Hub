import { describe, it, expect } from '@ohos/hypium';
import { common } from '@kit.AbilityKit';
import { Any, MethodCall, MethodResult } from '@ohos/flutter_ohos';
import PasskeyPlugin, { passkeyBytes } from '../../../main/ets/auth/PasskeyPlugin';

class CapabilityResult implements MethodResult {
  value: Any = null;
  code: string = '';
  success(value: Any): void { this.value = value; }
  error(code: string, _message: string, _details: Any): void { this.code = code; }
  notImplemented(): void { this.code = 'not_implemented'; }
}

export default function passkeyTest(context: common.UIAbilityContext): void {
  describe('PasskeyEncoding', () => {
    it('queriesTheRealSystemCapabilityWithoutOpeningAuthentication', 0, async () => {
      const result = new CapabilityResult();
      await new PasskeyPlugin(context).onMethodCall(new MethodCall('isAvailable', null), result);
      expect(result.code === 'deviceNotSupported' ||
        (result.code === '' && typeof result.value === 'boolean')).assertTrue();
    });
    it('decodesUnpaddedBase64UrlWithoutChangingBytes', 0, () => {
      const bytes = passkeyBytes('-_8', 1, 64);
      expect(bytes.length).assertEqual(2);
      expect(bytes[0]).assertEqual(251);
      expect(bytes[1]).assertEqual(255);
    });
    it('rejectsInvalidAlphabetAndCredentialSizes', 0, () => {
      for (const value of ['', 'AQ+/', 'A Q', 'A', 'AQI=extra']) {
        let rejected = false;
        try { passkeyBytes(value, 1, 64); } catch (_) { rejected = true; }
        expect(rejected).assertTrue();
      }
      let shortChallengeRejected = false;
      try { passkeyBytes('AQI', 16, 512); } catch (_) { shortChallengeRejected = true; }
      expect(shortChallengeRejected).assertTrue();
    });
  });
}
