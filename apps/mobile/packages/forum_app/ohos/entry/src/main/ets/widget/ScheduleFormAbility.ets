import { Want } from '@kit.AbilityKit';
import { FormExtensionAbility, formBindingData, formInfo, formProvider } from '@kit.FormKit';
import { BusinessError } from '@kit.BasicServicesKit';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { cardData, widgetPreferences } from './ScheduleFormStore';

export default class ScheduleFormAbility extends FormExtensionAbility {
  onAddForm(want: Want): formBindingData.FormBindingData {
    const id = want.parameters?.[formInfo.FormParam.IDENTITY_KEY] as string;
    const name = want.parameters?.[formInfo.FormParam.NAME_KEY] as string;
    const kind = name === 'TodaySchedule' ? 'today' : name === 'CourseTimeline' ? 'timeline' : 'nextClass';
    const store = widgetPreferences(this.context);
    const ids = store.getSync('form_ids', []) as string[];
    if (!ids.includes(id)) ids.push(id);
    store.putSync('form_ids', ids);
    store.putSync(`form_kind_${id}`, kind);
    return formBindingData.createFormBindingData(cardData(this.context, kind, id));
  }

  async onUpdateForm(id: string): Promise<void> {
    const store = widgetPreferences(this.context);
    const kind = store.getSync(`form_kind_${id}`, 'nextClass') as string;
    try {
      await formProvider.updateForm(id, formBindingData.createFormBindingData(cardData(this.context, kind, id)));
      // Form Kit permits no interval shorter than five minutes.
      await formProvider.setFormNextRefreshTime(id, 5);
    } catch (error) {
      hilog.warn(0x0000, 'ScheduleForm', 'Card refresh failed: %{public}d', (error as BusinessError).code);
    }
  }

  onFormEvent(id: string, message: string): void {
    const store = widgetPreferences(this.context);
    if (store.getSync(`form_kind_${id}`, '') !== 'timeline') return;
    try {
      const event = JSON.parse(message) as Record<string, string>;
      if (event['method'] !== 'toggleDay') return;
      store.putSync(`form_tomorrow_${id}`, !store.getSync(`form_tomorrow_${id}`, false));
      this.onUpdateForm(id);
    } catch (_) {}
  }

  onCastToNormalForm(id: string): void {
    this.onUpdateForm(id);
  }

  onRemoveForm(id: string): void {
    const store = widgetPreferences(this.context);
    const ids = store.getSync('form_ids', []) as string[];
    store.putSync('form_ids', ids.filter(value => value !== id));
    store.deleteSync(`form_kind_${id}`);
    store.deleteSync(`form_tomorrow_${id}`);
  }

  onAcquireFormState(_want: Want): formInfo.FormState {
    return formInfo.FormState.READY;
  }
}
