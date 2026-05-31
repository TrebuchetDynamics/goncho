#!/usr/bin/env python3
"""Local goncho-server smoke test.

Builds goncho-server, starts it on a random loopback port, exercises health,
write, search, recall, and context, then shuts it down.
"""

from __future__ import annotations

import json
import socket
import subprocess
import sys
import tempfile
import time
import urllib.parse
from pathlib import Path

try:
    from shared.http_json import post_json_url, read_json_url
except ModuleNotFoundError:  # pragma: no cover - package import path
    from scripts.shared.http_json import post_json_url, read_json_url


WORKSPACE = "server-smoke-workspace"
PEER = "server-smoke-peer"
SESSION = "server-smoke-session"
MEMORY = "Goncho server smoke remembers the quartz llama."
QUERY = "quartz llama"


def main() -> int:
    root = Path(__file__).resolve().parents[2]
    with tempfile.TemporaryDirectory(prefix="goncho-server-smoke-") as tmp:
        tmpdir = Path(tmp)
        binary = tmpdir / ("goncho-server.exe" if sys.platform == "win32" else "goncho-server")
        subprocess.run(["go", "build", "-o", str(binary), "./cmd/goncho-server"], cwd=root, check=True)

        port = free_loopback_port()
        base_url = f"http://127.0.0.1:{port}"
        db_path = tmpdir / "goncho.db"
        proc = subprocess.Popen(
            [
                str(binary),
                "serve",
                "-db",
                str(db_path),
                "-addr",
                f"127.0.0.1:{port}",
                "-workspace",
                WORKSPACE,
                "-observer",
                "server-smoke-observer",
            ],
            cwd=root,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
        )
        try:
            wait_for_health(base_url, proc)
            health = get_json(f"{base_url}/health")
            require(health.get("status") == "ok", f"health status not ok: {health}")
            require(health.get("db", {}).get("status") == "ok", f"db health not ok: {health}")

            mcp_tools = post_json(f"{base_url}/mcp", {"jsonrpc": "2.0", "id": 1, "method": "tools/list"})
            tool_names = {tool.get("name") for tool in mcp_tools.get("result", {}).get("tools", [])}
            require({"goncho_remember", "goncho_search", "goncho_recall"}.issubset(tool_names), f"MCP tools missing: {mcp_tools}")

            post_json(
                f"{base_url}/v3/workspaces/{WORKSPACE}/conclusions",
                {"peer_id": PEER, "conclusion": MEMORY, "session_key": SESSION},
            )

            search = post_json(
                f"{base_url}/v3/workspaces/{WORKSPACE}/peers/{PEER}/search",
                {"query": QUERY, "session_key": SESSION},
            )
            require(any(row.get("content") == MEMORY for row in search.get("results", [])), f"search missed memory: {search}")

            recall = post_json(
                f"{base_url}/v3/workspaces/{WORKSPACE}/peers/{PEER}/recall",
                {"query": QUERY, "session_key": SESSION, "limit": 5},
            )
            require(
                any(row.get("candidate", {}).get("content") == MEMORY for row in recall.get("selected", [])),
                f"recall missed memory: {recall}",
            )

            params = urllib.parse.urlencode({"query": QUERY, "session_id": SESSION})
            context = get_json(f"{base_url}/v3/workspaces/{WORKSPACE}/peers/{PEER}/context?{params}")
            require(MEMORY in context.get("conclusions", []), f"context missed memory: {context}")

            print(json.dumps({"status": "ok", "base_url": base_url, "db": str(db_path)}, sort_keys=True))
            return 0
        finally:
            terminate(proc)


def free_loopback_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("127.0.0.1", 0))
        return int(sock.getsockname()[1])


def wait_for_health(base_url: str, proc: subprocess.Popen[str]) -> None:
    deadline = time.monotonic() + 20
    last_error: Exception | None = None
    while time.monotonic() < deadline:
        if proc.poll() is not None:
            raise RuntimeError(f"goncho-server exited early with {proc.returncode}: {collect_output(proc)}")
        try:
            health = get_json(f"{base_url}/health", timeout=1)
            if health.get("status") == "ok":
                return
        except Exception as exc:  # noqa: BLE001 - retry transient startup failures
            last_error = exc
            time.sleep(0.1)
    raise RuntimeError(f"timed out waiting for /health: {last_error}; {collect_output(proc)}")


def get_json(url: str, timeout: float = 5) -> dict:
    return read_json_url(url, timeout=timeout)


def post_json(url: str, body: dict, timeout: float = 5) -> dict:
    return post_json_url(url, body, timeout=timeout)


def require(condition: bool, message: str) -> None:
    if not condition:
        raise AssertionError(message)


def terminate(proc: subprocess.Popen[str]) -> None:
    if proc.poll() is not None:
        return
    proc.terminate()
    try:
        proc.wait(timeout=5)
    except subprocess.TimeoutExpired:
        proc.kill()
        proc.wait(timeout=5)


def collect_output(proc: subprocess.Popen[str]) -> str:
    stdout = ""
    stderr = ""
    if proc.stdout is not None:
        try:
            stdout = proc.stdout.read()
        except Exception:  # noqa: BLE001
            stdout = ""
    if proc.stderr is not None:
        try:
            stderr = proc.stderr.read()
        except Exception:  # noqa: BLE001
            stderr = ""
    return f"stdout={stdout!r} stderr={stderr!r}"


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Exception as exc:  # noqa: BLE001
        print(f"server smoke failed: {exc}", file=sys.stderr)
        raise SystemExit(1)
