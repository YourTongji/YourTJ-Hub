// Pure-push Service Worker for yourtj.
//
// Registration contract (owned by the registering page, not this file):
// - Register with `updateViaCache: 'none'` so the worker script never comes
//   from the HTTP cache after the first install.
// - This file's URL is stable: do not rename or move it, or clients that
//   registered the old URL keep running the previously cached script.
//
// No fetch handler: no request is intercepted or cached. No skipWaiting /
// clients.claim: pure push needs neither.

const FALLBACK_URL = '/notifications';
// Share source IDs and a bounded durable ledger with page notifications.
// The timestamp preserves compatibility with pages running the previous worker.
const DEDUP_CHANNEL_NAME = 'goose:browser-notifications-dedup';
// Client id → peer id of the conversation that client currently shows.
const activeChats = new Map();
self.addEventListener('message', event => {
  if (event.source && event.data && event.data.type === 'goose:active-chat') {
    activeChats.set(event.source.id, Number(event.data.peerId) || 0);
  }
});
async function currentRecipientMatches(recipientId) {
  if (!Number.isSafeInteger(recipientId) || recipientId <= 0) return false;
  try {
    const response = await fetch('/messages', {headers: {'X-Goose-Page': 'true', 'X-Goose-Prefetch': '1', Accept: 'application/json'}, credentials: 'same-origin', cache: 'no-store', signal: AbortSignal.timeout(5000)});
    if (!response.ok) return false;
    const page = await response.json();
    return page.layout && page.layout.viewer && page.layout.viewer.isAuthenticated && page.layout.viewer.id === recipientId;
  } catch { return false; }
}


// Show a push notification unless the user is currently looking at a focused
// window of this site. Whether to push while a page is visible is the
// server's semantic decision; the worker only avoids interrupting the reader.
self.addEventListener('push', (event) => {
  let payload = null;
  if (event.data) {
    try {
      // Payload shape: {title, body, url, icon, id (numeric, optional)}
      payload = event.data.json();
    } catch {
      payload = null; // malformed payload: treat as absent
    }
  }
  if (!payload || !payload.title) return;

  event.waitUntil((async () => {
    if ((payload.recipientId != null || payload.sourceKind === 'chat_message') && !await currentRecipientMatches(payload.recipientId)) return;
    const windows = await self.clients.matchAll({ type: 'window', includeUncontrolled: true });
    if (windows.some(client => {
      if (!client.focused) return false;
      if (payload.sourceKind !== 'chat_message') return true;
      try {
        const url = new URL(client.url);
        const peer = Number(new URL(payload.url, client.url).searchParams.get('userId'));
        return url.pathname === '/messages' && activeChats.get(client.id) === peer;
      } catch { return false; }
    })) return;
    const source = payload.id != null ? `${payload.sourceKind || 'notification'}:${payload.id}` : '';
    const notificationOptions = {
      body: payload.body,
      icon: payload.icon,
      data: { url: payload.url, recipientId: payload.recipientId, sourceKind: payload.sourceKind },
    };
    // 以服务端下发的数字 id 作为独立 tag，新通知不会在系统托盘静默覆盖旧通知；
    // payload 缺 id 时不设 tag，各通知按默认行为分别展示。
    if (source) notificationOptions.tag = 'goose-push-' + source;
    const show = async () => {
      let cache;
      const key = new URL(`/__notification-source/${source}`, self.location.origin).href;
      try {
        if (source && typeof caches !== 'undefined') {
          cache = await caches.open('goose-notification-sources-v1');
          if (await cache.match(key)) return false;
        }
      } catch { cache = undefined; }
      await self.registration.showNotification(payload.title, notificationOptions);
      try { if (cache) {
        await cache.put(key, new Response('1'));
        // The ledger only needs to cover the replay window; 256 newest entries
        // are enough for the polling interval. A longer window needs expiry.
        const keys = await cache.keys();
        for (const old of keys.slice(0, Math.max(0, keys.length - 256))) await cache.delete(old);
      } } catch {}
      return true;
    };
    let shown;
    try { shown = source && self.navigator && self.navigator.locks
      ? await self.navigator.locks.request(`goose-notification:${source}`, show) : await show(); }
    catch { shown = await show(); }
    if (!shown) return;
    // 仅在真正弹出后广播：前台聚焦的窗口会提前 return，不广播（SW 未弹出时
    // 轮询通道应正常兜底）。页面端 browser-notification.ts 模块加载即监听
    // 同一频道，收到 {at} 后在去重窗口内不再重复弹出。
    if (typeof BroadcastChannel !== 'undefined') {
      try {
        const dedupChannel = new BroadcastChannel(DEDUP_CHANNEL_NAME);
        dedupChannel.postMessage({ at: Date.now(), source });
        dedupChannel.close();
      } catch {
        // 广播失败只影响去重：轮询通道可能重复弹一次，可接受。
      }
    }
  })());
});

// Clicking the notification closes it, then brings up the payload URL:
// a window already on that path is only focused, any other window is
// navigated to the deep link before focusing (focusing without navigating
// would leave the user on an unrelated page), and with no usable window the
// payload URL is opened — falling back to the notifications page when the
// payload or its URL is missing.
self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  event.waitUntil((async () => {
    const payloadUrl = event.notification.data && event.notification.data.url;
    let target = FALLBACK_URL;
    try {
      const parsed = new URL(payloadUrl || FALLBACK_URL, self.location.origin);
      if (parsed.origin === self.location.origin) target = parsed.pathname + parsed.search + parsed.hash;
    } catch {}
    if (event.notification.data && (event.notification.data.recipientId != null || event.notification.data.sourceKind === 'chat_message') && !await currentRecipientMatches(event.notification.data.recipientId)) target = event.notification.data.sourceKind === 'chat_message' ? '/messages' : FALLBACK_URL;
    const windows = await self.clients.matchAll({ type: 'window' });
    // Only focus an existing window when its full target, including the conversation/floor, matches.
    for (const client of windows) {
      if (!('focus' in client)) continue;
      try {
        const clientUrl = new URL(client.url);
        const targetUrl = new URL(target, client.url);
        if (clientUrl.origin === targetUrl.origin && clientUrl.pathname === targetUrl.pathname && clientUrl.search === targetUrl.search && clientUrl.hash === targetUrl.hash) {
          return client.focus();
        }
      } catch {
        // URL 无法解析（如 about:blank）：跳过该窗口。
      }
    }
    // 第二遍：把某个可导航窗口带到深链再聚焦；navigate 不可用或抛错
    // （cross-origin/about:blank 等）时尝试下一个窗口。
    for (const client of windows) {
      if (!('focus' in client && 'navigate' in client)) continue;
      try {
        await client.navigate(target);
        return client.focus();
      } catch {
        // navigate 失败：尝试下一个窗口。
      }
    }
    // 无可用窗口：新开窗口打开目标。
    return self.clients.openWindow(target);
  })());
});
