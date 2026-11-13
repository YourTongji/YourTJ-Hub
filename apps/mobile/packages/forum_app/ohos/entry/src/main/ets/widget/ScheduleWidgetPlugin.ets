import {
  FlutterPlugin, FlutterPluginBinding, MethodCall, MethodCallHandler,
  MethodChannel, MethodResult, EventChannel, EventSink, Any
} from '@ohos/flutter_ohos';
import { common, Want } from '@kit.AbilityKit';
import { preferences } from '@kit.ArkData';
import { formProvider } from '@kit.FormKit';
import { BusinessError } from '@kit.BasicServicesKit';
import { refreshCards, widgetPreferences } from './ScheduleFormStore';

export default class ScheduleWidgetPlugin implements FlutterPlugin, MethodCallHandler {
  private channel: MethodChannel | null = null;
  private events: EventChannel | null = null;
  private sink: EventSink | null = null;
  private initialLink: string | null = null;
  private context: common.Context;

  constructor(context: common.Context) {
    this.context = context;
  }

  getUniqueClassName(): string { return 'YourTJScheduleWidget'; }

  onAttachedToEngine(binding: FlutterPluginBinding): void {
    this.channel = new MethodChannel(binding.getBinaryMessenger(), 'home_widget');
    this.channel.setMethodCallHandler(this);
    this.events = new EventChannel(binding.getBinaryMessenger(), 'home_widget/updates');
    this.events.setStreamHandler(this);
  }

  onDetachedFromEngine(_binding: FlutterPluginBinding): void {
    this.channel?.setMethodCallHandler(null);
    this.events?.setStreamHandler(null);
    this.sink = null;
  }

  onListen(_args: Any, events: EventSink): void { this.sink = events; }
  onCancel(_args: Any): void { this.sink = null; }

  acceptWant(want: Want): void {
    const link = want.parameters?.['widgetUri'];
    if (link !== 'yourtj://campus/today' && link !== 'yourtj://campus/timetable') return;
    if (this.sink) this.sink.success(link);
    else this.initialLink = link as string;
  }

  async onMethodCall(call: MethodCall, result: MethodResult): Promise<void> {
    try {
      const store = widgetPreferences(this.context);
      switch (call.method) {
        case 'setAppGroupId':
          result.success(true);
          return;
        case 'saveWidgetData': {
          const args = call.args as Map<string, preferences.ValueType | null>;
          const id = args.get('id') as string;
          if (typeof id !== 'string' || !id.startsWith('schedule_widget_') || id.length > 128) {
            result.error('invalid_key', 'Invalid widget storage key', null);
            return;
          }
          const value = args.get('data');
          if (value === null || value === undefined) store.deleteSync(id);
          else store.putSync(id, value);
          result.success(true);
          return;
        }
        case 'getWidgetData': {
          const args = call.args as Map<string, preferences.ValueType | null>;
          const id = args.get('id') as string;
          const values = store.getAllSync() as Record<string, preferences.ValueType>;
          result.success(store.hasSync(id) ? values[id] : args.get('defaultValue') ?? null);
          return;
        }
        case 'updateWidget':
          await refreshCards(this.context);
          result.success(true);
          return;
        case 'isRequestPinWidgetSupported':
          result.success(true);
          return;
        case 'requestPinWidget': {
          const args = call.args as Map<string, string>;
          const receiver = args.get('qualifiedAndroidName') ?? '';
          const name = receiver.endsWith('TodayScheduleWidgetReceiver') ? 'TodaySchedule' :
            receiver.endsWith('CourseTimelineWidgetReceiver') ? 'CourseTimeline' : 'NextClass';
          await formProvider.openFormManager({
            bundleName: this.context.applicationInfo.name,
            abilityName: 'ScheduleFormAbility',
            parameters: {
              'ohos.extra.param.key.form_dimension': name === 'NextClass' ? 2 : name === 'TodaySchedule' ? 3 : 4,
              'ohos.extra.param.key.form_name': name,
              'ohos.extra.param.key.module_name': 'entry'
            }
          });
          result.success(null);
          return;
        }
        case 'initiallyLaunchedFromHomeWidget':
          result.success(this.initialLink);
          this.initialLink = null;
          return;
        case 'scheduleWidgetUpdates':
        case 'cancelScheduledWidgetUpdates':
        case 'canScheduleExactWidgetUpdates':
        case 'updateWidgetPreview':
          // Form Kit owns background refresh and gallery previews.
          result.success(false);
          return;
        default:
          result.notImplemented();
      }
    } catch (error) {
      result.error('widget_error', `Widget operation failed (${(error as BusinessError).code})`, null);
    }
  }
}
