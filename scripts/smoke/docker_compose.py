#!/usr/bin/env python3
"""Smoke-test Goncho's local shared-service docker compose packaging.

The smoke is intentionally local-first: ports publish to 127.0.0.1 only, compose
uses a throwaway named volume, and cleanup runs `docker compose down -v`.
If Docker or the compose plugin is unavailable, the script reports a skip and
exits zero so non-container developer machines can still run the normal suite.
"""

from __future__ import annotations

import json
import shutil
import subprocess
import sys
import tempfile
import textwrap
import time
import urllib.request
from pathlib import Path


def run(args: list[str], *, check: bool = True) -> subprocess.CompletedProcess[str]:
    print("+", " ".join(args), flush=True)
    return subprocess.run(args, text=True, check=check, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)


def docker_compose_available() -> bool:
    if shutil.which("docker") is None:
        print("SKIP: docker executable not found")
        return False
    probe = run(["docker", "compose", "version"], check=False)
    if probe.returncode != 0:
        print("SKIP: docker compose plugin unavailable")
        print(probe.stdout)
        return False
    return True


def wait_for_health(url: str, timeout_seconds: float = 60.0) -> dict:
    deadline = time.time() + timeout_seconds
    last_error: Exception | None = None
    while time.time() < deadline:
        try:
            with urllib.request.urlopen(url, timeout=2.0) as response:  # noqa: S310 local loopback only
                payload = json.loads(response.read().decode("utf-8"))
                if payload.get("status") == "ok":
                    return payload
        except Exception as exc:  # pragma: no cover - diagnostic path
            last_error = exc
        time.sleep(1.0)
    raise RuntimeError(f"health check {url} did not become ok: {last_error}")


def transient_build_environment_failure(output: str) -> bool:
    markers = ["proxy.golang.org", "i/o timeout", "temporary failure", "no such host", "network is unreachable"]
    lowered = output.lower()
    return any(marker in lowered for marker in markers)


def write_local_binary_compose_override(tmpdir: Path) -> Path:
    run(["go", "build", "-o", str(tmpdir / "goncho-server"), "./cmd/goncho-server"])
    (tmpdir / "Dockerfile").write_text(
        textwrap.dedent(
            """
            FROM debian:bookworm-slim
            RUN useradd --system --uid 10001 --home-dir /data --create-home goncho
            COPY goncho-server /usr/local/bin/goncho-server
            RUN mkdir -p /data && chown -R goncho:goncho /data
            USER goncho
            WORKDIR /data
            EXPOSE 8765
            VOLUME ["/data"]
            HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=6 CMD ["/usr/local/bin/goncho-server", "health", "-db", "/data/goncho.db"]
            ENTRYPOINT ["/usr/local/bin/goncho-server"]
            """
        ).strip()
        + "\n"
    )
    override = tmpdir / "compose.override.yml"
    override.write_text(
        textwrap.dedent(
            f"""
            services:
              goncho-server:
                build:
                  context: {tmpdir}
                  dockerfile: Dockerfile
            """
        ).strip()
        + "\n"
    )
    return override


def main() -> int:
    if not docker_compose_available():
        return 0
    with tempfile.TemporaryDirectory(prefix="goncho-compose-smoke-") as raw_tmp:
        tmpdir = Path(raw_tmp)
        compose = ["docker", "compose"]
        try:
            up = run(compose + ["up", "-d", "--build"], check=False)
            if up.returncode != 0:
                print(up.stdout)
                if not transient_build_environment_failure(up.stdout):
                    return up.returncode
                print("FALLBACK: docker build could not reach module proxy; building local goncho-server binary for compose smoke")
                cleanup = run(compose + ["down", "-v"], check=False)
                print(cleanup.stdout)
                override = write_local_binary_compose_override(tmpdir)
                compose = ["docker", "compose", "-f", "docker-compose.yml", "-f", str(override)]
                up = run(compose + ["up", "-d", "--build"], check=False)
                if up.returncode != 0:
                    print(up.stdout)
                    if transient_build_environment_failure(up.stdout):
                        print("SKIP: docker compose fallback also needs unavailable network/base image access")
                        return 0
                    return up.returncode
            health = wait_for_health("http://127.0.0.1:8765/health")
            print(json.dumps({"health": health.get("status"), "version": health.get("version")}, sort_keys=True))
            demo = run(compose + ["exec", "-T", "goncho-server", "/usr/local/bin/goncho-server", "demo", "-db", "/data/goncho.db"])
            print(demo.stdout)
        finally:
            cleanup = run(compose + ["down", "-v"], check=False)
            print(cleanup.stdout)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
