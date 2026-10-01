import Flutter
import UIKit
import UserNotifications
import Darwin

@main
@objc class AppDelegate: FlutterAppDelegate, FlutterImplicitEngineDelegate {
  private var appleAuth: YourTJAppleAuth?
  private var pushChannel: FlutterMethodChannel?
  private var tokenResult: FlutterResult?
  private var pendingRoute: String?
  private var deliveryEnabled = false
  private var registrationGeneration = 0
  private var startupLaunchKind = "cold"
  private var hasBecomeActiveOnce = false
  private var accessibilityObserver: NSObjectProtocol?

  override func application(
    _ application: UIApplication,
    didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]?
  ) -> Bool {
    UNUserNotificationCenter.current().delegate = self
    return super.application(application, didFinishLaunchingWithOptions: launchOptions)
  }

  func didInitializeImplicitFlutterEngine(_ engineBridge: FlutterImplicitEngineBridge) {
    GeneratedPluginRegistrant.register(with: engineBridge.pluginRegistry)
    if let registrar = engineBridge.pluginRegistry.registrar(forPlugin: "YourTJAccessibility") {
      let channel = FlutterMethodChannel(name: "yourtj/accessibility", binaryMessenger: registrar.messenger())
      channel.setMethodCallHandler { call, result in
        if call.method == "reduceTransparency" {
          result(UIAccessibility.isReduceTransparencyEnabled)
        } else {
          result(FlutterMethodNotImplemented)
        }
      }
      if let observer = accessibilityObserver { NotificationCenter.default.removeObserver(observer) }
      accessibilityObserver = NotificationCenter.default.addObserver(
        forName: UIAccessibility.reduceTransparencyStatusDidChangeNotification,
        object: nil, queue: .main
      ) { _ in
        channel.invokeMethod("reduceTransparencyChanged", arguments: UIAccessibility.isReduceTransparencyEnabled)
      }
    }
    if let appleRegistrar = engineBridge.pluginRegistry.registrar(forPlugin: "YourTJAppleAuth") {
      appleAuth = YourTJAppleAuth(registrar: appleRegistrar)
    }
    if let storageRegistrar = engineBridge.pluginRegistry.registrar(forPlugin: "YourTJStorage") {
      let storage = FlutterMethodChannel(name: "yourtj/storage", binaryMessenger: storageRegistrar.messenger())
      storage.setMethodCallHandler { call, result in
        guard call.method == "excludeFromBackup",
              let arguments = call.arguments as? [String: Any],
              let path = arguments["path"] as? String else {
          result(FlutterMethodNotImplemented)
          return
        }
        let url = URL(fileURLWithPath: path).standardizedFileURL
        let support = FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask)[0].standardizedFileURL
        guard url.path == support.appendingPathComponent("yourtj_private").path else {
          result(FlutterError(code: "invalid_path", message: "Unsupported storage path", details: nil))
          return
        }
        do {
          guard let widgetDirectory = FileManager.default.containerURL(
            forSecurityApplicationGroupIdentifier: "group.tj.yourtj.forumApp.widgets"
          ) else {
            throw NSError(domain: "YourTJStorage", code: 1)
          }
          try Self.excludeStorageFromBackup(privateDirectory: url, widgetContainer: widgetDirectory)
          result(nil)
        } catch {
          result(FlutterError(code: "backup_exclusion_failed", message: "Unable to configure local storage", details: nil))
        }
      }
    }
    if let startupRegistrar = engineBridge.pluginRegistry.registrar(forPlugin: "YourTJStartup") {
      let startupChannel = FlutterMethodChannel(
        name: "yourtj/startup",
        binaryMessenger: startupRegistrar.messenger()
      )
      startupChannel.setMethodCallHandler { [weak self] call, result in
        guard let self else { return }
        switch call.method {
        case "getDeviceProfile":
          result([
            "device": Self.deviceModelIdentifier,
            "osVersion": "iOS \(UIDevice.current.systemVersion)",
            "refreshRateHz": UIScreen.main.maximumFramesPerSecond,
            "launchKind": self.startupLaunchKind
          ])
        case "reportFullyDrawn":
          result(false)
        default:
          result(FlutterMethodNotImplemented)
        }
      }
    }
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

  static func excludeStorageFromBackup(
    privateDirectory: URL,
    widgetContainer: URL,
    exclude: (URL) throws -> Void = { url in
      var url = url
      var values = URLResourceValues()
      values.isExcludedFromBackup = true
      try url.setResourceValues(values)
    }
  ) throws {
    try exclude(privateDirectory)
    // iOS owns the App Group root and denies file-write-xattr there on devices.
    // HomeWidget stores its projection in this app-owned preferences directory;
    // excluding the directory also covers future atomic UserDefaults rewrites.
    let preferences = widgetContainer.appendingPathComponent("Library/Preferences", isDirectory: true)
    try FileManager.default.createDirectory(at: preferences, withIntermediateDirectories: true)
    try exclude(preferences)
  }

  override func applicationDidBecomeActive(_ application: UIApplication) {
    if hasBecomeActiveOnce {
      startupLaunchKind = "hot"
    } else {
      hasBecomeActiveOnce = true
    }
    super.applicationDidBecomeActive(application)
  }

  private static var deviceModelIdentifier: String {
    var systemInfo = utsname()
    uname(&systemInfo)
    let capacity = MemoryLayout.size(ofValue: systemInfo.machine)
    return withUnsafePointer(to: &systemInfo.machine) {
      $0.withMemoryRebound(to: CChar.self, capacity: capacity) {
        String(cString: $0)
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
