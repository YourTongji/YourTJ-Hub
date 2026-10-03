"""Receipt and public-store gates for the display-only mobile catalog."""
import unittest
import json
import tempfile
import subprocess
from pathlib import Path
from unittest.mock import patch

from model import FILES, ReleaseError, render_changelog, validate_candidate
from test_model import candidate
from catalog import (successful_channels, public_app_store_version, build_catalog, previous_store_builds,
                     candidate_content_digest, serialize_catalog, MAX_CATALOG_BYTES)


class FakeGitHub:
    repository = "YourTongji/YourTJ-Hub"

    def __init__(self, rows):
        self.rows = rows

    def pages(self, endpoint):
        if endpoint.startswith("deployments?"):
            channel = endpoint.split("environment=release-", 1)[1]
            return [{key: value for key, value in row.items() if key != "_status"}
                    for row in self.rows.get(channel, [])]
        deployment_id = endpoint.split("/")[1]
        return [{"state": self.rows["statuses"].get(deployment_id, "success")}]


def deployment(identity, channel, availability, *, state="success", digest=None):
    assets = {f"app-{i}.apk": digest for i in range(3)} if channel == "android" else {}
    payload = {"candidateId": identity, "channel": channel, "tag": "mobile-v1.2.3",
               "details": {"availability": availability, "buildNumber": 23, "assets": assets}}
    return {"id": str(len(identity) + len(channel)), "sha": "a" * 40, "payload": payload,
            "_status": state}


