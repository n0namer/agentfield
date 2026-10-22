"""Independent future native receipt/persistence/restart acceptance oracle.

Run directly with:
    python sdk/python/tests/test_releasegate_future_idempotency_qa_20261013.py -v

Set NATIVE_FUTURE_ORACLE_BUNDLE to a JSON evidence bundle to execute the live
source-bound gate. Synthetic fixtures exercise oracle behavior but can never
produce a live acceptance result.
"""

from __future__ import annotations

import hashlib
import json
import os
import re
import unittest
from pathlib import Path
from typing import Any

BUNDLE_ENV = "NATIVE_FUTURE_ORACLE_BUNDLE"
SHA256_RE = re.compile(r"^[0-9a-f]{64}$")
QUALIFIED_SIGNATURE_RE = re.compile(r"^[A-Za-z_][\w.]*(?:\([^\n]*\))?(?:\s*->\s*[^\n]+)?$")
TERMINAL_STATUSES = {"SUCCEEDED", "FAILED"}


class OracleViolation(AssertionError):
    """Raised when evidence cannot support the native acceptance contract."""


def _require(condition: bool, message: str) -> None:
    if not condition:
        raise OracleViolation(message)


def _canonical_digest(value: Any) -> str:
    payload = json.dumps(value, sort_keys=True, separators=(",", ":")).encode("utf-8")
    return hashlib.sha256(payload).hexdigest()


def _require_sha(value: Any, label: str) -> None:
    _require(isinstance(value, str) and SHA256_RE.fullmatch(value) is not None, f"{label} must be lowercase SHA-256")


def _require_signature(value: Any, label: str) -> None:
    _require(
        isinstance(value, str) and QUALIFIED_SIGNATURE_RE.fullmatch(value) is not None and "." in value,
        f"{label} must be a qualified implementation signature",
    )


def validate_native_bundle(bundle: dict[str, Any], *, require_live: bool) -> dict[str, str]:
    """Validate source identity, DTO boundaries, positive/failure/restart receipts."""

    _require(bundle.get("schema") == "native-future-independent-evidence.v1", "unsupported evidence schema")
    mode = bundle.get("evidence_mode")
    _require(mode in {"synthetic", "live"}, "evidence_mode must be synthetic or live")
    if require_live:
        _require(mode == "live", "live gate rejects synthetic fixtures")

    source = bundle.get("source")
    _require(isinstance(source, dict), "source identity is required")
    _require_sha(source.get("git_sha256"), "source.git_sha256")
    files = source.get("files")
    _require(isinstance(files, list) and files, "source.files must bind implementation files")
    for index, item in enumerate(files):
        _require(isinstance(item, dict), f"source.files[{index}] must be an object")
        _require(isinstance(item.get("path"), str) and item["path"].strip(), f"source.files[{index}].path is required")
        _require_sha(item.get("sha256"), f"source.files[{index}].sha256")

    boundary = bundle.get("dto_boundary")
    _require(isinstance(boundary, dict), "dto_boundary is required")
    _require(isinstance(boundary.get("version"), str) and boundary["version"].strip(), "dto_boundary.version is required")
    for field in ("request_signature", "receipt_signature", "persistence_signature"):
        _require_signature(boundary.get(field), f"dto_boundary.{field}")
    boundary_files = boundary.get("source_paths")
    _require(isinstance(boundary_files, list) and boundary_files, "dto_boundary.source_paths are required")
    known_paths = {item["path"] for item in files}
    _require(set(boundary_files).issubset(known_paths), "DTO signatures must be bound to hashed source files")

    attempts = bundle.get("attempts")
    _require(isinstance(attempts, list) and len(attempts) >= 3, "positive, failure, and restart attempts are required")
    by_phase: dict[str, dict[str, Any]] = {}
    for attempt in attempts:
        _require(isinstance(attempt, dict), "each attempt must be an object")
        phase = attempt.get("phase")
        _require(phase in {"initial", "failure", "restart_retry"}, "unknown attempt phase")
        _require(phase not in by_phase, f"duplicate {phase} attempt")
        _require(attempt.get("status") in TERMINAL_STATUSES, f"{phase} must be terminal")
        _require(attempt.get("persisted") is True, f"{phase} terminal state must be persisted")
        for field in ("request_id", "operation_id", "receipt_id", "process_instance"):
            _require(isinstance(attempt.get(field), str) and attempt[field], f"{phase}.{field} is required")
        _require_sha(attempt.get("payload_sha256"), f"{phase}.payload_sha256")
        by_phase[phase] = attempt

    initial = by_phase.get("initial")
    failure = by_phase.get("failure")
    restart = by_phase.get("restart_retry")
    _require(initial is not None and failure is not None and restart is not None, "all three phases are required")
    _require(initial["status"] == "SUCCEEDED", "initial positive path must succeed")
    _require(failure["status"] == "FAILED", "failure path must remain failed")
    _require(isinstance(failure.get("error_code"), str) and failure["error_code"], "failure path needs an error_code")
    _require(restart["status"] == "SUCCEEDED", "restart retry must converge to success")
    _require(restart.get("recovered_from_restart") is True, "restart recovery marker is required")
    _require(restart["process_instance"] != initial["process_instance"], "restart must use a new process instance")

    for field in ("request_id", "operation_id", "receipt_id", "payload_sha256"):
        _require(restart[field] == initial[field], f"restart must preserve initial {field}")
    _require(restart.get("publication_count") == 1, "restart must not duplicate result publication")
    _require(initial.get("publication_count") == 1, "initial path must publish exactly once")
    _require(failure.get("publication_count") == 1, "failure path must publish exactly one failed receipt")
    _require(failure["receipt_id"] != initial["receipt_id"], "failure receipt must not alias success receipt")

    persistence = bundle.get("persistence")
    _require(isinstance(persistence, dict), "persistence evidence is required")
    _require(persistence.get("durable_after_restart") is True, "receipt must survive restart")
    _require(persistence.get("receipt_id") == initial["receipt_id"], "persisted receipt must match initial receipt")
    _require_sha(persistence.get("record_sha256"), "persistence.record_sha256")

    command = bundle.get("command")
    _require(isinstance(command, dict), "reproducible command receipt is required")
    _require(isinstance(command.get("argv"), list) and command["argv"], "command.argv is required")
    _require(command.get("exit_code") == 0, "native evidence command must exit zero")
    _require(isinstance(command.get("duration_ms"), int) and command["duration_ms"] >= 0, "command.duration_ms is required")

    return {
        "bundle_sha256": _canonical_digest(bundle),
        "source_sha256": source["git_sha256"],
        "dto_version": boundary["version"],
        "receipt_id": initial["receipt_id"],
    }


