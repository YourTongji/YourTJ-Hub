"""Release authority is external evidence, never a flag supplied by a candidate."""
import copy
import json
import re
import tempfile
import unittest
from pathlib import Path

from model import (ReleaseError, validate_candidate, validate_approval, next_identity, channels_for, render_changelog,
                   validate_changelog, validate_note_text)


SHA = "a" * 40


def candidate():
    return {
        "schemaVersion": 1, "candidateId": "mobile-1.0.15-15", "sourceSha": SHA,
        "product": "mobile", "version": "1.0.15", "tag": "mobile-v1.0.15", "buildNumber": 15,
        "channels": ["android", "ios-testflight"], "operation": "release", "existingRelease": None,
        "baselines": {"android": {"tag": "mobile-v1.0.14", "sourceSha": "b" * 40},
                      "ios-testflight": {"tag": "mobile-v1.0.13", "sourceSha": "c" * 40}},
        "notes": {"android": "android.zh-CN.md", "ios-testflight": "testflight.en-US.txt"},
        "serverRequirement": None, "requiredDisclosures": [],
    }


class CandidateTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.path = Path(self.temp.name)
        (self.path / "android.zh-CN.md").write_text("改善 Android 聊天图片显示。\n", encoding='utf-8')
        (self.path / "testflight.en-US.txt").write_text("Test iOS widgets after changing courses.\n", encoding='utf-8')

    def test_platform_files_and_identity_are_bound(self):
        manifest = candidate()
        original = validate_candidate(manifest, self.path)
        (self.path / "android.zh-CN.md").write_text("改善 Android 搜索。\n", encoding='utf-8')
        self.assertNotEqual(original, validate_candidate(manifest, self.path))

    def test_rejects_self_approval_unknown_channels_and_cross_platform_file(self):
        for change in [{"approved": True}, {"channels": ["android", "unknown"]},
                       {"notes": {"android": "testflight.en-US.txt", "ios-testflight": "testflight.en-US.txt"}},
                       {"sourceSha": "main"}, {"tag": "v1.0.15"}]:
            with self.subTest(change=change), self.assertRaises(ReleaseError):
                validate_candidate(candidate() | change, self.path)

    def test_symlink_and_empty_draft_cannot_publish(self):
        note = self.path / "android.zh-CN.md"
        note.write_text("[DRAFT: human review required]\n", encoding='utf-8')
        with self.assertRaises(ReleaseError):
            validate_candidate(candidate(), self.path)
        note.unlink()
        try:
            note.symlink_to(self.path / "testflight.en-US.txt")
        except OSError:
            self.skipTest("Symlink creation is unavailable on this host")
        with self.assertRaises(ReleaseError):
            validate_candidate(candidate(), self.path)

    def test_store_is_plain_text_bounded_and_disclosures_required(self):
        manifest = candidate() | {"channels": ["ios-app-store"], "notes": {"ios-app-store": "ios.zh-Hans.txt"},
                                  "baselines": {"ios-app-store": {"tag": None, "sourceSha": None}}}
        note = self.path / "ios.zh-Hans.txt"
        for text in ["x" * 4001, "[click](https://example.org)", "# Heading"]:
            note.write_text(text, encoding='utf-8')
            with self.assertRaises(ReleaseError):
                validate_candidate(manifest, self.path)
        note.write_text("改进 iOS 小组件。", encoding='utf-8')
        manifest["requiredDisclosures"] = [{"id": "analytics", "channels": ["ios-app-store"], "text": "自动统计"}]
        with self.assertRaises(ReleaseError):
            validate_candidate(manifest, self.path)
        note.write_text("改进 iOS 小组件。自动统计", encoding='utf-8')
        validate_candidate(manifest, self.path)

    def test_explicit_scope(self):
        self.assertEqual(channels_for("android", "testflight"), ["android"])
        self.assertEqual(channels_for("mobile", "testflight"), ["android", "ios-testflight"])
        with self.assertRaises(ReleaseError):
            channels_for("android", "app-store")

    def test_version_namespaces_and_build_high_water(self):
        tags = [{"tag": "mobile-v1.2.3", "buildNumber": 80}, {"tag": "v9.9.9", "buildNumber": 999}]
        self.assertEqual(next_identity("mobile", "patch", tags), ("1.2.4", 81))
        self.assertEqual(next_identity("web", "minor", tags), ("9.10.0", None))
        with self.assertRaises(ReleaseError):
            next_identity("mobile", "patch", [{"tag": "mobile-v1.2.3", "buildNumber": None}])

    def test_schema_two_freezes_reviewed_facts_and_derives_each_platform_note(self):
        manifest = candidate() | {"schemaVersion": 2}
        android = {"id": "android-search", "title": "搜索更顺手", "summary": "改进搜索结果排序。",
                   "platforms": ["android"], "kind": "improvement"}
        ios = {"id": "ios-widget", "title": "课程小组件", "summary": "优化课程小组件显示。",
               "platforms": ["ios-testflight"], "kind": "improvement"}
        changelog = {"schemaVersion": 1, "version": manifest["version"], "buildNumber": manifest["buildNumber"],
                     "highlights": [android, ios], "breaking": [], "requiredActions": [],
                     "evidence": {"android-search": ["android-proof"], "ios-widget": ["ios-proof"]},
                     "testflightNotes": [{"id": "ios-widget", "text": "Please verify the course widget.",
                                          "evidenceIds": ["ios-proof"]}]}
        (self.path / "evidence.json").write_text(json.dumps({"schemaVersion": 1, "sourceSha": SHA, "evidence": [
            {"id": "android-proof", "channels": ["android"]},
            {"id": "ios-proof", "channels": ["ios-testflight"]}]}), encoding="utf-8")
        (self.path / "changelog.json").write_text(json.dumps(changelog, ensure_ascii=False), encoding="utf-8")
        (self.path / "android.zh-CN.md").write_text(render_changelog(changelog, "android"), encoding="utf-8")
        (self.path / "testflight.en-US.txt").write_text(render_changelog(changelog, "ios-testflight"), encoding="utf-8")
        digest = validate_candidate(manifest, self.path)
        self.assertTrue(digest)
        self.assertIn("搜索更顺手", (self.path / "android.zh-CN.md").read_text(encoding="utf-8"))
        self.assertEqual((self.path / "testflight.en-US.txt").read_text(encoding="utf-8"), "Please verify the course widget.")

        for invalid in [
            changelog | {"highlights": [android | {"platforms": ["android", "ios"]}]},
            changelog | {"evidence": {"android-search": ["missing"], "ios-widget": ["ios-proof"]}},
            changelog | {"testflightNotes": [{"id": "unrelated", "text": "Verify this.", "evidenceIds": ["ios-proof"]}]},
            changelog | {"highlights": [android, ios | {"platforms": ["android"]}]},
        ]:
            (self.path / "changelog.json").write_text(json.dumps(invalid, ensure_ascii=False), encoding="utf-8")
            with self.subTest(invalid=invalid), self.assertRaises(ReleaseError):
                validate_candidate(manifest, self.path)

    def test_each_changelog_group_fits_the_client_limit(self):
        def changelog(count, group="highlights"):
            value = {"schemaVersion": 1, "version": "1.0.15", "buildNumber": 15, "highlights": [], "breaking": [],
                     "requiredActions": [], "evidence": {}, "testflightNotes": []}
            for i in range(count):
                if group == "testflightNotes":
                    value[group].append({"id": f"t-{i}", "text": "Check it.", "evidenceIds": ["proof"]})
                else:
                    value[group].append({"id": f"n-{i}", "title": "Fix", "summary": "Fixed.",
                                         "platforms": ["android"], "kind": "fix"})
                value["evidence"][value[group][-1]["id"]] = ["proof"]
            return value
        for group in ("highlights", "breaking", "requiredActions", "testflightNotes"):
            with self.subTest(group=group):
                validate_changelog(changelog(100, group), "1.0.15", 15, ["android", "ios-testflight"], draft=True)
                with self.assertRaisesRegex(ReleaseError, "at most 100"):
                    validate_changelog(changelog(101, group), "1.0.15", 15, ["android", "ios-testflight"], draft=True)

    def test_each_user_facing_channel_needs_structured_notes_before_publishing(self):
        manifest = candidate() | {"schemaVersion": 2, "channels": ["android", "ios-app-store"],
                                  "notes": {"android": "android.zh-CN.md", "ios-app-store": "ios.zh-Hans.txt"},
                                  "baselines": {c: {"tag": "mobile-v1.0.14", "sourceSha": "b" * 40}
                                                for c in ("android", "ios-app-store")}}
        android = {"id": "android-search", "title": "搜索更顺手", "summary": "改进搜索结果排序。",
                   "platforms": ["android"], "kind": "improvement"}
        changelog = {"schemaVersion": 1, "version": manifest["version"], "buildNumber": manifest["buildNumber"],
                     "highlights": [android], "breaking": [], "requiredActions": [],
                     "evidence": {"android-search": ["android-proof"]}, "testflightNotes": []}
        (self.path / "evidence.json").write_text(json.dumps({"schemaVersion": 1, "sourceSha": SHA, "evidence": [
            {"id": "android-proof", "channels": ["android"]}]}), encoding="utf-8")
        (self.path / "changelog.json").write_text(json.dumps(changelog, ensure_ascii=False), encoding="utf-8")
        (self.path / "android.zh-CN.md").write_text(render_changelog(changelog, "android"), encoding="utf-8")
        (self.path / "ios.zh-Hans.txt").write_text("错误修复和性能改进。", encoding="utf-8")
        with self.assertRaisesRegex(ReleaseError, "ios-app-store has no reviewed changelog entry"):
            validate_candidate(manifest, self.path)
        validate_candidate(manifest, self.path, draft=True)


