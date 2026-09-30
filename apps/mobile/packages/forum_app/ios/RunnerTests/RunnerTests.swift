import Flutter
import UIKit
import XCTest
@testable import Runner

class RunnerTests: XCTestCase {

  func testStorageBackupPolicyDoesNotModifySystemOwnedAppGroupRoot() throws {
    let manager = FileManager.default
    let root = manager.temporaryDirectory.appendingPathComponent(UUID().uuidString, isDirectory: true)
    let privateDirectory = root.appendingPathComponent("yourtj_private", isDirectory: true)
    let group = root.appendingPathComponent("AppGroup", isDirectory: true)
    try manager.createDirectory(at: privateDirectory, withIntermediateDirectories: true)
    try manager.createDirectory(at: group, withIntermediateDirectories: true)
    defer { try? manager.removeItem(at: root) }

    var excluded: [URL] = []
    try AppDelegate.excludeStorageFromBackup(privateDirectory: privateDirectory, widgetContainer: group) { url in
      // Real iPhones deny file-write-xattr on the system-owned App Group root.
      // Simulator directory ownership does not reproduce that sandbox boundary.
      guard url != group else { throw CocoaError(.fileWriteNoPermission) }
      excluded.append(url)
    }
    let preferences = group.appendingPathComponent("Library/Preferences", isDirectory: true)
    XCTAssertEqual(excluded, [privateDirectory, preferences])
    XCTAssertTrue(manager.fileExists(atPath: preferences.path))
  }

  func testStorageBackupExclusionPreservesExistingDataAndCanBeRetried() throws {
    let manager = FileManager.default
    let root = manager.temporaryDirectory.appendingPathComponent(UUID().uuidString, isDirectory: true)
    let privateDirectory = root.appendingPathComponent("yourtj_private", isDirectory: true)
    let group = root.appendingPathComponent("AppGroup", isDirectory: true)
    let preferences = group.appendingPathComponent("Library/Preferences", isDirectory: true)
    try manager.createDirectory(at: privateDirectory, withIntermediateDirectories: true)
    try manager.createDirectory(at: preferences, withIntermediateDirectories: true)
    defer { try? manager.removeItem(at: root) }
    let draft = privateDirectory.appendingPathComponent("user_work.sqlite")
    let projection = preferences.appendingPathComponent("group.tj.yourtj.forumApp.widgets.plist")
    try Data("existing encrypted work".utf8).write(to: draft)
    try Data("existing widget projection".utf8).write(to: projection)

    for _ in 0..<2 {
      try AppDelegate.excludeStorageFromBackup(privateDirectory: privateDirectory, widgetContainer: group)
    }
    for directory in [privateDirectory, preferences] {
      XCTAssertEqual(try directory.resourceValues(forKeys: [.isExcludedFromBackupKey]).isExcludedFromBackup, true)
    }
    XCTAssertNotEqual(try group.resourceValues(forKeys: [.isExcludedFromBackupKey]).isExcludedFromBackup, true)
    XCTAssertEqual(try Data(contentsOf: draft), Data("existing encrypted work".utf8))
    XCTAssertEqual(try Data(contentsOf: projection), Data("existing widget projection".utf8))
  }

  func testActualBackupExclusionFailureStillPropagates() throws {
    let root = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString, isDirectory: true)
    for failingCall in 1...2 {
      var calls = 0
      defer { try? FileManager.default.removeItem(at: root) }
      XCTAssertThrowsError(try AppDelegate.excludeStorageFromBackup(
        privateDirectory: root.appendingPathComponent("private"),
        widgetContainer: root.appendingPathComponent("AppGroup")
      ) { _ in
        calls += 1
        if calls == failingCall { throw CocoaError(.fileWriteNoPermission) }
      })
      XCTAssertEqual(calls, failingCall)
    }
  }

  func testPreferencesDirectoryFailurePreservesObstructionAndAllowsRetry() throws {
    let manager = FileManager.default
    let root = manager.temporaryDirectory.appendingPathComponent(UUID().uuidString, isDirectory: true)
    let privateDirectory = root.appendingPathComponent("yourtj_private", isDirectory: true)
    let group = root.appendingPathComponent("AppGroup", isDirectory: true)
    try manager.createDirectory(at: privateDirectory, withIntermediateDirectories: true)
    try manager.createDirectory(at: group, withIntermediateDirectories: true)
    defer { try? manager.removeItem(at: root) }
    let obstruction = group.appendingPathComponent("Library")
    let contents = Data("filesystem error, not permission to delete data".utf8)
    try contents.write(to: obstruction)

    XCTAssertThrowsError(try AppDelegate.excludeStorageFromBackup(privateDirectory: privateDirectory, widgetContainer: group))
    XCTAssertEqual(try Data(contentsOf: obstruction), contents)
    try manager.removeItem(at: obstruction)
    XCTAssertNoThrow(try AppDelegate.excludeStorageFromBackup(privateDirectory: privateDirectory, widgetContainer: group))
  }

}
