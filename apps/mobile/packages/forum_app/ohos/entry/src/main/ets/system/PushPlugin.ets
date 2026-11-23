import {
  FlutterPlugin, FlutterPluginBinding, MethodChannel, MethodCall, MethodResult, MethodCallHandler
} from '@ohos/flutter_ohos';
import { UIAbility, Want } from '@kit.AbilityKit';
import { notificationManager } from '@kit.NotificationKit';
import { pushService, pushCommon } from '@kit.PushKit';

export default class PushPlugin implements FlutterPlugin, MethodCallHandler {
  private ability: UIAbility;
  private channel: MethodChannel | null = null;
  private enabled: boolean = false;
  private listening: boolean = false;
  private profileId: string = '';
  private boundProfileId: string = '';
  private dartReady: boolean = false;
  private initialRoute: string | null = null;
  private operations: Promise<void> = Promise.resolve();
  private tokenUpdated = async (token: string): Promise<void> => {
    try {
      if (this.enabled && await notificationManager.isNotificationEnabled()) {
        this.channel?.invokeMethod('token', token);
      }
    } catch (_) {}
  };

  constructor(ability: UIAbility) { this.ability = ability; }
  getUniqueClassName(): string { return 'YourTJPush'; }

  private configured(): boolean {
    const clientId = this.ability.context.currentHapModuleInfo.metadata?.find(item => item.name === 'client_id')?.value;
    return !!clientId?.trim();
  }

  onAttachedToEngine(binding: FlutterPluginBinding): void {
    this.channel = new MethodChannel(binding.getBinaryMessenger(), 'yourtj/push');
    this.channel.setMethodCallHandler(this);
  }

  onDetachedFromEngine(_binding: FlutterPluginBinding): void {
    this.enabled = false;
    this.removeListener();
    this.channel?.setMethodCallHandler(null);
    this.channel = null;
    this.dartReady = false;
  }

  acceptWant(want: Want): void {
    const match = /^yourtj:\/\/push\?route=([^&]+)$/.exec(want.uri ?? '');
    if (!match) return;
    try {
      const route = decodeURIComponent(match[1]);
      if (!route.startsWith('/') || route.startsWith('//') || route.length > 4096) return;
      if (this.dartReady) this.channel?.invokeMethod('opened', route);
      else this.initialRoute = route;
    } catch (_) {}
  }

  private async bindProfile(): Promise<void> {
    if (this.boundProfileId && this.boundProfileId !== this.profileId) {
      await pushService.unbindAppProfileId(this.boundProfileId);
      this.boundProfileId = '';
    }
    if (this.profileId && this.boundProfileId !== this.profileId) {
      await pushService.bindAppProfileId(pushCommon.AppProfileType.PROFILE_TYPE_APPLICATION_ACCOUNT, this.profileId);
      this.boundProfileId = this.profileId;
    }
  }

  private removeListener(): void {
    if (this.listening) pushService.off('tokenUpdate', this.tokenUpdated);
    this.listening = false;
  }

  onMethodCall(call: MethodCall, result: MethodResult): void {
    this.operations = this.operations.then(async (): Promise<void> => {
      try {
        switch (call.method) {
          case 'configured': result.success(this.configured()); return;
          case 'permission': result.success(await notificationManager.isNotificationEnabled()); return;
          case 'requestPermission':
            try { await notificationManager.requestEnableNotification(this.ability.context); } catch (_) {}
            result.success(await notificationManager.isNotificationEnabled());
            return;
          case 'register': {
            if (!this.channel || !this.profileId || !this.configured() || !await notificationManager.isNotificationEnabled()) {
              result.error('push_unavailable', 'Push is not configured or authorized', null);
              return;
            }
            const token = await pushService.getToken();
            if (!this.channel) {
              await pushService.deleteToken();
              result.error('push_detached', 'Push engine is no longer attached', null);
              return;
            }
            try { await this.bindProfile(); }
            catch (_) { await pushService.deleteToken(); throw new Error('Push account binding failed'); }
            this.enabled = true;
            if (!this.listening) {
              pushService.on('tokenUpdate', this.ability, this.tokenUpdated);
              this.listening = true;
            }
            result.success(token);
            return;
          }
          case 'stop': {
            this.enabled = false;
            this.removeListener();
            this.profileId = '';
            let failed = false;
            try { if (this.boundProfileId) await this.bindProfile(); } catch (_) { failed = true; }
            try { if (this.configured()) await pushService.deleteToken(); } catch (_) { failed = true; }
            if (failed) throw new Error('Push cleanup failed');
            result.success(null);
            return;
          }
          case 'initialRoute':
            this.dartReady = true;
            result.success(this.initialRoute);
            this.initialRoute = null;
            return;
          case 'presentation': {
            const args = call.args as Map<string, string | number | boolean>;
            const profile = args.get('profileId');
            if (typeof profile !== 'string' || (profile !== '' && !/^[a-f0-9]{64}$/.test(profile))) {
              result.error('invalid_push_profile', 'Invalid push account profile', null);
              return;
            }
            this.profileId = profile;
            if (this.enabled) await this.bindProfile();
            result.success(null);
            return;
          }
          default: result.notImplemented();
        }
      } catch (_) { result.error('push_failed', 'Native push operation failed', null); }
    });
  }
}