class CatalogReceiptTests(unittest.TestCase):
    def test_reviewed_store_correction_uses_merge_order_not_receipt_retry_order(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            def git(*args):
                return subprocess.check_output(['git', '-C', str(root), *args], text=True).strip()
            git('init', '-q', '-b', 'main')
            git('config', 'user.name', 'Fixture')
            git('config', 'user.email', 'fixture@example.org')
            original = candidate()['candidateId']
            manifests = {}
            rows = {'android': [], 'ios-app-store': [], 'statuses': {}}
            for index, identity in enumerate((original, original + '-store-1', original + '-store-2')):
                channels = ['android', 'ios-app-store'] if index == 0 else ['ios-app-store']
                manifest = candidate() | {'schemaVersion': 2, 'candidateId': identity, 'channels': channels,
                    'notes': {c: FILES[c] for c in channels},
                    'baselines': {c: {'tag': None, 'sourceSha': None} for c in channels}}
                if index:
                    manifest.update(operation='promote-ios', existingRelease={'candidateId': original, 'buildId': 'apple-15'})
                folder = root / 'releases/requests' / identity
                folder.mkdir(parents=True)
                changelog = {'schemaVersion': 1, 'version': manifest['version'], 'buildNumber': manifest['buildNumber'],
                    'highlights': [{'id': c, 'title': 'Update', 'summary': f'Copy {index}', 'kind': 'feature',
                                    'platforms': [c]} for c in channels],
                    'breaking': [], 'requiredActions': [], 'testflightNotes': [],
                    'evidence': {c: [c] for c in channels}}
                (folder / 'manifest.json').write_text(json.dumps(manifest))
                (folder / 'changelog.json').write_text(json.dumps(changelog))
                (folder / 'evidence.json').write_text(json.dumps({'schemaVersion': 1, 'sourceSha': manifest['sourceSha'],
                    'evidence': [{'id': c, 'channels': [c]} for c in channels]}))
                for channel in channels:
                    (folder / FILES[channel]).write_text(render_changelog(changelog, channel))
                    row = deployment(identity, channel, 'available' if channel == 'android' else 'READY_FOR_DISTRIBUTION', digest='b' * 64)
                    row['id'] = identity + channel
                    row['payload']['tag'] = manifest['tag']
                    row['payload']['details']['buildId'] = 'apple-15'
                    rows[channel].append(row)
                validate_candidate(manifest, folder)
                for row in (r for c in channels for r in rows[c] if r['payload']['candidateId'] == identity):
                    row['payload']['binding'] = {'sourceSha': manifest['sourceSha'], 'contentDigest': candidate_content_digest(folder)}
                manifests[identity] = manifest, folder
                git('add', '.')
                git('commit', '-qm', f'Reviewed candidate {index}')
            # Original receipt retried last; list order must not roll back reviewed copy.
            with patch('catalog.git', side_effect=git), \
                 patch('catalog.load_published_candidate', side_effect=lambda cid, _: manifests[cid]), \
                 patch('catalog.public_app_store_version', return_value=None):
                for reverse in (False, True):
                    if reverse:
                        rows['ios-app-store'].reverse()
                    catalog = build_catalog(FakeGitHub(rows))
                    entries = {e['id']: e for e in catalog['releases'][0]['highlights']}
                    self.assertEqual(entries['ios-app-store']['summary'], 'Copy 2')
                    self.assertEqual(entries['android']['summary'], 'Copy 0')
                # A different source or unrelated promotion may not supersede this binary.
                newer, folder = manifests[original + '-store-2']
                for change in ({'sourceSha': 'c' * 40},
                               {'existingRelease': {'candidateId': 'mobile-9.0.0-15', 'buildId': 'apple-15'}},
                               {'existingRelease': {'candidateId': original, 'buildId': 'other-build'}}):
                    manifests[newer['candidateId']] = newer | change, folder
                    with self.assertRaises(ReleaseError):
                        build_catalog(FakeGitHub(rows))
                manifests[newer['candidateId']] = newer, folder

    def test_requires_successful_receipts_and_channel_specific_public_availability(self):
        rows = {
            "android": [deployment("mobile-1.2.3-23", "android", "available", digest="b" * 64)],
            "ios-testflight": [deployment("mobile-1.2.3-23", "ios-testflight", "APPROVED")],
            "ios-app-store": [deployment("mobile-1.2.3-23", "ios-app-store", "IN_REVIEW")],
            "statuses": {},
        }
        for channel in ("android", "ios-testflight", "ios-app-store"):
            rows["statuses"][rows[channel][0]["id"]] = "success"
        receipt = successful_channels(FakeGitHub(rows), store_version="1.2.3")
        self.assertEqual(set(receipt["mobile-1.2.3-23"]), {"android", "ios-testflight", "ios-app-store"})

        rows["ios-testflight"][0]["payload"]["details"]["availability"] = "WAITING_FOR_REVIEW"
        rows["statuses"][rows["android"][0]["id"]] = "failure"
        receipt = successful_channels(FakeGitHub(rows), store_version="1.2.3")
        self.assertEqual(set(receipt["mobile-1.2.3-23"]), {"ios-app-store"})

    def test_android_receipt_requires_three_sha256_apks(self):
        item = deployment("mobile-1.2.3-23", "android", "available", digest="b" * 63)
        rows = {"android": [item], "statuses": {item["id"]: "success"}}
        with self.assertRaises(ReleaseError):
            successful_channels(FakeGitHub(rows))

    def test_exact_china_store_lookup_is_bounded_and_validates_identity(self):
        class Response:
            headers = {"content-length": "100"}

            def __init__(self, payload): self.payload = payload
            def __enter__(self): return self
            def __exit__(self, *_): pass
            def geturl(self): return "https://itunes.apple.com/cn/lookup?id=6809457637&country=cn"
            def read(self, size):
                self.size = size
                return self.payload[:size]

        good = b'{"results":[{"trackId":6809457637,"bundleId":"tj.yourtj.forumApp","version":"1.2.3"}]}'
        with patch("catalog.urlopen", return_value=Response(good)) as opener:
            self.assertEqual(public_app_store_version(), "1.2.3")
            self.assertEqual(opener.call_args.kwargs["timeout"], 5)
        wrong_app = good.replace(b"6809457637", b"6809457638")
        with patch("catalog.urlopen", return_value=Response(wrong_app)):
            self.assertIsNone(public_app_store_version())

    def test_catalog_composes_only_receipt_channels_and_tracks_structured_history(self):
        with tempfile.TemporaryDirectory() as root:
            root = Path(root)
            manifests, paths, rows = {}, {}, {channel: [] for channel in ("android", "ios-testflight", "ios-app-store")}
            rows["statuses"] = {}
            def add(candidate_id, build, channels, structured=True):
                folder = root / candidate_id
                folder.mkdir()
                manifest = {"candidateId": candidate_id, "product": "mobile", "schemaVersion": 2 if structured else 1,
                            "version": "2.3.4", "buildNumber": build, "tag": "mobile-v2.3.4", "sourceSha": "a" * 40,
                            "channels": channels}
                if structured:
                    changelog = {"highlights": [{"id": "chat-stickers", "title": "Sticker picker",
                                                   "summary": "Browse stickers in chat.",
                                                   "kind": "feature", "platforms": ["android"]}],
                     "breaking": [], "requiredActions": [], "testflightNotes": [{"id": "course-widget-check",
                         "text": "Verify the course widget after changing a course."}]}
                    (folder / "changelog.json").write_text(json.dumps(changelog), encoding="utf-8")
                manifests[candidate_id] = (manifest, folder)
                paths[candidate_id] = f"releases/requests/{candidate_id}"
                for channel in channels:
                    item = deployment(candidate_id, channel, {"android": "available", "ios-testflight": "APPROVED",
                                                               "ios-app-store": "READY_FOR_SALE"}[channel],
                                       digest="b" * 64 if channel == "android" else None)
                    item["id"] = f"{candidate_id}:{channel}"
                    item["payload"]["tag"] = manifest["tag"]
                    item["payload"]["binding"] = {"sourceSha": manifest["sourceSha"],
                                                   "contentDigest": "digest"}
                    rows[channel].append(item)
                    rows["statuses"][item["id"]] = "success"
            add("mobile-2.3.4-41", 41, ["android"], structured=False)
            add("mobile-2.3.4-42", 42, ["android"])
            add("mobile-2.3.4-43", 43, ["android", "ios-testflight"])
            github = FakeGitHub(rows)
            with patch("catalog.candidate_paths", return_value=paths), \
                 patch("catalog.load_published_candidate", side_effect=lambda cid, _folder: manifests[cid]), \
                 patch("catalog.candidate_content_digest", return_value="digest"), \
                 patch("catalog.public_app_store_version", return_value=None):
                catalog = build_catalog(github, published_at="2026-10-03T00:00:00Z")
            self.assertEqual([row["buildNumber"] for row in catalog["releases"]], [43, 42])
            self.assertEqual(catalog["releases"][0]["channels"], ["android", "ios-testflight"])
            self.assertEqual(catalog["releases"][0]["highlights"][0]["platforms"], ["android"])
            self.assertEqual(catalog["releases"][0]["testflightNotes"], [{"id": "course-widget-check",
                "text": "Verify the course widget after changing a course."}])
            self.assertEqual(catalog["historyCoverage"]["byChannel"]["android"],
                             {"completeFromBuild": 41, "throughBuild": 43, "coveredBuilds": [42, 43]})
            fixture = json.loads(Path("apps/mobile/packages/forum_app/test/fixtures/mobile_release_catalog.json").read_text())
            self.maxDiff = None
            self.assertEqual(catalog, fixture)

    def structured_catalog(self, builds, entries=lambda build: [{"id": f"change-{build}", "title": "Update",
                                                                 "summary": "Faster feed.", "kind": "improvement",
                                                                 "platforms": ["android"]}]):
        manifests, paths, rows = {}, {}, {"android": [], "ios-testflight": [], "ios-app-store": [], "statuses": {}}
        with tempfile.TemporaryDirectory() as root:
            for build in builds:
                candidate_id = f"mobile-2.3.4-{build}"
                folder = Path(root) / candidate_id
                folder.mkdir()
                (folder / "changelog.json").write_text(json.dumps({
                    "highlights": entries(build), "breaking": [], "requiredActions": [], "testflightNotes": []},
                    ensure_ascii=False), encoding="utf-8")
                manifests[candidate_id] = ({"candidateId": candidate_id, "product": "mobile", "schemaVersion": 2,
                                            "version": "2.3.4", "buildNumber": build, "tag": "mobile-v2.3.4",
                                            "sourceSha": "a" * 40, "channels": ["android"]}, folder)
                paths[candidate_id] = f"releases/requests/{candidate_id}"
                item = deployment(candidate_id, "android", "available", digest="b" * 64)
                item["id"] = candidate_id
                item["payload"]["tag"] = "mobile-v2.3.4"
                item["payload"]["binding"] = {"sourceSha": "a" * 40, "contentDigest": "digest"}
                rows["android"].append(item)
            with patch("catalog.candidate_paths", return_value=paths), \
                 patch("catalog.load_published_candidate", side_effect=lambda cid, _folder: manifests[cid]), \
                 patch("catalog.candidate_content_digest", return_value="digest"), \
                 patch("catalog.public_app_store_version", return_value=None):
                return build_catalog(FakeGitHub(rows), published_at="2026-10-03T00:00:00Z")

    def test_catalog_keeps_the_client_release_window_and_matching_coverage(self):
        catalog = self.structured_catalog(range(1, 302))
        self.assertEqual(len(catalog["releases"]), 300)
        self.assertEqual(catalog["releases"][-1]["buildNumber"], 2)
        coverage = catalog["historyCoverage"]["byChannel"]["android"]
        self.assertEqual((coverage["completeFromBuild"], coverage["throughBuild"]), (1, 301))
        self.assertEqual(coverage["coveredBuilds"], list(range(2, 302)))

    def test_catalog_trims_oldest_history_to_the_client_byte_budget(self):
        long = "界面" * 175
        catalog = self.structured_catalog(range(1, 101), lambda build: [
            {"id": f"change-{build}-{i}", "title": "体验改进", "summary": long, "kind": "improvement",
             "platforms": ["android"]} for i in range(10)])
        size = len(serialize_catalog(catalog).encode("utf-8"))
        self.assertLessEqual(size, MAX_CATALOG_BYTES)
        self.assertGreater(size, MAX_CATALOG_BYTES - 20000)
        builds = [release["buildNumber"] for release in catalog["releases"]]
        self.assertEqual(builds[0], 100)
        self.assertLess(len(builds), 100)
        coverage = catalog["historyCoverage"]["byChannel"]["android"]
        self.assertEqual(coverage["coveredBuilds"], sorted(builds))
        self.assertEqual(coverage["completeFromBuild"], min(builds) - 1)

    def test_catalog_omits_a_build_whose_merged_groups_exceed_the_client_limit(self):
        catalog = self.structured_catalog([1, 2, 3], lambda build: [
            {"id": f"change-{build}-{i}", "title": "Fix", "summary": "Fixed.", "kind": "fix",
             "platforms": ["android"]} for i in range(101 if build == 2 else 1)])
        self.assertEqual([release["buildNumber"] for release in catalog["releases"]], [3, 1])
        coverage = catalog["historyCoverage"]["byChannel"]["android"]
        self.assertEqual((coverage["completeFromBuild"], coverage["coveredBuilds"]), (2, [3]))

    def test_catalog_rejects_candidate_receipt_digest_drift_and_conflicting_shared_ids(self):
        with tempfile.TemporaryDirectory() as root:
            root = Path(root)
            manifests, paths = {}, {}
            for candidate_id, channel, title, summary in (
                ("mobile-1.2.3-43", "android", "Android", "Android copy"),
                ("mobile-1.2.3-43-store-991", "ios-app-store", "iOS", "iOS copy"),
            ):
                folder = root / candidate_id
                folder.mkdir()
                changelog = {"highlights": [{"id": "same", "title": title, "summary": summary,
                                              "kind": "feature", "platforms": [channel if channel == "android" else "ios"]}],
                             "breaking": [], "requiredActions": [], "testflightNotes": []}
                (folder / "changelog.json").write_text(json.dumps(changelog), encoding="utf-8")
                manifests[candidate_id] = ({"candidateId": candidate_id, "product": "mobile", "schemaVersion": 2,
                    "version": "1.2.3", "buildNumber": 43, "tag": "mobile-v1.2.3", "sourceSha": "a" * 40,
                    "channels": [channel]}, folder)
                paths[candidate_id] = "request"

            def call(binding_digest):
                rows = {"android": [], "ios-testflight": [], "ios-app-store": [], "statuses": {}}
                for candidate_id, channel, availability in (
                    ("mobile-1.2.3-43", "android", "available"),
                    ("mobile-1.2.3-43-store-991", "ios-app-store", "READY_FOR_SALE"),
                ):
                    item = deployment(candidate_id, channel, availability, digest="b" * 64)
                    item["id"] = channel
                    item["payload"]["tag"] = "mobile-v1.2.3"
                    item["payload"]["binding"] = {"sourceSha": "a" * 40, "contentDigest": binding_digest}
                    rows[channel].append(item)
                    rows["statuses"][channel] = "success"
                with patch("catalog.candidate_paths", return_value=paths), \
                     patch("catalog.load_published_candidate", side_effect=lambda cid, _folder: manifests[cid]), \
                     patch("catalog.candidate_content_digest", return_value="digest"), \
                     patch("catalog.public_app_store_version", return_value=None):
                    return build_catalog(FakeGitHub(rows))
            with self.assertRaisesRegex(ReleaseError, "does not match"):
                call("wrong-digest")
            with self.assertRaisesRegex(ReleaseError, "Conflicting platform text"):
                call("digest")

    def test_prior_catalog_preserves_only_verified_public_store_builds(self):
        prior = {"schemaVersion": 1, "historyCoverage": {"byChannel": {"ios-app-store": {
            "coveredBuilds": [10, 11]}}}, "releases": [
                {"version": "1.0.0", "buildNumber": 10, "channels": ["ios-app-store"]},
                {"version": "1.1.0", "buildNumber": 11, "channels": ["ios-app-store"]},
                {"version": "1.2.0", "buildNumber": 12, "channels": ["ios-app-store"]},
                {"version": "1.0.0", "buildNumber": 10, "channels": ["android"]}]}
        release = {"assets": [{"name": "releases.json"}]}
        def download(*args):
            directory = Path(args[args.index("--dir") + 1])
            (directory / "releases.json").write_text(json.dumps(prior), encoding="utf-8")
        with patch("catalog.run", side_effect=download):
            self.assertEqual(previous_store_builds(FakeGitHub({}), release), {("1.0.0", 10), ("1.1.0", 11)})


if __name__ == "__main__":
    unittest.main()
