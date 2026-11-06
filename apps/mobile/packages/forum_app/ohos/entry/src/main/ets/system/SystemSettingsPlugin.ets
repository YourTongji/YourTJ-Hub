import {
  FlutterPlugin, FlutterPluginBinding, MethodCall, MethodCallHandler, MethodChannel, MethodResult
} from '@ohos/flutter_ohos';
import { common, ConfigurationConstant, abilityAccessCtrl } from '@kit.AbilityKit';
import { notificationManager } from '@kit.NotificationKit';
import { deviceInfo, settings } from '@kit.BasicServicesKit';

export default class SystemSettingsPlugin implements FlutterPlugin, MethodCallHandler {
  private context: common.UIAbilityContext;
  private channels: MethodChannel[] = [];

  constructor(context: common.UIAbilityContext) { this.context = context; }
  getUniqueClassName(): string { return 'YourTJSystemSettings'; }

  onAttachedToEngine(binding: FlutterPluginBinding): void {
    for (const name of ['com.spencerccf.app_settings/methods', 'yourtj/startup']) {
      const channel = new MethodChannel(binding.getBinaryMessenger(), name);
      channel.setMethodCallHandler(this);
      this.channels.push(channel);
    }
  }

  onDetachedFromEngine(_binding: FlutterPluginBinding): void {
    for (const channel of this.channels) channel.setMethodCallHandler(null);
    this.channels = [];
  }

  async onMethodCall(call: MethodCall, result: MethodResult): Promise<void> {
    try {
      switch (call.method) {
        case 'openSettings': {
          const args = call.args as Map<string, string>;
          if (args.get('type') === 'notification') {
            await notificationManager.openNotificationSettings(this.context);
          } else if (args.get('type') === 'clipboard') {
            await abilityAccessCtrl.createAtManager().requestPermissionOnSetting(
              this.context, ['ohos.permission.READ_PASTEBOARD']);
          } else {
            if (deviceInfo.sdkApiVersion >= 26) {
              settings.openAppDetailSettingsPage(this.context, this.context.applicationInfo.name);
            } else {
              await this.context.startAbility({
                bundleName: 'com.huawei.hmos.settings',
                abilityName: 'com.huawei.hmos.settings.MainAbility',
                uri: 'application_info_entry',
                parameters: { pushParams: this.context.applicationInfo.name }
              });
            }
          }
          result.success(null);
          return;
        }
        case 'setThemeMode': {
          const args = call.args as Map<string, string>;
          const mode = args.get('mode');
          if (mode !== 'light' && mode !== 'dark' && mode !== 'system') throw new Error('Invalid theme mode');
          this.context.getApplicationContext().setColorMode(mode === 'dark' ?
            ConfigurationConstant.ColorMode.COLOR_MODE_DARK : mode === 'light' ?
            ConfigurationConstant.ColorMode.COLOR_MODE_LIGHT : ConfigurationConstant.ColorMode.COLOR_MODE_NOT_SET);
          result.success(true);
          return;
        }
        case 'reportFullyDrawn':
          this.context.reportDrawnCompleted(error => {
            if (error) result.error('startup_report_failed', 'Could not report the first complete frame', null);
            else result.success(true);
          });
          return;
        case 'getDeviceProfile': {
          const profile = new Map<string, string>();
          profile.set('device', 'HarmonyOS');
          profile.set('osVersion', deviceInfo.displayVersion);
          result.success(profile);
          return;
        }
        default: result.notImplemented();
      }
    } catch (_) { result.error('system_settings_failed', 'System settings operation failed', null); }
  }
}
