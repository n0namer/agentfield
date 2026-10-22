"""External real-process Windows canary for the provider-neutral harness substrate."""

from __future__ import annotations

import json
import os
import subprocess
import sys


_CANARY = r'''
import asyncio
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import time

from agentfield.harness._cli import run_cli


def alive(pid):
    try:
        os.kill(pid, 0)
    except OSError:
        return False
    return True


async def wait_for_file(path):
    deadline = time.monotonic() + 5
    while time.monotonic() < deadline:
        if path.exists():
            return int(path.read_text(encoding="utf-8"))
        await asyncio.sleep(0.02)
    raise AssertionError("child did not record its grandchild PID")


async def main():
    with tempfile.TemporaryDirectory() as root:
        cwd = Path(root) / "nested-cwd"
        cwd.mkdir()
        pid_path = Path(root) / "grandchild.pid"
        marker = "agentfield-runtime-canary"
        expected_depth = str(max(int(os.environ.get("AGENTFIELD_HARNESS_DEPTH", "0")), 0) + 1)
        report = (
            "import json, os; "
            "print(json.dumps({'cwd': os.getcwd(), 'marker': os.getenv('AF_CANARY_MARKER'), "
            "'depth': os.getenv('AGENTFIELD_HARNESS_DEPTH')}), flush=True)"
        )
        stdout, stderr, code = await run_cli(
            [sys.executable, "-c", report], cwd=str(cwd),
            env={"AF_CANARY_MARKER": marker}, timeout=5, idle_seconds=2,
        )
        first = json.loads(stdout)
        assert stderr == "" and code == 0
        assert Path(first["cwd"]).resolve() == cwd.resolve()
        assert first["marker"] == marker and first["depth"] == expected_depth
        assert os.environ.get("AF_CANARY_MARKER") != marker

        stdout, _, code = await run_cli(
            [sys.executable, "-c", report], cwd=str(cwd), timeout=5, idle_seconds=2,
        )
        restarted = json.loads(stdout)
        assert code == 0 and restarted["marker"] is None
        assert restarted["depth"] == expected_depth

        try:
            await run_cli([sys.executable, "-c", "import time; time.sleep(30)"],
                          cwd=str(cwd), timeout=0.1, idle_seconds=0)
        except TimeoutError as exc:
            assert "timed out" in str(exc)
        else:
            raise AssertionError("timeout was reported as success")

        tree = (
            "import pathlib, subprocess, sys, time; "
            "child = subprocess.Popen([sys.executable, '-c', 'import time; time.sleep(30)']); "
            "pathlib.Path(sys.argv[1]).write_text(str(child.pid), encoding='utf-8'); "
            "print('ready', flush=True); time.sleep(30)"
        )
        task = asyncio.create_task(run_cli(
            [sys.executable, "-c", tree, str(pid_path)], cwd=str(cwd),
            timeout=20, idle_seconds=0,
        ))
        pid = await wait_for_file(pid_path)
        assert alive(pid)
        task.cancel()
        try:
            await task
        except asyncio.CancelledError:
            pass
        else:
            raise AssertionError("cancellation did not propagate")
        deadline = time.monotonic() + 5
        while alive(pid) and time.monotonic() < deadline:
            await asyncio.sleep(0.05)
        assert not alive(pid), "cancellation leaked a grandchild process"
        print(json.dumps({"cwd_env": True, "restart": True, "timeout": True, "cancel_tree": True}))


asyncio.run(main())
'''


def test_real_runtime_canary_isolates_state_and_reaps_nested_child():
    """The canary process uses AgentField's real subprocess implementation."""
    # pytest-cov exports process-start controls which cause a nested Python
    # interpreter to boot test instrumentation rather than the user runtime.
    # Strip only those controls; the canary otherwise inherits the real host
    # environment so cwd/env isolation and harness-depth behavior remain real.
    child_env = {
        key: value
        for key, value in os.environ.items()
        if not key.startswith(("COV_CORE_", "COVERAGE_", "PYTEST_"))
    }
    completed = subprocess.run(
        [sys.executable, "-c", _CANARY],
        capture_output=True,
        text=True,
        timeout=45,
        check=False,
        env=child_env,
    )
    assert completed.returncode == 0, completed.stderr or completed.stdout
    assert json.loads(completed.stdout) == {
        "cwd_env": True,
        "restart": True,
        "timeout": True,
        "cancel_tree": True,
    }
