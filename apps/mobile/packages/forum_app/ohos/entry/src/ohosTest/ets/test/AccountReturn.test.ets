import { describe, it, expect } from '@ohos/hypium';
import { Any, EventSink } from '@ohos/flutter_ohos';
import AccountReturnPlugin from '../../../main/ets/auth/AccountReturnPlugin';

class ReturnEvents implements EventSink {
  values: string[] = [];
  success(event: Any): void { this.values.push(event as string); }
  error(_code: string, _message: string, _details: Any): void { throw new Error('Unexpected account return error'); }
  endOfStream(): void {}
}

export default function accountReturnTest(): void {
  describe('AccountReturn', () => {
    it('acceptsOnlyTheExactReturnAndConsumesColdLaunchOnce', 0, () => {
      const plugin = new AccountReturnPlugin();
      const events = new ReturnEvents();
      for (const uri of [
        'yourtj://account/email-verified?token=synthetic',
        'yourtj://account/email-verified#fragment',
        'other://account/email-verified', 'yourtj://account/other'
      ]) plugin.acceptWant({ uri: uri });
      plugin.onListen(null, events);
      expect(events.values.length).assertEqual(0);
      plugin.onCancel(null);
      plugin.acceptWant({ uri: 'yourtj://account/email-verified' });
      plugin.onListen(null, events);
      expect(events.values.length).assertEqual(1);
      plugin.onCancel(null);
      plugin.onListen(null, events);
      expect(events.values.length).assertEqual(1);
      plugin.acceptWant({ uri: 'yourtj://account/email-verified' });
      expect(events.values.length).assertEqual(2);
    });
  });
}
