import unittest
from select_inputs import select, verify_results


class SelectTests(unittest.TestCase):
    def test_docs_still_get_governance_and_current_vulnerability_scan(self):
        plan = select(["docs/operations/releases.md"])
        self.assertEqual({k for k, v in plan.items() if v["run"]}, {"docs", "govulncheck"})

    def test_mobile_platforms_and_unknown_executables(self):
        for path in ["apps/mobile/packages/forum_app/android/app/build.gradle.kts", "apps/mobile/packages/forum_app/ios/Runner/A.swift"]:
            plan = select([path])
            self.assertTrue(plan["mobile"]["run"])
            self.assertTrue(plan["android" if "/android/" in path else "ios"]["run"])
            self.assertFalse(plan["ios" if "/android/" in path else "android"]["run"])
            self.assertFalse(plan["deploy"]["run"])
        self.assertTrue(all(v["run"] for v in select(["unknown/execute.sh"]).values()))

    def test_missing_skipped_cancelled_fail_closed(self):
        plan = select(["apps/mobile/packages/core/lib/client.dart"])
        good = {k: {"result": "success" if v["run"] else "skipped"} for k, v in plan.items() if k != "deploy"}
        verify_results(plan, good)
        for result in ["skipped", "cancelled", "failure", ""]:
            with self.assertRaises(ValueError):
                verify_results(plan, good | {"mobile": {"result": result}})
        del good["mobile"]
        with self.assertRaises(ValueError):
            verify_results(plan, good)

    def test_service_changes_include_postgres_and_race(self):
        plan = select(["apps/gooseforum/app/service/test.go"])
        self.assertTrue(plan["backend"]["run"] and plan["postgres"]["run"] and plan["deploy"]["run"])

    def test_metadata_does_not_deploy_or_build_native(self):
        plan = select(["releases/requests/mobile-1.0.15-15/ios.zh-Hans.txt"])
        self.assertTrue(plan["release"]["run"])
        self.assertFalse(plan["deploy"]["run"] or plan["mobile"]["run"])


if __name__ == "__main__":
    unittest.main()