def _fixture(*, failure_status: str = "FAILED", restart_publications: int = 1) -> dict[str, Any]:
    sha_a = "a" * 64
    sha_b = "b" * 64
    payload_sha = "c" * 64
    return {
        "schema": "native-future-independent-evidence.v1",
        "evidence_mode": "synthetic",
        "source": {
            "git_sha256": "d" * 64,
            "files": [
                {"path": "src/native/request.ts", "sha256": sha_a},
                {"path": "src/native/store.ts", "sha256": sha_b},
            ],
        },
        "dto_boundary": {
            "version": "c4-native.v1",
            "request_signature": "native.request.submit(NativeRequest) -> NativeReceipt",
            "receipt_signature": "native.receipt.publish(NativeReceipt) -> None",
            "persistence_signature": "native.store.save(NativeReceipt) -> PersistedReceipt",
            "source_paths": ["src/native/request.ts", "src/native/store.ts"],
        },
        "attempts": [
            {
                "phase": "initial",
                "status": "SUCCEEDED",
                "persisted": True,
                "request_id": "req-1",
                "operation_id": "op-1",
                "receipt_id": "receipt-1",
                "process_instance": "process-a",
                "payload_sha256": payload_sha,
                "publication_count": 1,
            },
            {
                "phase": "failure",
                "status": failure_status,
                "persisted": True,
                "request_id": "req-failure",
                "operation_id": "op-failure",
                "receipt_id": "receipt-failure",
                "process_instance": "process-a",
                "payload_sha256": "e" * 64,
                "publication_count": 1,
                "error_code": "PRODUCER_FAILED",
            },
            {
                "phase": "restart_retry",
                "status": "SUCCEEDED",
                "persisted": True,
                "request_id": "req-1",
                "operation_id": "op-1",
                "receipt_id": "receipt-1",
                "process_instance": "process-b",
                "payload_sha256": payload_sha,
                "publication_count": restart_publications,
                "recovered_from_restart": True,
            },
        ],
        "persistence": {
            "durable_after_restart": True,
            "receipt_id": "receipt-1",
            "record_sha256": "f" * 64,
        },
        "command": {
            "argv": ["native-e2e", "--failure", "--restart"],
            "exit_code": 0,
            "duration_ms": 125,
        },
    }


class NativeFutureOracleContractTests(unittest.TestCase):
    def test_positive_failure_and_restart_contract(self) -> None:
        result = validate_native_bundle(_fixture(), require_live=False)
        self.assertEqual(result["dto_version"], "c4-native.v1")
        self.assertEqual(result["receipt_id"], "receipt-1")

    def test_failure_cannot_be_reduced_to_success(self) -> None:
        with self.assertRaisesRegex(OracleViolation, "failure path must remain failed"):
            validate_native_bundle(_fixture(failure_status="SUCCEEDED"), require_live=False)

    def test_restart_duplicate_publication_is_rejected(self) -> None:
        with self.assertRaisesRegex(OracleViolation, "must not duplicate"):
            validate_native_bundle(_fixture(restart_publications=2), require_live=False)

    def test_synthetic_bundle_cannot_pass_live_gate(self) -> None:
        with self.assertRaisesRegex(OracleViolation, "rejects synthetic"):
            validate_native_bundle(_fixture(), require_live=True)

    def test_live_source_bound_bundle_from_environment(self) -> None:
        bundle_path = os.environ.get(BUNDLE_ENV)
        if not bundle_path:
            self.skipTest(f"{BUNDLE_ENV} is not set; live native evidence is unavailable")
        path = Path(bundle_path)
        self.assertTrue(path.is_file(), f"live evidence bundle not found: {path}")
        with path.open("r", encoding="utf-8") as stream:
            bundle = json.load(stream)
        result = validate_native_bundle(bundle, require_live=True)
        print(json.dumps({"live_oracle": "PASS", **result}, sort_keys=True))


if __name__ == "__main__":
    unittest.main()
