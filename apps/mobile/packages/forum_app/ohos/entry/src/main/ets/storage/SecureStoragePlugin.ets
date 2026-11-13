import {
  FlutterPlugin, FlutterPluginBinding, MethodCall, MethodCallHandler, MethodChannel, MethodResult
} from '@ohos/flutter_ohos';
import { SecureStore } from './SecureStore';

export default class SecureStoragePlugin implements FlutterPlugin, MethodCallHandler {
  private channel: MethodChannel | null = null;
  private store: SecureStore | null = null;
  private tail: Promise<void> = Promise.resolve();

  getUniqueClassName(): string { return 'YourTJSecureStorage'; }

  onAttachedToEngine(binding: FlutterPluginBinding): void {
    this.store = new SecureStore(binding.getApplicationContext());
    this.channel = new MethodChannel(binding.getBinaryMessenger(), 'yourtj/secure_storage');
    this.channel.setMethodCallHandler(this);
  }

  onDetachedFromEngine(_binding: FlutterPluginBinding): void {
    this.channel?.setMethodCallHandler(null);
    this.channel = null;
  }

  onMethodCall(call: MethodCall, result: MethodResult): void {
    const operation = this.tail.then(() => this.handle(call, result));
    this.tail = operation.catch(() => {
      result.error('secure_storage_failed', 'Secure storage operation failed', null);
    });
  }

  private async handle(call: MethodCall, result: MethodResult): Promise<void> {
    if (!this.store) throw new Error('Secure storage unavailable');
    if (call.method === 'readAll') {
      result.success(await this.store.readAll());
      return;
    }
    if (call.method === 'deleteAll') {
      await this.store.deleteAll();
      result.success(null);
      return;
    }
    const args = call.args as Map<string, string>;
    const key = args.get('key');
    if (typeof key !== 'string' || key.length === 0 || key.length > 512) {
      throw new Error('Invalid secure storage key');
    }
    switch (call.method) {
      case 'read': result.success(await this.store.read(key)); return;
      case 'containsKey': result.success(this.store.containsKey(key)); return;
      case 'delete':
        await this.store.delete(key);
        result.success(null);
        return;
      case 'write': {
        const value = args.get('value');
        if (typeof value !== 'string' || value.length > 1024 * 1024) throw new Error('Invalid secure storage value');
        await this.store.write(key, value);
        result.success(null);
        return;
      }
      default: result.notImplemented();
    }
  }
}
