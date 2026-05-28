#!/usr/bin/env node
import fs from "node:fs";
import os from "node:os";
import path from "node:path";

const args = process.argv.slice(2);
const rootArg = readArg("--root") ?? path.join(os.homedir(), ".gormes");
const root = path.resolve(rootArg.replace(/^~(?=$|\/)/, os.homedir()));
const now = Date.now();
const findings = [];
const facts = [];

function readArg(name) {
  const index = args.indexOf(name);
  return index === -1 ? undefined : args[index + 1];
}
function add(severity, pathLabel, message, detail = "") { findings.push({ severity, path: pathLabel, message, detail }); }
function fact(label, value) { facts.push({ label, value }); }
function rel(file) { return path.relative(root, file).split(path.sep).join("/") || "."; }
function exists(file) { try { return fs.existsSync(file); } catch { return false; } }
function stat(file) { try { return fs.statSync(file); } catch { return undefined; } }
function ageText(ms) {
  const mins = Math.round(ms / 60000);
  if (mins < 120) return `${mins}m`;
  const hours = Math.round(mins / 60);
  if (hours < 72) return `${hours}h`;
  return `${Math.round(hours / 24)}d`;
}
function fileMode(file) {
  const s = stat(file);
  return s ? `0${(s.mode & 0o777).toString(8)}` : "missing";
}
function parseJsonFile(file) {
  if (!exists(file)) return undefined;
  try {
    const parsed = JSON.parse(fs.readFileSync(file, "utf8"));
    fact(`${rel(file)} parse`, "ok");
    return parsed;
  } catch (error) {
    add("high", rel(file), "invalid JSON", error.message);
    return undefined;
  }
}
function parseJsonl(file, maxRecentLines = 200) {
  if (!exists(file)) return;
  const lines = fs.readFileSync(file, "utf8").split(/\r?\n/).filter(Boolean);
  let bad = 0;
  const start = Math.max(0, lines.length - maxRecentLines);
  for (let i = start; i < lines.length; i += 1) {
    try { JSON.parse(lines[i]); } catch { bad += 1; }
  }
  fact(`${rel(file)} records`, `${lines.length}`);
  if (bad) add("medium", rel(file), `${bad} malformed JSONL records in last ${lines.length - start} non-empty lines`);
}
function countYamlMapEntries(text, sectionName) {
  const lines = text.split(/\r?\n/);
  const sectionIndex = lines.findIndex((line) => line === `${sectionName}:`);
  if (sectionIndex === -1) return 0;
  let count = 0;
  for (let i = sectionIndex + 1; i < lines.length; i += 1) {
    const line = lines[i];
    if (/^\S/.test(line)) break;
    if (/^\s{2}[^\s#][^:]*:\s*(?:\S.*)?$/.test(line)) count += 1;
  }
  return count;
}
function auditIndex(file) {
  if (!exists(file)) return add("medium", rel(file), "missing session index");
  const text = fs.readFileSync(file, "utf8");
  const sessionCount = countYamlMapEntries(text, "sessions");
  const lineageCount = countYamlMapEntries(text, "lineage");
  const updatedAt = text.match(/^updated_at:\s*(.+)$/m)?.[1]?.trim();
  fact(`${rel(file)} sessions`, `${sessionCount}`);
  fact(`${rel(file)} lineage_entries`, `${lineageCount}`);
  if (!/^# Auto-generated session index/m.test(text)) add("low", rel(file), "index does not include expected generated-file header");
  if (!updatedAt) return add("medium", rel(file), "missing updated_at");
  const t = Date.parse(updatedAt);
  if (!Number.isFinite(t)) return add("medium", rel(file), "unparseable updated_at", updatedAt);
  const age = now - t;
  fact(`${rel(file)} updated_at_age`, ageText(age));
  if (age > 1000 * 60 * 60 * 24 * 14) add("medium", rel(file), "session index older than 14 days", updatedAt);
}
function auditMemoryMarkdown(file) {
  if (!exists(file)) return;
  const s = stat(file);
  const text = fs.readFileSync(file, "utf8");
  const nonEmptyLines = text.split(/\r?\n/).filter((line) => line.trim()).length;
  const headings = text.split(/\r?\n/).filter((line) => /^#{1,6}\s+/.test(line)).length;
  const gonchoMentions = (text.match(/\bgoncho\b/gi) ?? []).length;
  const gormesMentions = (text.match(/\bgormes\b/gi) ?? []).length;
  fact(`${rel(file)} memory_size`, `${s.size} bytes`);
  fact(`${rel(file)} memory_age`, ageText(now - s.mtimeMs));
  fact(`${rel(file)} memory_lines`, `${nonEmptyLines}`);
  fact(`${rel(file)} memory_headings`, `${headings}`);
  fact(`${rel(file)} goncho_mentions`, `${gonchoMentions}`);
  fact(`${rel(file)} gormes_mentions`, `${gormesMentions}`);
  if (s.size === 0) add("medium", rel(file), "memory file is empty");
}
function pidFromFile(file) {
  const raw = fs.readFileSync(file, "utf8").trim();
  if (/^\d+$/.test(raw)) return { pid: raw, shape: "numeric" };
  try {
    const parsed = JSON.parse(raw);
    if (Number.isInteger(parsed.pid)) return { pid: String(parsed.pid), shape: "json" };
  } catch {}
  return { error: raw.slice(0, 80) };
}
function auditPid(file) {
  if (!exists(file)) return add("low", rel(file), "missing PID file");
  const info = pidFromFile(file);
  if (info.error) return add("high", rel(file), "PID file is neither numeric nor JSON with integer pid", info.error);
  if (info.shape === "json") fact(`${rel(file)} shape`, "json pid file");
  const proc = `/proc/${info.pid}`;
  if (!exists(proc)) return add("high", rel(file), "PID is not running", info.pid);
  let cmd = "";
  try { cmd = fs.readFileSync(path.join(proc, "cmdline"), "utf8").replace(/\0/g, " ").trim(); } catch {}
  const safeCmd = cmd.replace(/(token|auth|password|secret)=\S+/gi, "$1=<redacted>").slice(0, 180);
  fact(`${rel(file)} live_pid`, `${info.pid} ${safeCmd}`.trim());
  if (cmd && !/gormes|gateway|go\b/i.test(cmd)) add("medium", rel(file), "PID is live but command does not look like gormes/gateway", safeCmd);
}
function auditSecretPath(file) {
  if (!exists(file)) return;
  const mode = fileMode(file);
  fact(`${rel(file)} mode`, mode);
  if ((Number.parseInt(mode, 8) & 0o077) !== 0) add("critical", rel(file), "secret-bearing file is readable by group/other", mode);
}
function auditStale(file, severity, label, maxAgeMs) {
  if (!exists(file)) return;
  const s = stat(file);
  const age = now - s.mtimeMs;
  fact(`${rel(file)} age`, ageText(age));
  if (age > maxAgeMs) add(severity, rel(file), label, `mtime ${s.mtime.toISOString()}`);
}

if (!exists(root)) {
  console.error(`Gormes root not found: ${root}`);
  process.exit(2);
}

fact("root", root);
fact("audit_time", new Date(now).toISOString());
for (const p of ["memory.db", "sessions.db", "memory.db-wal", "memory.db-shm", "gateway.log"]) {
  const file = path.join(root, p);
  const s = stat(file);
  if (s) fact(`${rel(file)} size`, `${s.size} bytes`);
}
for (const p of [".env", "auth.json"]) auditSecretPath(path.join(root, p));
for (const p of ["gateway_state.json", "channel_directory_sources.json", "auth.json"]) parseJsonFile(path.join(root, p));
for (const p of ["tools/audit.jsonl", "subagents/runs.jsonl", "lifecycle/install.log.jsonl", "install.log.jsonl"]) parseJsonl(path.join(root, p));
for (const p of ["memory/MEMORY.md", "memory/USER.md", "workspace/memory/MEMORY.md", "workspace/memory/USER.md"]) auditMemoryMarkdown(path.join(root, p));

auditIndex(path.join(root, "sessions/index.yaml"));
auditPid(path.join(root, "gateway.pid"));

const profilesDir = path.join(root, "profiles");
if (exists(profilesDir)) {
  for (const entry of fs.readdirSync(profilesDir, { withFileTypes: true }).filter((e) => e.isDirectory()).sort((a, b) => a.name.localeCompare(b.name))) {
    const dir = path.join(profilesDir, entry.name);
    auditIndex(path.join(dir, "sessions/index.yaml"));
    auditPid(path.join(dir, "gateway.pid"));
    parseJsonFile(path.join(dir, "gateway_state.json"));
    for (const p of ["memory/MEMORY.md", "memory/USER.md"]) auditMemoryMarkdown(path.join(dir, p));
  }
}
for (const dir of [path.join(root, "gateway-locks"), path.join(root, "memory")]) {
  if (!exists(dir)) continue;
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    if (entry.isFile() && entry.name.endsWith(".lock")) auditStale(path.join(dir, entry.name), "medium", "lock file older than 24h; verify no writer before removing", 1000 * 60 * 60 * 24);
  }
}

const severityOrder = { critical: 0, high: 1, medium: 2, low: 3, info: 4 };
findings.sort((a, b) => severityOrder[a.severity] - severityOrder[b.severity] || a.path.localeCompare(b.path));
console.log("# Gormes profile session and memory audit for Goncho\n");
console.log("## Facts");
for (const item of facts) console.log(`- ${item.label}: ${item.value}`);
console.log("\n## Findings");
if (!findings.length) console.log("- none");
else for (const f of findings) console.log(`- [${f.severity}] ${f.path}: ${f.message}${f.detail ? ` (${f.detail})` : ""}`);
console.log("\n## Suggested next step");
if (findings.some((f) => ["critical", "high"].includes(f.severity))) console.log("Plan fixes only when runtime state blocks trustworthy Goncho session/memory evidence; back up state before mutation.");
else if (findings.length) console.log("Review medium/low hygiene findings, then convert session/memory signals into Goncho hypotheses or tests.");
else console.log("No deterministic runtime issues found; use the session/memory facts to shape Goncho improvement hypotheses.");
