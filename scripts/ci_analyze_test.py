"""Hermetic checks for the local CI analyst; no GitHub or inference calls."""

import argparse
import importlib.util
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch


spec = importlib.util.spec_from_file_location("ci_analyze", Path(__file__).with_name("ci-analyze.py"))
analyst = importlib.util.module_from_spec(spec)
spec.loader.exec_module(analyst)


class AnalystTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.args = argparse.Namespace(repo="acme/dva", branch="master", run_id=42,
                                       output=Path(self.temp.name), collect_only=False, timeout=10)
        self.run = {"id": 42, "run_attempt": 1, "status": "completed", "conclusion": "failure",
                    "head_sha": "a" * 40, "head_branch": "master", "event": "push",
                    "head_repository": {"full_name": "acme/dva"}, "path": ".github/workflows/ci.yml"}
        self.destination = self.args.output / ("42-1-" + "a" * 12)

    def api(self, repo, endpoint):
        if endpoint.startswith("commits/"):
            return {"sha": "a" * 40, "files": []}
        if endpoint.endswith("/jobs?per_page=100"):
            return {"jobs": []}
        return self.run

    def codex(self, args, **kwargs):
        if args[:3] == ["codex", "login", "status"]:
            return "Logged in using ChatGPT"
        self.assertIn("read-only", args)
        self.assertIn("--ignore-user-config", args)
        self.assertIn("--ignore-rules", args)
        self.assertNotIn("OPENAI_API_KEY", kwargs["env"])
        Path(args[args.index("--output-last-message") + 1]).write_text("# 분석\n근거 부족\n")
        return ""

    def test_rejects_success_pending_stale_fork_and_wrong_workflow(self):
        for field, value in [("conclusion", "success"), ("status", "in_progress"),
                             ("head_sha", "b" * 40), ("event", "pull_request"),
                             ("head_repository", {"full_name": "fork/dva"}),
                             ("path", ".github/workflows/other.yml")]:
            with self.subTest(field=field):
                self.assertFalse(analyst.eligible(dict(self.run, **{field: value}),
                                                 "acme/dva", "master", "a" * 40))

    def test_report_and_duplicate_suppression(self):
        with patch.object(analyst, "github", self.api), patch.object(analyst, "command", side_effect=self.codex) as cmd:
            analyst.analyze(self.args)
            analyst.analyze(self.args)
            self.assertEqual(cmd.call_count, 2)  # login + one analysis
        self.assertTrue((self.destination / "report.md").is_file())

    def test_collect_only_never_invokes_codex(self):
        self.args.collect_only = True
        with patch.object(analyst, "github", self.api), patch.object(analyst, "command") as cmd:
            analyst.analyze(self.args)
            cmd.assert_not_called()
        self.assertTrue((self.destination / "prompt.txt").is_file())
        self.assertFalse((self.destination / "report.md").exists())

    def test_api_login_is_rejected(self):
        with patch.object(analyst, "github", self.api), patch.object(analyst, "command", return_value="Logged in using an API key"):
            with self.assertRaisesRegex(RuntimeError, "subscription login required"):
                analyst.analyze(self.args)
        self.assertFalse((self.destination / "report.md").exists())

    def test_rerun_during_analysis_is_stale(self):
        calls = 0

        def api(repo, endpoint):
            nonlocal calls
            if endpoint == "actions/runs/42":
                calls += 1
                return dict(self.run, run_attempt=calls)
            return self.api(repo, endpoint)

        with patch.object(analyst, "github", api), patch.object(analyst, "command", side_effect=self.codex):
            analyst.analyze(self.args)
        self.assertTrue((self.destination / "report.stale.md").is_file())
        self.assertFalse((self.destination / "report.md").exists())

    def test_missing_log_is_visible_in_evidence(self):
        self.args.collect_only = True

        def api(repo, endpoint):
            if endpoint.endswith("/jobs?per_page=100"):
                return {"jobs": [{"id": 7, "name": "test", "conclusion": "cancelled"}]}
            return self.api(repo, endpoint)

        with patch.object(analyst, "github", api), patch.object(analyst, "command", side_effect=RuntimeError("404 BlobNotFound")):
            analyst.analyze(self.args)
        self.assertIn("BlobNotFound", (self.destination / "evidence.json").read_text())

    def test_evidence_truncation_keeps_both_ends(self):
        result = analyst.bounded("start" + "x" * (analyst.MAX_EVIDENCE * 2) + "end")
        self.assertTrue(result.startswith("start"))
        self.assertTrue(result.endswith("end"))
        self.assertIn("EVIDENCE TRUNCATED", result)

    def test_command_timeout_fails_instead_of_passing(self):
        with self.assertRaises(subprocess.TimeoutExpired):
            analyst.command([sys.executable, "-c", "import time; time.sleep(30)"], timeout=0.05)


if __name__ == "__main__":
    unittest.main()
