#!/usr/bin/env python3
"""Print Goncho's next minor semver tag from local git tags.

Usage:
  python3 .agents/skills/goncho-release/scripts/next_minor.py
  python3 .agents/skills/goncho-release/scripts/next_minor.py --from v0.2.0
"""

from __future__ import annotations

import argparse
import re
import subprocess
import sys
from dataclasses import dataclass

TAG_RE = re.compile(r"^v(\d+)\.(\d+)\.(\d+)$")


@dataclass(frozen=True, order=True)
class Version:
    major: int
    minor: int
    patch: int

    @classmethod
    def parse(cls, tag: str) -> "Version | None":
        match = TAG_RE.match(tag.strip())
        if not match:
            return None
        return cls(*(int(part) for part in match.groups()))

    def tag(self) -> str:
        return f"v{self.major}.{self.minor}.{self.patch}"

    def next_minor(self) -> "Version":
        return Version(self.major, self.minor + 1, 0)


def git_tags() -> list[str]:
    try:
        out = subprocess.check_output(
            ["git", "tag", "--list", "v*"], text=True, stderr=subprocess.STDOUT
        )
    except subprocess.CalledProcessError as exc:
        print(exc.output, file=sys.stderr, end="")
        raise SystemExit(exc.returncode) from exc
    return [line.strip() for line in out.splitlines() if line.strip()]


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--from", dest="from_tag", help="base semver tag, e.g. v0.2.0")
    args = parser.parse_args()

    if args.from_tag:
        current = Version.parse(args.from_tag)
        if current is None:
            print(f"invalid semver tag: {args.from_tag}", file=sys.stderr)
            return 2
    else:
        versions = [v for tag in git_tags() if (v := Version.parse(tag)) is not None]
        if not versions:
            print("no semver tags matching vMAJOR.MINOR.PATCH found", file=sys.stderr)
            return 1
        current = max(versions)

    next_version = current.next_minor()
    print(f"current={current.tag()}")
    print(f"next_minor={next_version.tag()}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
