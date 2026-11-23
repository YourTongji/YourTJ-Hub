import {
  FlutterPlugin, FlutterPluginBinding, MethodCall, MethodCallHandler, MethodChannel, MethodResult
} from '@ohos/flutter_ohos';
import { common } from '@kit.AbilityKit';
import { BusinessError, deviceInfo } from '@kit.BasicServicesKit';
import { util } from '@kit.ArkTS';
// Keep API 18 startup independent of the API 20 authenticator module.
import lazy { fido2 } from '@kit.OnlineAuthenticationKit';

interface WireDescriptor {
  type: fido2.PublicKeyCredentialType;
  id: string;
  transports?: fido2.AuthenticatorTransport[];
}

interface WireUser {
  id: string;
  name: string;
  displayName: string;
}

interface WireOptions {
  challenge: string;
  rpId?: string;
  rp: fido2.PublicKeyCredentialRpEntity;
  user: WireUser;
  pubKeyCredParams: fido2.PublicKeyCredentialParameters[];
  timeout?: number;
  allowCredentials?: WireDescriptor[];
  excludeCredentials?: WireDescriptor[];
  userVerification?: fido2.UserVerificationRequirement;
  authenticatorSelection?: fido2.AuthenticatorSelectionCriteria;
  attestation?: fido2.AttestationConveyancePreference;
}

export function passkeyBytes(value: string, minimum: number, maximum: number): Uint8Array {
  if (typeof value !== 'string' || value.length > maximum * 2 || !/^[A-Za-z0-9_-]+={0,2}$/.test(value)) {
    throw new Error('Invalid credential encoding');
  }
  const padded = value.replace(/=+$/, '');
  const decoded = new util.Base64Helper().decodeSync(
    padded + '='.repeat((4 - padded.length % 4) % 4), util.Type.BASIC_URL_SAFE);
  if (decoded.length < minimum || decoded.length > maximum) throw new Error('Invalid credential size');
  return decoded;
}

function descriptors(values?: WireDescriptor[]): fido2.PublicKeyCredentialDescriptor[] {
  if (!values) return [];
  if (!Array.isArray(values) || values.length > 100) throw new Error('Invalid credential list');
  return values.map((value: WireDescriptor): fido2.PublicKeyCredentialDescriptor => {
    if (value.type !== fido2.PublicKeyCredentialType.PUBLIC_KEY) throw new Error('Invalid credential type');
    return { type: value.type, id: passkeyBytes(value.id, 1, 1024), transports: value.transports };
  });
}

export default class PasskeyPlugin implements FlutterPlugin, MethodCallHandler {
  private channel: MethodChannel | null = null;
  private busy: boolean = false;
  private context: common.UIAbilityContext;

  constructor(context: common.UIAbilityContext) { this.context = context; }
  getUniqueClassName(): string { return 'YourTJPasskeys'; }

  onAttachedToEngine(binding: FlutterPluginBinding): void {
    this.channel = new MethodChannel(binding.getBinaryMessenger(), 'yourtj/passkeys');
    this.channel.setMethodCallHandler(this);
  }

  onDetachedFromEngine(_binding: FlutterPluginBinding): void {
    this.channel?.setMethodCallHandler(null);
    this.channel = null;
  }

  async onMethodCall(call: MethodCall, result: MethodResult): Promise<void> {
    if (!['isAvailable', 'register', 'authenticate'].includes(call.method)) {
      result.notImplemented();
      return;
    }
    if (deviceInfo.sdkApiVersion < 20 || !canIUse('SystemCapability.Security.FIDO2')) {
      if (call.method === 'isAvailable') result.success(false);
      else result.error('deviceNotSupported', 'System authenticator is unavailable', null);
      return;
    }
    if (this.busy) {
      result.error('busy', 'An authenticator operation is already active', null);
      return;
    }
    this.busy = true;
    try {
      if (call.method === 'isAvailable') {
        const authenticators = await fido2.getPlatformAuthenticators(this.context);
        result.success(authenticators.some((item: fido2.AuthenticatorMetadata): boolean => item.isAvailable));
        return;
      }
      const encoded = call.args as string;
      if (typeof encoded !== 'string' || encoded.length > 65536) throw new Error('Invalid credential request');
      const options = JSON.parse(encoded) as WireOptions;
      const challenge = passkeyBytes(options.challenge, 16, 512);
      if (call.method === 'register') {
        const publicKey: fido2.PublicKeyCredentialCreationOptions = {
          rp: options.rp,
          user: { id: passkeyBytes(options.user.id, 1, 64), name: options.user.name,
            displayName: options.user.displayName },
          challenge: challenge,
          pubKeyCredParams: options.pubKeyCredParams,
          timeout: options.timeout,
          excludeCredentials: descriptors(options.excludeCredentials),
          authenticatorSelection: options.authenticatorSelection,
          attestation: options.attestation
        };
        const credential = await fido2.register(this.context, { publicKey: publicKey });
        result.success(JSON.stringify(credential.registrationResponseJson));
      } else {
        const publicKey: fido2.PublicKeyCredentialRequestOptions = {
          challenge: challenge,
          rpId: options.rpId,
          timeout: options.timeout,
          allowCredentials: descriptors(options.allowCredentials),
          userVerification: options.userVerification
        };
        const credential = await fido2.authenticate(this.context, { publicKey: publicKey,
          mediation: fido2.CredentialMediationRequirement.OPTIONAL });
        result.success(JSON.stringify(credential.authenticationResponseJson));
      }
    } catch (error) {
      const code = (error as BusinessError).code;
      const cancelled = [1021300004, 1021300005, 1021300010, 1021300013, 1021310006].includes(code);
      const unsupported = [801, 1021300001].includes(code);
      result.error(cancelled ? 'cancelled' : unsupported ? 'deviceNotSupported' : 'passkey_failed',
        'System authenticator operation did not complete', null);
    } finally {
      this.busy = false;
    }
  }
}