class ApprovalTests(unittest.TestCase):
    def setUp(self):
        self.pr = {"head": {"sha": SHA, "ref": "codex/release/mobile-1.0.15-15"},
                   "base": {"ref": "main"}, "merged": True, "merge_commit_sha": "d" * 40}
        self.review = {"user": {"login": "maintainer", "type": "User"}, "state": "APPROVED",
                       "commit_id": SHA, "id": 1, "submitted_at": "2026-10-01T00:00:00Z"}
        self.files = ["releases/requests/mobile-1.0.15-15/manifest.json"]

    def test_final_head_human_review_and_candidate_only_changes(self):
        result = validate_approval(self.pr, [self.review], self.files, "mobile-1.0.15-15", {"maintainer"})
        self.assertEqual(result["approvedHead"], SHA)

    def test_bot_stale_revoked_ineligible_or_direct_push_rejected(self):
        for reviews in [[], [self.review | {"commit_id": "b" * 40}],
                        [self.review | {"user": {"login": "maintainer", "type": "Bot"}}],
                        [self.review, self.review | {"state": "DISMISSED", "id": 2}],
                        [self.review | {"user": {"login": "stranger", "type": "User"}}]]:
            with self.subTest(reviews=reviews), self.assertRaises(ReleaseError):
                validate_approval(self.pr, reviews, self.files, "mobile-1.0.15-15", {"maintainer"})
        with self.assertRaises(ReleaseError):
            validate_approval(self.pr, [self.review], self.files + [".github/workflows/release-publish.yml"],
                              "mobile-1.0.15-15", {"maintainer"})

    def test_changes_requested_by_another_maintainer_blocks(self):
        other = self.review | {"user": {"login": "other", "type": "User"}, "state": "CHANGES_REQUESTED", "id": 2}
        with self.assertRaises(ReleaseError):
            validate_approval(self.pr, [self.review, other], self.files, "mobile-1.0.15-15", {"maintainer", "other"})


