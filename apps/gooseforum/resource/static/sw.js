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

const NOTIFICATION_TAG = 'goose-push';
const FALLBACK_URL = '/notifications';

// Show a push notification unless the user is currently looking at a focused
// window of this site. Whether to push while a page is visible is the
// server's semantic decision; the worker only avoids interrupting the reader.
self.addEventListener('push', (event) => {
  let payload = null;
  if (event.data) {
    try {
      // Payload shape: {title, body, url, icon}
      payload = event.data.json();
    } catch {
      payload = null; // malformed payload: treat as absent
    }
  }
  if (!payload || !payload.title) return;

  event.waitUntil((async () => {
    const windows = await self.clients.matchAll({ type: 'window', includeUncontrolled: true });
    if (windows.some((client) => client.focused)) return; // reader is watching the site
    await self.registration.showNotification(payload.title, {
      body: payload.body,
      icon: payload.icon,
      data: { url: payload.url },
      tag: NOTIFICATION_TAG,
    });
  })());
});

// Clicking the notification closes it, focuses an open same-origin window if
// there is one, otherwise opens the payload URL — falling back to the
// notifications page when the payload or its URL is missing.
self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  event.waitUntil((async () => {
    const payloadUrl = event.notification.data && event.notification.data.url;
    const target = payloadUrl || FALLBACK_URL;
    const windows = await self.clients.matchAll({ type: 'window' });
    for (const client of windows) {
      if ('focus' in client) return client.focus();
    }
    return self.clients.openWindow(target);
  })());
});
