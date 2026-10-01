"""Release authority is external evidence, never a flag supplied by a candidate."""
import copy
import tempfile
import unittest
from pathlib import Path

from model import ReleaseError, validate_candidate, validate_approval, next_identity, channels_for


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


if __name__ == "__main__":
    unittest.main()
