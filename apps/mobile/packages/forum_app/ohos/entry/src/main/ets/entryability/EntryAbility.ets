
import { FlutterAbility, FlutterEngine } from '@ohos/flutter_ohos';
import { GeneratedPluginRegistrant } from '../plugins/GeneratedPluginRegistrant';
import { AbilityConstant, Want } from '@kit.AbilityKit';
import ScheduleWidgetPlugin from '../widget/ScheduleWidgetPlugin';
import SecureStoragePlugin from '../storage/SecureStoragePlugin';
import OidcCallbackPlugin from '../auth/OidcCallbackPlugin';
import AccountReturnPlugin from '../auth/AccountReturnPlugin';
import PushPlugin from '../system/PushPlugin';
import GalleryPlugin from '../system/GalleryPlugin';
import SharePlugin from '../system/SharePlugin';
import ConnectivityPlugin from '../system/ConnectivityPlugin';
import ClipboardPlugin from '../system/ClipboardPlugin';
import SystemSettingsPlugin from '../system/SystemSettingsPlugin';

export default class EntryAbility extends FlutterAbility {
  private wasBackgrounded: boolean = false;
  private accountReturn: AccountReturnPlugin = new AccountReturnPlugin();
  private push: PushPlugin | null = null;
  private widgets: ScheduleWidgetPlugin | null = null;
  private pendingLaunchWant: Want | null = null;
  private oidc: OidcCallbackPlugin | null = null;

  onCreate(want: Want, launchParam: AbilityConstant.LaunchParam): void {
    this.pendingLaunchWant = want;
    super.onCreate(want, launchParam);
    this.widgets?.acceptWant(want);
    this.oidc?.acceptWant(want);
    this.push?.acceptWant(want);
    this.accountReturn.acceptWant(want);
  }

  onBackground(): void {
    this.wasBackgrounded = true;
    super.onBackground();
  }

  onForeground(): void {
    super.onForeground();
    if (!this.wasBackgrounded) return;
    this.wasBackgrounded = false;
    const view = this.getFlutterView();
    if (!view?.isAttachedToFlutterEngine()) return;
    // shortcut: Rebind stale GPU buffers until the pinned embedding restores them on resume.
    // Keep ArkUI platform views alive so WebView forms are not reloaded.
    const napi = view.getFlutterEngine()?.getFlutterNapi();
    napi?.xComponentDetachFlutterEngine(view.getId());
    napi?.xComponentAttachFlutterEngine(view.getId());
  }

  onNewWant(want: Want, launchParam: AbilityConstant.LaunchParam): void {
    super.onNewWant(want, launchParam);
    this.widgets?.acceptWant(want);
    this.oidc?.acceptWant(want);
    this.push?.acceptWant(want);
    this.accountReturn.acceptWant(want);
  }

  configureFlutterEngine(flutterEngine: FlutterEngine) {
    super.configureFlutterEngine(flutterEngine)
    GeneratedPluginRegistrant.registerWith(flutterEngine)
    flutterEngine.getPlugins()?.add(new SecureStoragePlugin());
    flutterEngine.getPlugins()?.add(this.accountReturn);
    flutterEngine.getPlugins()?.add(new ConnectivityPlugin());
    flutterEngine.getPlugins()?.add(new ClipboardPlugin(this.context));
    flutterEngine.getPlugins()?.add(new SharePlugin(this.context));
    flutterEngine.getPlugins()?.add(new GalleryPlugin(this.context));
    this.push = new PushPlugin(this);
    flutterEngine.getPlugins()?.add(this.push);
    flutterEngine.getPlugins()?.add(new SystemSettingsPlugin(this.context));
    this.oidc = new OidcCallbackPlugin(this.context);
    flutterEngine.getPlugins()?.add(this.oidc);
    this.widgets = new ScheduleWidgetPlugin(this.context);
    flutterEngine.getPlugins()?.add(this.widgets);
    if (this.pendingLaunchWant) {
      this.widgets.acceptWant(this.pendingLaunchWant);
      this.oidc.acceptWant(this.pendingLaunchWant);
      this.push.acceptWant(this.pendingLaunchWant);
      this.accountReturn.acceptWant(this.pendingLaunchWant);
      this.pendingLaunchWant = null;
    }
  }
}