class StoreChannelIsolationTests(unittest.TestCase):
    """Store What's New files name only their own mobile platform."""

    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.path = Path(self.temp.name)

    def test_ios_note_rejects_the_1311_android_sentence(self):
        leak = "打开账户抽屉情况下，Android 返回操作优先关闭账户抽屉，保留当前页面。"
        manifest = candidate() | {"channels": ["ios-app-store"], "notes": {"ios-app-store": "ios.zh-Hans.txt"},
                                  "baselines": {"ios-app-store": {"tag": None, "sourceSha": None}}}
        (self.path / "ios.zh-Hans.txt").write_text(leak + "\n", encoding="utf-8")
        with self.assertRaisesRegex(ReleaseError, r"ios-app-store note ios\.zh-Hans\.txt mentions Android"):
            validate_candidate(manifest, self.path)
        with self.assertRaisesRegex(ReleaseError, r"ios-testflight note testflight\.en-US\.txt mentions Android"):
            validate_note_text("Confirm history shows no Android entries.", "testflight.en-US.txt", "ios-testflight")

    def test_opposite_tokens_fail_and_same_platform_words_pass(self):
        forbidden = {
            "ios-app-store": ("ios.zh-Hans.txt", [("请改用 Android 版。", "Android"), ("Android返回优先关闭抽屉。", "Android"),
                                                 ("安装安卓客户端。", "安卓"), ("另行下载 APK。", "APK"), ("下载 apk。", "APK")]),
            "ios-testflight": ("testflight.en-US.txt", [("Install the APK.", "APK"), ("Check the 安卓 build.", "安卓")]),
            "android": ("android.zh-CN.md", [("与 iOS 行为一致。", "iOS"), ("在 iPhone 上打开。", "iPhone"),
                                            ("支持 iPad 分屏。", "iPad"), ("需要 iPadOS 17。", "iPadOS"),
                                            ("前往 App Store 更新。", "App Store"), ("使用 TestFlight 验证。", "TestFlight"),
                                            ("在苹果设备上可用。", "苹果")]),
        }
        for channel, (name, samples) in forbidden.items():
            for sample, token in samples:
                with self.subTest(channel=channel, sample=sample):
                    with self.assertRaisesRegex(ReleaseError,
                                                 rf"{re.escape(channel)} note {re.escape(name)} mentions {re.escape(token)}"):
                        validate_note_text(sample, name, channel)
        allowed = {
            "ios-app-store": ("ios.zh-Hans.txt", "需要 iOS 16 或更高版本，支持 iPhone 与 iPad。本次说明不涉及 androidx。"),
            "ios-testflight": ("testflight.en-US.txt", "On iPhone and iPad, open TestFlight and check the App Store build."),
            "android": ("android.zh-CN.md", "发现 Android 更新时展示该渠道的版本变化，并保留 APK 校验。支持 Apple Music 链接预览。修复 bios 显示。"),
            "web": ("web.zh-CN.md", "本次仅发布 Web，不要求 Android/iOS 配套发版。"),
            "operators": ("operators.zh-CN.md", "不要求 Android/iOS 配套发版，也不发布 App Store 或 APK。"),
        }
        for channel, (name, text) in allowed.items():
            with self.subTest(channel=channel):
                self.assertEqual(validate_note_text(text, name, channel), text)

    def test_changelog_rejects_opposite_platform_cues_and_keeps_shared_wording(self):
        def entry(identity, platforms, summary, title="搜索"):
            return {"id": identity, "title": title, "summary": summary, "platforms": platforms, "kind": "fix"}

        def changelog(rows):
            return {"schemaVersion": 1, "version": "1.0.15", "buildNumber": 15,
                    "highlights": rows, "breaking": [], "requiredActions": [], "testflightNotes": [],
                    "evidence": {row["id"]: ["proof"] for row in rows}}

        channels = ["android", "ios-app-store", "ios-testflight"]
        shared = entry("shared-search", ["android", "ios"], "改进搜索结果排序。")
        validate_changelog(changelog([shared]), "1.0.15", 15, channels, draft=True)
        android_only = entry("android-drawer", ["android"],
                             "打开账户抽屉情况下，Android 返回操作优先关闭账户抽屉，保留当前页面。",
                             title="账户抽屉返回")
        validate_changelog(changelog([android_only]), "1.0.15", 15, channels, draft=True)
        leaked = entry("pr-1247", ["android", "ios"], android_only["summary"], title="账户抽屉返回")
        with self.assertRaisesRegex(ReleaseError, r"changelog entry pr-1247 lists ios but mentions Android"):
            validate_changelog(changelog([leaked]), "1.0.15", 15, channels, draft=True)
        testflight_only = entry("beta-apk", ["ios-testflight"], "Compare the installed APK.", title="安装包")
        with self.assertRaisesRegex(ReleaseError, r"changelog entry beta-apk lists ios-testflight but mentions APK"):
            validate_changelog(changelog([testflight_only]), "1.0.15", 15, channels)

    def test_scrubbed_store_file_still_rejects_a_mistagged_changelog_entry(self):
        manifest = candidate() | {"schemaVersion": 2, "channels": ["android", "ios-app-store"],
                                  "notes": {"android": "android.zh-CN.md", "ios-app-store": "ios.zh-Hans.txt"},
                                  "baselines": {c: {"tag": "mobile-v1.0.14", "sourceSha": "b" * 40}
                                                for c in ("android", "ios-app-store")}}
        leaked = {"id": "pr-1247", "title": "账户抽屉返回",
                  "summary": "打开账户抽屉情况下，Android 返回操作优先关闭账户抽屉，保留当前页面。",
                  "platforms": ["android", "ios"], "kind": "fix"}
        changelog = {"schemaVersion": 1, "version": manifest["version"], "buildNumber": manifest["buildNumber"],
                     "highlights": [leaked], "breaking": [], "requiredActions": [], "testflightNotes": [],
                     "evidence": {"pr-1247": ["proof"]}}
        (self.path / "evidence.json").write_text(json.dumps({"schemaVersion": 1, "sourceSha": SHA, "evidence": [
            {"id": "proof", "channels": ["android", "ios-app-store"]}]}), encoding="utf-8")
        (self.path / "changelog.json").write_text(json.dumps(changelog, ensure_ascii=False), encoding="utf-8")
        (self.path / "android.zh-CN.md").write_text(render_changelog(changelog, "android"), encoding="utf-8")
        (self.path / "ios.zh-Hans.txt").write_text("改进搜索结果排序。", encoding="utf-8")
        with self.assertRaisesRegex(ReleaseError, r"changelog entry pr-1247 lists ios but mentions Android"):
            validate_candidate(manifest, self.path)

    def test_exact_merged_notes_remain_valid_and_any_edit_is_checked(self):
        root = Path(__file__).resolve().parents[2] / "releases" / "requests"
        for candidate_id in ("mobile-1.0.16-17", "mobile-1.0.17-33", "mobile-1.0.17-33-store-39439793116"):
            folder = root / candidate_id
            manifest = json.loads((folder / "manifest.json").read_text(encoding="utf-8"))
            with self.subTest(candidate_id=candidate_id):
                self.assertTrue(validate_candidate(manifest, folder))
        ios = (root / "mobile-1.0.17-33" / "ios.zh-Hans.txt").read_text(encoding="utf-8").strip()
        fresh = candidate() | {'channels': ['ios-app-store'], 'notes': {'ios-app-store': 'ios.zh-Hans.txt'},
                               'baselines': {'ios-app-store': {'tag': None, 'sourceSha': None}}}
        (self.path / 'ios.zh-Hans.txt').write_text(ios)
        with self.assertRaisesRegex(ReleaseError, 'mentions Android'):
            validate_candidate(fresh, self.path)
        validate_note_text(ios, "ios.zh-Hans.txt", "ios-app-store", candidate_id='mobile-1.0.17-33')
        with self.assertRaisesRegex(ReleaseError, 'mentions Android'):
            validate_note_text(ios, "ios.zh-Hans.txt", "ios-app-store", candidate_id='mobile-1.0.18-40')
        with self.assertRaisesRegex(ReleaseError, r"ios-app-store note ios\.zh-Hans\.txt mentions Android"):
            validate_note_text(ios + "。", "ios.zh-Hans.txt", "ios-app-store")


if __name__ == "__main__":
    unittest.main()
