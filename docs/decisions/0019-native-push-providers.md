# Native push delivery providers

## Status
Accepted
Class: architecture

## Context and Problem Statement

The app must receive system notifications on iOS and on Android phones without Google Play services.
The server owns notification events and device registrations. A successful store upload does not
prove notification authorization, device registration or delivery. The native transport must match
its device token; an FCM token cannot be sent directly to APNs.

## Decision Drivers

- Support iOS through Apple's system service without a Firebase dependency.
- Support Android OEM offline delivery when the app process is absent.
- Preserve existing device registrations and the shared notification outbox.
- Keep provider credentials out of clients and make unavailable states visible.

## Considered Options

- Direct APNs on iOS and JPush aggregation of Android OEM channels.
- Firebase Cloud Messaging for both platforms.
- Direct APNs and separate integrations with each Android manufacturer.

## Decision Outcome

Use direct APNs on iOS and JPush on Android. The native method channel provides authorization,
registration, stop and notification-open handling. Android initializes the SDK only after explicit
push consent; optional analytics, geolocation and dynamic loading are disabled. Device registration
includes a provider; omitted values preserve legacy iOS APNs / Android FCM routing. The backend
retains FCM for existing registrations. No automatic migration of tokens between providers occurs.

JPush's AppKey and OEM client identifiers enter Android builds through the signing environment.
The Master Secret remains in the production server environment. OEM credentials also require
configuration in the provider console; SDK integration alone does not establish offline delivery.
The app discloses provider processing before consent. Store privacy disclosures must match it.

Delivery is `Partial` until provider accounts, signed builds and physical-device delivery are verified.
System force-stop, vendor restrictions and external provider availability remain outside the app's
control. No always-running foreground service or forced battery exemption is introduced.

## Pros and Cons of the Options

- Direct APNs + JPush: matches the target devices and centralizes OEM delivery; adds a third-party
  processor and still requires vendor enrollment, credentials and per-vendor validation.
- Firebase alone: simpler unified tooling, but cannot satisfy delivery on Android without Google
  services; also adds an unnecessary iOS dependency for the existing direct APNs backend.
- Separate OEM integrations: avoids the aggregation provider but duplicates SDK, server API,
  credential rotation and receipt handling across manufacturers.

## Links

- [Apple notification authorization](https://developer.apple.com/documentation/usernotifications/asking-permission-to-use-notifications)
- [JPush OEM integration](https://docs.jiguang.cn/jpush/client/Android/android_3rd_guide)
- [JPush privacy policy](https://www.jiguang.cn/license/privacy)
- [Native platform direction](0010-mobile-route-a-native-alignment.md)
- [Mobile release operations](../operations/mobile-releases.md)
