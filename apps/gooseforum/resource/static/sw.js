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
    const windows = await self.clients.matchAll({ type: 'window', includeUncontrolled: true });
    if (windows.some((client) => client.focused)) return; // reader is watching the site
    const notificationOptions = {
      body: payload.body,
      icon: payload.icon,
      data: { url: payload.url },
    };
    // 以服务端下发的数字 id 作为独立 tag，新通知不会在系统托盘静默覆盖旧通知；
    // payload 缺 id 时不设 tag，各通知按默认行为分别展示。
    if (payload.id != null) notificationOptions.tag = 'goose-push-' + payload.id;
    await self.registration.showNotification(payload.title, notificationOptions);
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
    const target = payloadUrl || FALLBACK_URL;
    const windows = await self.clients.matchAll({ type: 'window' });
    // 第一遍：已有窗口停留在目标页（同 origin + pathname，忽略 query/hash）时只聚焦。
    for (const client of windows) {
      if (!('focus' in client)) continue;
      try {
        const clientUrl = new URL(client.url);
        const targetUrl = new URL(target, client.url);
        if (clientUrl.origin === targetUrl.origin && clientUrl.pathname === targetUrl.pathname) {
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
