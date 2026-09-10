import Flutter
import UIKit
import UserNotifications

@main
@objc class AppDelegate: FlutterAppDelegate, FlutterImplicitEngineDelegate {
  private var pushChannel: FlutterMethodChannel?
  private var tokenResult: FlutterResult?
  private var pendingRoute: String?
  private var deliveryEnabled = false
  private var registrationGeneration = 0

  override func application(
    _ application: UIApplication,
    didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]?
  ) -> Bool {
    UNUserNotificationCenter.current().delegate = self
    return super.application(application, didFinishLaunchingWithOptions: launchOptions)
  }

  func didInitializeImplicitFlutterEngine(_ engineBridge: FlutterImplicitEngineBridge) {
    GeneratedPluginRegistrant.register(with: engineBridge.pluginRegistry)
    guard let registrar = engineBridge.pluginRegistry.registrar(forPlugin: "YourTJPush") else { return }
    let channel = FlutterMethodChannel(name: "yourtj/push", binaryMessenger: registrar.messenger())
    pushChannel = channel
    channel.setMethodCallHandler { [weak self] call, result in
      guard let self else { return }
      let center = UNUserNotificationCenter.current()
      switch call.method {
      case "configured": result(true) // APNs does not require client-side provider secrets.
      case "permission":
        center.getNotificationSettings { settings in
          DispatchQueue.main.async {
            result(settings.authorizationStatus == .authorized || settings.authorizationStatus == .provisional)
          }
        }
      case "requestPermission":
        center.requestAuthorization(options: [.alert, .badge, .sound]) { allowed, error in
          DispatchQueue.main.async {
            if error != nil { result(FlutterError(code: "permission_failed", message: "Unable to request notifications", details: nil)) }
            else { result(allowed) }
          }
        }
      case "register":
        self.tokenResult?(FlutterError(code: "cancelled", message: "Registration replaced", details: nil))
        self.tokenResult = result
        self.deliveryEnabled = true
        self.registrationGeneration += 1
        let generation = self.registrationGeneration
        UIApplication.shared.registerForRemoteNotifications()
        DispatchQueue.main.asyncAfter(deadline: .now() + 20) { [weak self] in
          guard let self, generation == self.registrationGeneration, let callback = self.tokenResult else { return }
          self.tokenResult = nil
          callback(FlutterError(code: "registration_timeout", message: "APNs registration timed out", details: nil))
        }
      case "stop":
        self.deliveryEnabled = false
        self.registrationGeneration += 1
        UIApplication.shared.unregisterForRemoteNotifications()
        center.removeAllDeliveredNotifications()
        self.pendingRoute = nil
        self.tokenResult?(FlutterError(code: "cancelled", message: "Registration cancelled", details: nil))
        self.tokenResult = nil
        result(nil)
      case "initialRoute":
        result(self.pendingRoute)
        self.pendingRoute = nil
      default: result(FlutterMethodNotImplemented)
      }
    }
  }

  override func application(_ application: UIApplication, didRegisterForRemoteNotificationsWithDeviceToken deviceToken: Data) {
    guard deliveryEnabled else { return }
    let token = deviceToken.map { String(format: "%02x", $0) }.joined()
    if let callback = tokenResult {
      tokenResult = nil
      callback(token)
    } else {
      pushChannel?.invokeMethod("token", arguments: token)
    }
  }

  override func application(_ application: UIApplication, didFailToRegisterForRemoteNotificationsWithError error: Error) {
    tokenResult?(FlutterError(code: "registration_failed", message: "APNs registration failed. Check signing and connectivity.", details: nil))
    tokenResult = nil
  }

  override func userNotificationCenter(_ center: UNUserNotificationCenter,
    willPresent notification: UNNotification,
    withCompletionHandler completionHandler: @escaping (UNNotificationPresentationOptions) -> Void) {
    if #available(iOS 14.0, *) {
      completionHandler(deliveryEnabled ? [.banner, .list, .sound, .badge] : [])
    } else {
      completionHandler(deliveryEnabled ? [.alert, .sound, .badge] : [])
    }
  }

  override func userNotificationCenter(_ center: UNUserNotificationCenter,
    didReceive response: UNNotificationResponse,
    withCompletionHandler completionHandler: @escaping () -> Void) {
    if let route = response.notification.request.content.userInfo["route"] as? String {
      // Retain cold-start taps until Dart has restored session + delivery state.
      pendingRoute = route
      if deliveryEnabled {
        pushChannel?.invokeMethod("opened", arguments: route)
      }
    }
    completionHandler()
  }
}
