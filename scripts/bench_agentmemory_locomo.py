#!/usr/bin/env python3
"""LOCOMO agentmemory stable-ID adapter.

The Go benchmark harness owns scoring. This script emits retrieval JSONL only
when an agentmemory source checkout supports stable external_id/metadata
passthrough. Without such a checkout it fails closed as not comparable.
"""
from __future__ import annotations

import argparse
import json
import os
import platform
import subprocess
import tempfile
from collections import defaultdict
from pathlib import Path
from typing import Any

BACKEND = "agentmemory"
PR_COMMIT = "9b18a80c9d2839b025279978d3f4b5e1f9bc6e74"
SOURCE_LABEL = "rohitg00/agentmemory @agentmemory/agentmemory 0.9.20 stable external_id PR #583"
REASON_NO_SOURCE = (
    "not comparable: agentmemory source with stable external_id support was not provided; "
    "set AGENTMEMORY_SOURCE_DIR or pass --agentmemory-source"
)
REASON_CONTENT_ONLY = "content-only matching is rejected because LOCOMO contains duplicate content"


def load_jsonl(path: Path) -> list[dict[str, Any]]:
    rows: list[dict[str, Any]] = []
    with path.open("r", encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if line:
                rows.append(json.loads(line))
    return rows


def source_version(source: Path | None) -> dict[str, Any]:
    pkg: dict[str, Any] = {}
    if source and (source / "package.json").exists():
        try:
            raw = json.loads((source / "package.json").read_text(encoding="utf-8"))
            pkg = {"name": raw.get("name"), "version": raw.get("version")}
        except Exception as exc:
            pkg = {"error": str(exc)}
    commit = ""
    if source and (source / ".git").exists():
        try:
            commit = subprocess.check_output(["git", "-C", str(source), "rev-parse", "HEAD"], text=True).strip()
        except Exception:
            commit = ""
    return {"label": SOURCE_LABEL, "source_dir": str(source) if source else "", "commit": commit, "package_json": pkg}


def package_status(source: Path | None = None) -> dict[str, Any]:
    return {
        "python_version": platform.python_version(),
        "node_version": subprocess.check_output(["node", "--version"], text=True).strip(),
        "source": source_version(source),
        "install_commands": [
            "git clone https://github.com/rohitg00/agentmemory.git",
            f"cd agentmemory && git checkout {PR_COMMIT} && npm install --legacy-peer-deps",
        ],
    }


def content_collision_report(memories: list[dict[str, Any]]) -> dict[str, Any]:
    by_key: dict[tuple[str, str], list[str]] = defaultdict(list)
    by_content: dict[str, list[str]] = defaultdict(list)
    for m in memories:
        mid = str(m.get("memory_id", ""))
        content = str(m.get("content", ""))
        conv = str(m.get("conversation_id", ""))
        by_key[(conv, content)].append(mid)
        by_content[content].append(mid)
    duplicate_content = {k: v for k, v in by_content.items() if len(v) > 1}
    duplicate_composite = {f"{k[0]}\u241f{k[1]}": v for k, v in by_key.items() if len(v) > 1}
    return {
        "duplicate_content_count": len(duplicate_content),
        "duplicate_conversation_content_count": len(duplicate_composite),
        "collision_safe_content_only": len(duplicate_content) == 0,
        "collision_safe_conversation_content": len(duplicate_composite) == 0,
    }


def capability(source: Path | None) -> dict[str, Any]:
    comparable = bool(source and (source / "src/mcp/standalone.ts").exists() and (source / "node_modules/.bin/tsx").exists())
    reason = "" if comparable else REASON_NO_SOURCE
    return {
        "backend": BACKEND,
        "comparable": comparable,
        "reason": reason,
        "package": package_status(source),
        "id_strategy": "memory_save external_id plus metadata.memory_id passthrough returned by memory_smart_search",
        "required_contract": {
            "reset": "new InMemoryKV per conversation",
            "insert": "memory_save(content, external_id=memory_id, metadata={memory_id, conversation_id, session_id, turn_index})",
            "search": "memory_smart_search(query, limit) returning observation.external_id and observation.metadata.memory_id",
        },
    }


def write_not_comparable(out: Path | None, reason: str, source: Path | None, memories: Path | None = None, questions: Path | None = None) -> None:
    row = capability(source)
    row["comparable"] = False
    row["reason"] = reason
    if memories and memories.exists():
        row["collision_check"] = content_collision_report(load_jsonl(memories))
    if questions and questions.exists():
        row["question_count"] = len(load_jsonl(questions))
    line = json.dumps(row, sort_keys=True)
    if out:
        out.parent.mkdir(parents=True, exist_ok=True)
        out.write_text(line + "\n", encoding="utf-8")
    else:
        print(line)


def node_adapter_source() -> str:
    return r'''
import fs from 'node:fs';
import { InMemoryKV } from './src/mcp/in-memory-kv.ts';
import { handleToolCall } from './src/mcp/standalone.ts';

const [memPath, questionPath, outPath] = process.argv.slice(2);
function loadJsonl(path) {
  return fs.readFileSync(path, 'utf8').split(/\n/).filter(Boolean).map((line) => JSON.parse(line));
}
function indexableContent(mem) {
  const speaker = mem.speaker ? `${mem.speaker}: ` : '';
  return `${speaker}${mem.content || ''}`;
}
function extractMemoryId(hit) {
  if (!hit || typeof hit !== 'object') return '';
  const obs = hit.observation && typeof hit.observation === 'object' ? hit.observation : hit;
  const meta = obs.metadata && typeof obs.metadata === 'object' ? obs.metadata : {};
  return String(meta.memory_id || obs.external_id || '');
}
async function call(kv, tool, args) {
  const res = await handleToolCall(tool, args, kv);
  return JSON.parse(res.content[0].text);
}
const memories = loadJsonl(memPath);
const questions = loadJsonl(questionPath);
const byConv = new Map();
for (const mem of memories) {
  const conv = mem.conversation_id || '';
  if (!byConv.has(conv)) byConv.set(conv, []);
  byConv.get(conv).push(mem);
}
const questionsByConv = new Map();
for (const q of questions) {
  const conv = q.conversation_id || '';
  if (!questionsByConv.has(conv)) questionsByConv.set(conv, []);
  questionsByConv.get(conv).push(q);
}
const out = fs.createWriteStream(outPath, { encoding: 'utf8' });
for (const [conv, convQuestions] of questionsByConv.entries()) {
  const kv = new InMemoryKV();
  for (const mem of byConv.get(conv) || []) {
    await call(kv, 'memory_save', {
      content: indexableContent(mem),
      external_id: mem.memory_id,
      metadata: {
        memory_id: mem.memory_id,
        conversation_id: mem.conversation_id,
        session_id: mem.session_id,
        speaker: mem.speaker,
        turn_index: mem.turn_index,
        timestamp: mem.timestamp,
      },
    });
  }
  for (const q of convQuestions) {
    const result = await call(kv, 'memory_smart_search', { query: q.question, limit: 10 });
    const seen = new Set();
    const hits = [];
    for (const hit of result.results || []) {
      const memoryId = extractMemoryId(hit);
      if (!memoryId || seen.has(memoryId)) continue;
      seen.add(memoryId);
      const obs = hit.observation && typeof hit.observation === 'object' ? hit.observation : hit;
      hits.push({ memory_id: memoryId, score: Number(hit.score || 0), backend_raw_id: String(obs.id || ''), metadata: obs.metadata || {} });
    }
    out.write(JSON.stringify({ backend: 'agentmemory', question_id: q.question_id, comparable: true, results: hits }) + '\n');
  }
}
out.end();
'''


def run_agentmemory_adapter(source: Path, memories: Path, questions: Path, out: Path) -> None:
    if not (source / "src/mcp/standalone.ts").exists():
        raise RuntimeError(f"agentmemory source missing src/mcp/standalone.ts: {source}")
    if not (source / "node_modules/.bin/tsx").exists():
        raise RuntimeError(f"agentmemory source missing node_modules/.bin/tsx; run npm install --legacy-peer-deps in {source}")
    out.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.NamedTemporaryFile("w", suffix=".mjs", dir=source, delete=False, encoding="utf-8") as f:
        f.write(node_adapter_source())
        script = Path(f.name)
    try:
        subprocess.run(["npx", "tsx", str(script.name), str(memories.resolve()), str(questions.resolve()), str(out.resolve())], cwd=source, check=True)
    finally:
        try:
            script.unlink()
        except FileNotFoundError:
            pass


def smoke(source: Path | None) -> int:
    fixture = [
        {"memory_id": "m1", "conversation_id": "c1", "content": "duplicate text"},
        {"memory_id": "m2", "conversation_id": "c2", "content": "duplicate text"},
        {"memory_id": "m3", "conversation_id": "c1", "content": "unique text"},
    ]
    report = content_collision_report(fixture)
    ok = report["collision_safe_content_only"] is False and report["collision_safe_conversation_content"] is True
    if source and (source / "src/mcp/standalone.ts").exists() and (source / "node_modules/.bin/tsx").exists():
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            mem = root / "memories.jsonl"
            q = root / "questions.jsonl"
            out = root / "out.jsonl"
            mem.write_text(
                '\n'.join([
                    json.dumps({"memory_id": "m1", "conversation_id": "c1", "session_id": "s1", "speaker": "A", "turn_index": 1, "content": "duplicate text stable id"}),
                    json.dumps({"memory_id": "m2", "conversation_id": "c1", "session_id": "s1", "speaker": "B", "turn_index": 2, "content": "duplicate text stable id"}),
                ]) + '\n',
                encoding="utf-8",
            )
            q.write_text(json.dumps({"question_id": "q1", "conversation_id": "c1", "question": "stable id", "gold_memory_ids": ["m1"]}) + '\n', encoding="utf-8")
            run_agentmemory_adapter(source, mem, q, out)
            rows = load_jsonl(out)
            ids = [r["memory_id"] for r in rows[0].get("results", [])]
            ok = ok and "m1" in ids and "m2" in ids
    print(json.dumps({"backend": BACKEND, "smoke": ok, "comparable": bool(source), "collision_check": report, "package": package_status(source)}, indent=2, sort_keys=True))
    return 0 if ok else 1


def resolve_source(raw: str | None) -> Path | None:
    value = raw or os.environ.get("AGENTMEMORY_SOURCE_DIR", "")
    if not value:
        return None
    return Path(value).expanduser().resolve()


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--capability", action="store_true")
    parser.add_argument("--smoke", action="store_true")
    parser.add_argument("--memories", type=Path)
    parser.add_argument("--questions", type=Path)
    parser.add_argument("--out", type=Path)
    parser.add_argument("--agentmemory-source", type=str)
    args = parser.parse_args()
    source = resolve_source(args.agentmemory_source)
    if args.smoke:
        return smoke(source)
    if args.capability:
        print(json.dumps(capability(source), indent=2, sort_keys=True))
        return 0
    if not args.memories or not args.questions or not args.out:
        write_not_comparable(args.out, "not comparable: --memories, --questions, and --out are required", source, args.memories, args.questions)
        return 0
    if not source:
        write_not_comparable(args.out, REASON_NO_SOURCE, source, args.memories, args.questions)
        return 0
    try:
        run_agentmemory_adapter(source, args.memories, args.questions, args.out)
    except Exception as exc:
        write_not_comparable(args.out, f"not comparable: {exc}; {REASON_CONTENT_ONLY}", source, args.memories, args.questions)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
