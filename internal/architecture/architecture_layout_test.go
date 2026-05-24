package architecture

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestArchitectureLayoutScopedKeyImplementationLivesBehindInternalModule(t *testing.T) {
	root := repoRoot(t)

	implPath := filepath.Join(root, "internal", "scopedkey", "scopedkey.go")
	if _, err := os.Stat(implPath); err != nil {
		t.Fatalf("scoped-key implementation must live at %s: %v", implPath, err)
	}

	keysPath := filepath.Join(root, "keys.go")
	parsed, err := parser.ParseFile(token.NewFileSet(), keysPath, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("ParseFile(%s): %v", keysPath, err)
	}
	forbiddenImplementationImports := map[string]struct{}{
		"crypto/hmac":     {},
		"crypto/sha256":   {},
		"encoding/base64": {},
		"encoding/json":   {},
		"errors":          {},
		"fmt":             {},
		"strings":         {},
	}
	for _, imp := range parsed.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			t.Fatalf("Unquote(%s): %v", imp.Path.Value, err)
		}
		if _, forbidden := forbiddenImplementationImports[path]; forbidden {
			t.Fatalf("keys.go imports implementation package %q; keep root keys.go as a public facade and put implementation behind internal/scopedkey", path)
		}
	}
}

func TestArchitectureLayoutScopedKeyTestsLiveWithModule(t *testing.T) {
	root := repoRoot(t)

	moduleTestPath := filepath.Join(root, "internal", "scopedkey", "scopedkey_test.go")
	if _, err := os.Stat(moduleTestPath); err != nil {
		t.Fatalf("scoped-key behavior tests must live at %s: %v", moduleTestPath, err)
	}

	rootTestPath := filepath.Join(root, "keys_test.go")
	parsed, err := parser.ParseFile(token.NewFileSet(), rootTestPath, nil, 0)
	if err != nil {
		t.Fatalf("ParseFile(%s): %v", rootTestPath, err)
	}
	for _, decl := range parsed.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !strings.HasPrefix(fn.Name.Name, "Test") {
			continue
		}
		if !strings.HasPrefix(fn.Name.Name, "TestKeys_PublicFacade") {
			t.Fatalf("%s keeps %s in the root package; move pure scoped-key behavior tests to internal/scopedkey", rootTestPath, fn.Name.Name)
		}
	}
}

func TestArchitectureLayoutDynamicAgentRegistryImplementationLivesBehindInternalModule(t *testing.T) {
	root := repoRoot(t)

	implPath := filepath.Join(root, "internal", "dynamicagents", "registry.go")
	if _, err := os.Stat(implPath); err != nil {
		t.Fatalf("dynamic agent registry implementation must live at %s: %v", implPath, err)
	}

	facadePath := filepath.Join(root, "dynamic_agents.go")
	parsed, err := parser.ParseFile(token.NewFileSet(), facadePath, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("ParseFile(%s): %v", facadePath, err)
	}
	forbiddenImplementationImports := map[string]struct{}{
		"context": {},
		"errors":  {},
		"fmt":     {},
		"regexp":  {},
		"strings": {},
		"time":    {},
	}
	for _, imp := range parsed.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			t.Fatalf("Unquote(%s): %v", imp.Path.Value, err)
		}
		if _, forbidden := forbiddenImplementationImports[path]; forbidden {
			t.Fatalf("dynamic_agents.go imports implementation package %q; keep root dynamic_agents.go as a public facade and put implementation behind internal/dynamicagents", path)
		}
	}
}

func TestArchitectureLayoutDynamicAgentRegistryTestsLiveWithModule(t *testing.T) {
	root := repoRoot(t)

	moduleTestPath := filepath.Join(root, "internal", "dynamicagents", "registry_test.go")
	if _, err := os.Stat(moduleTestPath); err != nil {
		t.Fatalf("dynamic agent registry behavior tests must live at %s: %v", moduleTestPath, err)
	}

	rootTestPath := filepath.Join(root, "dynamic_agents_test.go")
	parsed, err := parser.ParseFile(token.NewFileSet(), rootTestPath, nil, 0)
	if err != nil {
		t.Fatalf("ParseFile(%s): %v", rootTestPath, err)
	}
	for _, decl := range parsed.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !strings.HasPrefix(fn.Name.Name, "Test") {
			continue
		}
		if !strings.HasPrefix(fn.Name.Name, "TestDynamicAgentRegistryPublicFacade") {
			t.Fatalf("%s keeps %s in the root package; move pure dynamic agent registry behavior tests to internal/dynamicagents", rootTestPath, fn.Name.Name)
		}
	}
}

func TestArchitectureLayoutReviewLogImplementationLivesBehindInternalModule(t *testing.T) {
	root := repoRoot(t)

	implPath := filepath.Join(root, "internal", "reviewlog", "log.go")
	if _, err := os.Stat(implPath); err != nil {
		t.Fatalf("review-log implementation must live at %s: %v", implPath, err)
	}

	facadePath := filepath.Join(root, "review.go")
	parsed, err := parser.ParseFile(token.NewFileSet(), facadePath, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("ParseFile(%s): %v", facadePath, err)
	}
	forbiddenImplementationImports := map[string]struct{}{
		"encoding/json": {},
		"hash/fnv":      {},
	}
	for _, imp := range parsed.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			t.Fatalf("Unquote(%s): %v", imp.Path.Value, err)
		}
		if _, forbidden := forbiddenImplementationImports[path]; forbidden {
			t.Fatalf("%s imports implementation package %q; keep root review.go as a public facade and put implementation behind internal/reviewlog", facadePath, path)
		}
	}
	content, err := os.ReadFile(facadePath)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", facadePath, err)
	}
	for _, token := range []string{"goncho_review_items", "func normalizeReviewItem", "func scanReviewItem", "func ensureReviewTable"} {
		if bytes.Contains(content, []byte(token)) {
			t.Fatalf("%s still contains review-log implementation token %q; move it behind internal/reviewlog", facadePath, token)
		}
	}
}

func TestArchitectureLayoutReviewLogTestsLiveWithModule(t *testing.T) {
	root := repoRoot(t)

	moduleTestPath := filepath.Join(root, "internal", "reviewlog", "log_test.go")
	if _, err := os.Stat(moduleTestPath); err != nil {
		t.Fatalf("review-log behavior tests must live at %s: %v", moduleTestPath, err)
	}

	rootTestPath := filepath.Join(root, "review_test.go")
	parsed, err := parser.ParseFile(token.NewFileSet(), rootTestPath, nil, 0)
	if err != nil {
		t.Fatalf("ParseFile(%s): %v", rootTestPath, err)
	}
	for _, decl := range parsed.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !strings.HasPrefix(fn.Name.Name, "Test") {
			continue
		}
		if !strings.HasPrefix(fn.Name.Name, "TestReviewPublicFacade") {
			t.Fatalf("%s keeps %s in the root package; move pure review-log behavior tests to internal/reviewlog", rootTestPath, fn.Name.Name)
		}
	}
}

func TestArchitectureLayoutMemoryPolicyImplementationLivesBehindInternalModule(t *testing.T) {
	root := repoRoot(t)

	implPath := filepath.Join(root, "internal", "memorypolicy", "policy.go")
	if _, err := os.Stat(implPath); err != nil {
		t.Fatalf("memory policy implementation must live at %s: %v", implPath, err)
	}

	for _, facade := range []string{"memory_tiers.go", "memory_acl.go"} {
		facadePath := filepath.Join(root, facade)
		parsed, err := parser.ParseFile(token.NewFileSet(), facadePath, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("ParseFile(%s): %v", facadePath, err)
		}
		forbiddenImplementationImports := map[string]struct{}{
			"fmt":     {},
			"strings": {},
		}
		for _, imp := range parsed.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				t.Fatalf("Unquote(%s): %v", imp.Path.Value, err)
			}
			if _, forbidden := forbiddenImplementationImports[path]; forbidden {
				t.Fatalf("%s imports implementation package %q; keep root memory policy files as public facades and put implementation behind internal/memorypolicy", facadePath, path)
			}
		}
		content, err := os.ReadFile(facadePath)
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", facadePath, err)
		}
		for _, token := range []string{"memory_acl", "goncho_memory_items", "strings.Join", "fmt.Sprintf", "switch MemoryTier", "func (q ACLQuery) ReadScopeSQL"} {
			if bytes.Contains(content, []byte(token)) {
				t.Fatalf("%s still contains memory policy implementation token %q; move it behind internal/memorypolicy", facadePath, token)
			}
		}
	}
}

func TestArchitectureLayoutMemoryPolicyTestsLiveWithModule(t *testing.T) {
	root := repoRoot(t)

	moduleTestPath := filepath.Join(root, "internal", "memorypolicy", "policy_test.go")
	if _, err := os.Stat(moduleTestPath); err != nil {
		t.Fatalf("memory policy behavior tests must live at %s: %v", moduleTestPath, err)
	}

	for _, rootTestFile := range []string{"memory_tiers_test.go", "memory_acl_test.go"} {
		rootTestPath := filepath.Join(root, rootTestFile)
		parsed, err := parser.ParseFile(token.NewFileSet(), rootTestPath, nil, 0)
		if err != nil {
			t.Fatalf("ParseFile(%s): %v", rootTestPath, err)
		}
		for _, decl := range parsed.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || !strings.HasPrefix(fn.Name.Name, "Test") {
				continue
			}
			if !strings.HasPrefix(fn.Name.Name, "TestMemoryPolicyPublicFacade") {
				t.Fatalf("%s keeps %s in the root package; move pure memory policy behavior tests to internal/memorypolicy", rootTestPath, fn.Name.Name)
			}
		}
	}
}

func TestArchitectureLayoutQueueStatusImplementationLivesBehindInternalModule(t *testing.T) {
	root := repoRoot(t)

	implPath := filepath.Join(root, "internal", "queuestatus", "status.go")
	if _, err := os.Stat(implPath); err != nil {
		t.Fatalf("queue status implementation must live at %s: %v", implPath, err)
	}

	facadePath := filepath.Join(root, "diagnostics.go")
	content, err := os.ReadFile(facadePath)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", facadePath, err)
	}
	for _, token := range []string{"FROM goncho_dreams", "GROUP BY status", "func readDreamQueueStatus", "func readDreamWorkUnitCounts", "func readDreamStatusEvidence", "func scanDreamIntent", "func effectiveQueueStatusConfig"} {
		if bytes.Contains(content, []byte(token)) {
			t.Fatalf("%s still contains queue status implementation token %q; move it behind internal/queuestatus", facadePath, token)
		}
	}
}

func TestArchitectureLayoutQueueStatusTestsLiveWithModule(t *testing.T) {
	root := repoRoot(t)

	moduleTestPath := filepath.Join(root, "internal", "queuestatus", "status_test.go")
	if _, err := os.Stat(moduleTestPath); err != nil {
		t.Fatalf("queue status behavior tests must live at %s: %v", moduleTestPath, err)
	}

	rootTestPath := filepath.Join(root, "queue_status_test.go")
	parsed, err := parser.ParseFile(token.NewFileSet(), rootTestPath, nil, 0)
	if err != nil {
		t.Fatalf("ParseFile(%s): %v", rootTestPath, err)
	}
	for _, decl := range parsed.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !strings.HasPrefix(fn.Name.Name, "Test") {
			continue
		}
		if !strings.HasPrefix(fn.Name.Name, "TestQueueStatusPublicFacade") {
			t.Fatalf("%s keeps %s in the root package; move pure queue status behavior tests to internal/queuestatus", rootTestPath, fn.Name.Name)
		}
	}
}

func TestArchitectureLayoutDreamSchedulerImplementationLivesBehindInternalModule(t *testing.T) {
	root := repoRoot(t)

	implPath := filepath.Join(root, "internal", "dreamscheduler", "scheduler.go")
	if _, err := os.Stat(implPath); err != nil {
		t.Fatalf("dream scheduler implementation must live at %s: %v", implPath, err)
	}

	facadePath := filepath.Join(root, "dream_scheduler.go")
	content, err := os.ReadFile(facadePath)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", facadePath, err)
	}
	for _, token := range []string{"INSERT INTO goncho_dreams", "UPDATE goncho_dreams", "FROM goncho_dreams", "FROM goncho_conclusions", "func readDreamEligibility", "func findActiveDreamIntent", "func latestCompletedDream", "func insertDreamIntent"} {
		if bytes.Contains(content, []byte(token)) {
			t.Fatalf("%s still contains dream scheduler implementation token %q; move it behind internal/dreamscheduler", facadePath, token)
		}
	}
}

func TestArchitectureLayoutDreamSchedulerTestsLiveWithModule(t *testing.T) {
	root := repoRoot(t)

	moduleTestPath := filepath.Join(root, "internal", "dreamscheduler", "scheduler_test.go")
	if _, err := os.Stat(moduleTestPath); err != nil {
		t.Fatalf("dream scheduler behavior tests must live at %s: %v", moduleTestPath, err)
	}

	rootTestPath := filepath.Join(root, "dream_scheduler_test.go")
	parsed, err := parser.ParseFile(token.NewFileSet(), rootTestPath, nil, 0)
	if err != nil {
		t.Fatalf("ParseFile(%s): %v", rootTestPath, err)
	}
	for _, decl := range parsed.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !strings.HasPrefix(fn.Name.Name, "Test") {
			continue
		}
		if !strings.HasPrefix(fn.Name.Name, "TestGonchoDreamPublicFacade") && !strings.HasPrefix(fn.Name.Name, "TestGonchoDreamContext") && !strings.HasPrefix(fn.Name.Name, "TestGonchoDreamQueueStatus") {
			t.Fatalf("%s keeps %s in the root package; move pure dream scheduler behavior tests to internal/dreamscheduler", rootTestPath, fn.Name.Name)
		}
	}
}

func TestArchitectureLayoutSkillLearningProposalsImplementationLivesBehindInternalModule(t *testing.T) {
	root := repoRoot(t)

	implPath := filepath.Join(root, "internal", "skillproposals", "proposals.go")
	if _, err := os.Stat(implPath); err != nil {
		t.Fatalf("skill-learning proposal implementation must live at %s: %v", implPath, err)
	}

	facadePath := filepath.Join(root, "skill_learning_proposals.go")
	parsed, err := parser.ParseFile(token.NewFileSet(), facadePath, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("ParseFile(%s): %v", facadePath, err)
	}
	forbiddenImplementationImports := map[string]struct{}{
		"encoding/json": {},
		"time":          {},
	}
	for _, imp := range parsed.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			t.Fatalf("Unquote(%s): %v", imp.Path.Value, err)
		}
		if _, forbidden := forbiddenImplementationImports[path]; forbidden {
			t.Fatalf("%s imports implementation package %q; keep root skill_learning_proposals.go as a public facade and put implementation behind internal/skillproposals", facadePath, path)
		}
	}
	content, err := os.ReadFile(facadePath)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", facadePath, err)
	}
	for _, token := range []string{"goncho_skill_learning_proposals", "func scanSkillLearningProposal", "func ensureSkillLearningProposalTable", "func validSkillLearningProposalStatus", "func reviewSkillLearningProposal"} {
		if bytes.Contains(content, []byte(token)) {
			t.Fatalf("%s still contains skill-learning proposal implementation token %q; move it behind internal/skillproposals", facadePath, token)
		}
	}
}

func TestArchitectureLayoutSkillLearningProposalsTestsLiveWithModule(t *testing.T) {
	root := repoRoot(t)

	moduleTestPath := filepath.Join(root, "internal", "skillproposals", "proposals_test.go")
	if _, err := os.Stat(moduleTestPath); err != nil {
		t.Fatalf("skill-learning proposal behavior tests must live at %s: %v", moduleTestPath, err)
	}

	rootTestPath := filepath.Join(root, "skill_learning_proposals_test.go")
	parsed, err := parser.ParseFile(token.NewFileSet(), rootTestPath, nil, 0)
	if err != nil {
		t.Fatalf("ParseFile(%s): %v", rootTestPath, err)
	}
	for _, decl := range parsed.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !strings.HasPrefix(fn.Name.Name, "Test") {
			continue
		}
		if !strings.HasPrefix(fn.Name.Name, "TestSkillLearningProposalPublicFacade") {
			t.Fatalf("%s keeps %s in the root package; move pure skill-learning proposal behavior tests to internal/skillproposals", rootTestPath, fn.Name.Name)
		}
	}
}

func TestArchitectureLayoutObservationAuditImplementationLivesBehindInternalModule(t *testing.T) {
	root := repoRoot(t)

	implPath := filepath.Join(root, "internal", "observationlog", "log.go")
	if _, err := os.Stat(implPath); err != nil {
		t.Fatalf("observation/audit implementation must live at %s: %v", implPath, err)
	}

	for _, facade := range []string{"observations.go", "audit.go"} {
		facadePath := filepath.Join(root, facade)
		parsed, err := parser.ParseFile(token.NewFileSet(), facadePath, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("ParseFile(%s): %v", facadePath, err)
		}
		forbiddenImplementationImports := map[string]struct{}{
			"crypto/rand":   {},
			"crypto/sha256": {},
			"encoding/hex":  {},
			"encoding/json": {},
			"regexp":        {},
			"sort":          {},
			"strconv":       {},
			"unicode/utf8":  {},
		}
		for _, imp := range parsed.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				t.Fatalf("Unquote(%s): %v", imp.Path.Value, err)
			}
			if _, forbidden := forbiddenImplementationImports[path]; forbidden {
				t.Fatalf("%s imports implementation package %q; keep root observation/audit files as public facades and put implementation behind internal/observationlog", facadePath, path)
			}
		}
	}
}

func TestArchitectureLayoutObservationAuditTestsLiveWithModule(t *testing.T) {
	root := repoRoot(t)

	moduleTestPath := filepath.Join(root, "internal", "observationlog", "log_test.go")
	if _, err := os.Stat(moduleTestPath); err != nil {
		t.Fatalf("observation/audit behavior tests must live at %s: %v", moduleTestPath, err)
	}

	rootTestPath := filepath.Join(root, "observations_test.go")
	parsed, err := parser.ParseFile(token.NewFileSet(), rootTestPath, nil, 0)
	if err != nil {
		t.Fatalf("ParseFile(%s): %v", rootTestPath, err)
	}
	for _, decl := range parsed.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !strings.HasPrefix(fn.Name.Name, "Test") {
			continue
		}
		if !strings.HasPrefix(fn.Name.Name, "TestObservationsPublicFacade") && !strings.HasPrefix(fn.Name.Name, "TestRunMigrationsCreatesObservation") {
			t.Fatalf("%s keeps %s in the root package; move pure observation/audit behavior tests to internal/observationlog", rootTestPath, fn.Name.Name)
		}
	}
}

func TestArchitectureLayoutImportanceScoringImplementationLivesBehindInternalModule(t *testing.T) {
	root := repoRoot(t)

	implPath := filepath.Join(root, "internal", "importance", "importance.go")
	if _, err := os.Stat(implPath); err != nil {
		t.Fatalf("importance scoring implementation must live at %s: %v", implPath, err)
	}

	for _, facade := range []string{"importance_scorer.go", "decay.go"} {
		facadePath := filepath.Join(root, facade)
		parsed, err := parser.ParseFile(token.NewFileSet(), facadePath, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("ParseFile(%s): %v", facadePath, err)
		}
		forbiddenImplementationImports := map[string]struct{}{
			"math":    {},
			"sort":    {},
			"strings": {},
		}
		for _, imp := range parsed.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				t.Fatalf("Unquote(%s): %v", imp.Path.Value, err)
			}
			if _, forbidden := forbiddenImplementationImports[path]; forbidden {
				t.Fatalf("%s imports implementation package %q; keep root importance scoring files as public facades and put implementation behind internal/importance", facadePath, path)
			}
		}
	}
}

func TestArchitectureLayoutImportanceScoringTestsLiveWithModule(t *testing.T) {
	root := repoRoot(t)

	moduleTestPath := filepath.Join(root, "internal", "importance", "importance_test.go")
	if _, err := os.Stat(moduleTestPath); err != nil {
		t.Fatalf("importance scoring behavior tests must live at %s: %v", moduleTestPath, err)
	}

	rootTestPath := filepath.Join(root, "importance_scorer_test.go")
	parsed, err := parser.ParseFile(token.NewFileSet(), rootTestPath, nil, 0)
	if err != nil {
		t.Fatalf("ParseFile(%s): %v", rootTestPath, err)
	}
	for _, decl := range parsed.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !strings.HasPrefix(fn.Name.Name, "Test") {
			continue
		}
		if !strings.HasPrefix(fn.Name.Name, "TestImportancePublicFacade") {
			t.Fatalf("%s keeps %s in the root package; move pure importance-scoring behavior tests to internal/importance", rootTestPath, fn.Name.Name)
		}
	}
}

func TestArchitectureLayoutMemoryToolsImplementationLivesBehindInternalModule(t *testing.T) {
	root := repoRoot(t)

	implPath := filepath.Join(root, "internal", "memorytools", "tools.go")
	if _, err := os.Stat(implPath); err != nil {
		t.Fatalf("memory tools implementation must live at %s: %v", implPath, err)
	}

	facadePath := filepath.Join(root, "memory_tools.go")
	parsed, err := parser.ParseFile(token.NewFileSet(), facadePath, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("ParseFile(%s): %v", facadePath, err)
	}
	forbiddenImplementationImports := map[string]struct{}{
		"errors":  {},
		"fmt":     {},
		"strings": {},
		"github.com/TrebuchetDynamics/goncho/memory": {},
	}
	for _, imp := range parsed.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			t.Fatalf("Unquote(%s): %v", imp.Path.Value, err)
		}
		if _, forbidden := forbiddenImplementationImports[path]; forbidden {
			t.Fatalf("memory_tools.go imports implementation package %q; keep root memory_tools.go as a public facade and put implementation behind internal/memorytools", path)
		}
	}
}

func TestArchitectureLayoutMemoryToolsTestsLiveWithModule(t *testing.T) {
	root := repoRoot(t)

	moduleTestPath := filepath.Join(root, "internal", "memorytools", "tools_test.go")
	if _, err := os.Stat(moduleTestPath); err != nil {
		t.Fatalf("memory tool behavior tests must live at %s: %v", moduleTestPath, err)
	}

	rootTestPath := filepath.Join(root, "memory_tools_test.go")
	parsed, err := parser.ParseFile(token.NewFileSet(), rootTestPath, nil, 0)
	if err != nil {
		t.Fatalf("ParseFile(%s): %v", rootTestPath, err)
	}
	for _, decl := range parsed.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !strings.HasPrefix(fn.Name.Name, "Test") {
			continue
		}
		if !strings.HasPrefix(fn.Name.Name, "TestMemoryToolsPublicFacade") {
			t.Fatalf("%s keeps %s in the root package; move pure memory-tool behavior tests to internal/memorytools", rootTestPath, fn.Name.Name)
		}
	}
}

func TestArchitectureLayoutLocalMarkdownMemoryImplementationLivesBehindInternalModule(t *testing.T) {
	root := repoRoot(t)

	implPath := filepath.Join(root, "internal", "localmarkdown", "store.go")
	if _, err := os.Stat(implPath); err != nil {
		t.Fatalf("local markdown memory implementation must live at %s: %v", implPath, err)
	}

	facadePath := filepath.Join(root, "local_markdown_memory.go")
	parsed, err := parser.ParseFile(token.NewFileSet(), facadePath, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("ParseFile(%s): %v", facadePath, err)
	}
	forbiddenImplementationImports := map[string]struct{}{
		"encoding/json": {},
		"errors":        {},
		"fmt":           {},
		"os":            {},
		"strings":       {},
		"time":          {},
		"github.com/TrebuchetDynamics/goncho/memory": {},
	}
	for _, imp := range parsed.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			t.Fatalf("Unquote(%s): %v", imp.Path.Value, err)
		}
		if _, forbidden := forbiddenImplementationImports[path]; forbidden {
			t.Fatalf("local_markdown_memory.go imports implementation package %q; keep root local_markdown_memory.go as a public facade and put implementation behind internal/localmarkdown", path)
		}
	}
}

func TestArchitectureLayoutLocalMarkdownMemoryTestsLiveWithModule(t *testing.T) {
	root := repoRoot(t)

	moduleTestPath := filepath.Join(root, "internal", "localmarkdown", "store_test.go")
	if _, err := os.Stat(moduleTestPath); err != nil {
		t.Fatalf("local markdown memory behavior tests must live at %s: %v", moduleTestPath, err)
	}

	rootTestPath := filepath.Join(root, "local_markdown_mcp_test.go")
	parsed, err := parser.ParseFile(token.NewFileSet(), rootTestPath, nil, 0)
	if err != nil {
		t.Fatalf("ParseFile(%s): %v", rootTestPath, err)
	}
	for _, decl := range parsed.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !strings.HasPrefix(fn.Name.Name, "Test") {
			continue
		}
		if !strings.HasPrefix(fn.Name.Name, "TestLocalMarkdownMemoryPublicFacade") {
			t.Fatalf("%s keeps %s in the root package; move pure local markdown memory behavior tests to internal/localmarkdown", rootTestPath, fn.Name.Name)
		}
	}
}

func TestArchitectureLayoutFileImportImplementationLivesBehindInternalModule(t *testing.T) {
	root := repoRoot(t)

	implPath := filepath.Join(root, "internal", "fileimport", "import.go")
	if _, err := os.Stat(implPath); err != nil {
		t.Fatalf("file import implementation must live at %s: %v", implPath, err)
	}

	facadePath := filepath.Join(root, "file_import.go")
	parsed, err := parser.ParseFile(token.NewFileSet(), facadePath, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("ParseFile(%s): %v", facadePath, err)
	}
	forbiddenImplementationImports := map[string]struct{}{
		"bytes":         {},
		"crypto/rand":   {},
		"encoding/hex":  {},
		"encoding/json": {},
		"errors":        {},
		"fmt":           {},
		"mime":          {},
		"strings":       {},
		"time":          {},
		"unicode/utf16": {},
		"unicode/utf8":  {},
	}
	for _, imp := range parsed.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			t.Fatalf("Unquote(%s): %v", imp.Path.Value, err)
		}
		if _, forbidden := forbiddenImplementationImports[path]; forbidden {
			t.Fatalf("file_import.go imports implementation package %q; keep root file_import.go as a public facade and put implementation behind internal/fileimport", path)
		}
	}
}

func TestArchitectureLayoutFileImportTestsLiveWithModule(t *testing.T) {
	root := repoRoot(t)

	moduleTestPath := filepath.Join(root, "internal", "fileimport", "import_test.go")
	if _, err := os.Stat(moduleTestPath); err != nil {
		t.Fatalf("file import behavior tests must live at %s: %v", moduleTestPath, err)
	}

	rootTestPath := filepath.Join(root, "file_import_test.go")
	parsed, err := parser.ParseFile(token.NewFileSet(), rootTestPath, nil, 0)
	if err != nil {
		t.Fatalf("ParseFile(%s): %v", rootTestPath, err)
	}
	for _, decl := range parsed.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !strings.HasPrefix(fn.Name.Name, "Test") {
			continue
		}
		if !strings.HasPrefix(fn.Name.Name, "TestService_ImportFilePublicFacade") {
			t.Fatalf("%s keeps %s in the root package; move pure file import behavior tests to internal/fileimport", rootTestPath, fn.Name.Name)
		}
	}
}

func TestArchitectureLayoutWebhooksImplementationLivesBehindInternalModule(t *testing.T) {
	root := repoRoot(t)

	for _, moduleFile := range []string{"endpoints.go", "delivery.go"} {
		implPath := filepath.Join(root, "internal", "webhooks", moduleFile)
		if _, err := os.Stat(implPath); err != nil {
			t.Fatalf("webhook implementation must live at %s: %v", implPath, err)
		}
	}

	for _, facade := range []struct {
		path      string
		forbidden map[string]struct{}
	}{
		{
			path: filepath.Join(root, "webhooks.go"),
			forbidden: map[string]struct{}{
				"crypto/hmac":   {},
				"crypto/rand":   {},
				"crypto/sha256": {},
				"encoding/hex":  {},
				"errors":        {},
				"fmt":           {},
				"net":           {},
				"net/url":       {},
				"strings":       {},
				"time":          {},
			},
		},
		{
			path: filepath.Join(root, "webhook_delivery.go"),
			forbidden: map[string]struct{}{
				"bytes":         {},
				"encoding/json": {},
				"errors":        {},
				"fmt":           {},
				"net/url":       {},
				"strings":       {},
				"time":          {},
			},
		},
	} {
		parsed, err := parser.ParseFile(token.NewFileSet(), facade.path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("ParseFile(%s): %v", facade.path, err)
		}
		for _, imp := range parsed.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				t.Fatalf("Unquote(%s): %v", imp.Path.Value, err)
			}
			if _, forbidden := facade.forbidden[path]; forbidden {
				t.Fatalf("%s imports implementation package %q; keep root webhook files as public facades and put implementation behind internal/webhooks", facade.path, path)
			}
		}
	}
}

func TestArchitectureLayoutWebhooksTestsLiveWithModule(t *testing.T) {
	root := repoRoot(t)

	for _, moduleTestFile := range []string{"endpoints_test.go", "delivery_test.go"} {
		moduleTestPath := filepath.Join(root, "internal", "webhooks", moduleTestFile)
		if _, err := os.Stat(moduleTestPath); err != nil {
			t.Fatalf("webhook behavior tests must live at %s: %v", moduleTestPath, err)
		}
	}

	for _, rootTest := range []struct {
		file   string
		prefix string
	}{
		{file: "webhooks_test.go", prefix: "TestWebhooksPublicFacade"},
		{file: "webhook_delivery_test.go", prefix: "TestWebhookDeliveryPublicFacade"},
	} {
		rootTestPath := filepath.Join(root, rootTest.file)
		parsed, err := parser.ParseFile(token.NewFileSet(), rootTestPath, nil, 0)
		if err != nil {
			t.Fatalf("ParseFile(%s): %v", rootTestPath, err)
		}
		for _, decl := range parsed.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || !strings.HasPrefix(fn.Name.Name, "Test") {
				continue
			}
			if !strings.HasPrefix(fn.Name.Name, rootTest.prefix) {
				t.Fatalf("%s keeps %s in the root package; move pure webhook behavior tests to internal/webhooks", rootTestPath, fn.Name.Name)
			}
		}
	}
}

func TestArchitectureLayoutPluginRuntimeImplementationLivesBehindInternalModule(t *testing.T) {
	root := repoRoot(t)

	for _, moduleFile := range []string{"config.go", "write.go"} {
		implPath := filepath.Join(root, "internal", "pluginruntime", moduleFile)
		if _, err := os.Stat(implPath); err != nil {
			t.Fatalf("plugin runtime implementation must live at %s: %v", implPath, err)
		}
	}

	for _, facade := range []struct {
		path      string
		forbidden map[string]struct{}
	}{
		{
			path: filepath.Join(root, "session_config.go"),
			forbidden: map[string]struct{}{
				"crypto/sha256": {},
				"encoding/hex":  {},
				"fmt":           {},
				"net/url":       {},
				"path/filepath": {},
				"regexp":        {},
				"strconv":       {},
				"strings":       {},
			},
		},
		{
			path: filepath.Join(root, "async_write.go"),
			forbidden: map[string]struct{}{
				"strconv": {},
				"strings": {},
				"sync":    {},
			},
		},
	} {
		parsed, err := parser.ParseFile(token.NewFileSet(), facade.path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("ParseFile(%s): %v", facade.path, err)
		}
		for _, imp := range parsed.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				t.Fatalf("Unquote(%s): %v", imp.Path.Value, err)
			}
			if _, forbidden := facade.forbidden[path]; forbidden {
				t.Fatalf("%s imports implementation package %q; keep root plugin runtime files as public facades and put implementation behind internal/pluginruntime", facade.path, path)
			}
		}
	}
}

func TestArchitectureLayoutPluginRuntimeTestsLiveWithModule(t *testing.T) {
	root := repoRoot(t)

	for _, moduleTestFile := range []string{"config_test.go", "write_test.go"} {
		moduleTestPath := filepath.Join(root, "internal", "pluginruntime", moduleTestFile)
		if _, err := os.Stat(moduleTestPath); err != nil {
			t.Fatalf("plugin runtime behavior tests must live at %s: %v", moduleTestPath, err)
		}
	}

	for _, rootTestFile := range []string{"plugin_session_config_test.go", "async_write_test.go"} {
		rootTestPath := filepath.Join(root, rootTestFile)
		parsed, err := parser.ParseFile(token.NewFileSet(), rootTestPath, nil, 0)
		if err != nil {
			t.Fatalf("ParseFile(%s): %v", rootTestPath, err)
		}
		for _, decl := range parsed.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || !strings.HasPrefix(fn.Name.Name, "Test") {
				continue
			}
			if !strings.HasPrefix(fn.Name.Name, "TestPluginRuntimePublicFacade") {
				t.Fatalf("%s keeps %s in the root package; move pure plugin runtime behavior tests to internal/pluginruntime", rootTestPath, fn.Name.Name)
			}
		}
	}
}

func TestArchitectureLayoutHostIntegrationImplementationLivesBehindInternalModule(t *testing.T) {
	root := repoRoot(t)

	implPath := filepath.Join(root, "internal", "hostintegration", "integration.go")
	if _, err := os.Stat(implPath); err != nil {
		t.Fatalf("host-integration implementation must live at %s: %v", implPath, err)
	}

	facadePath := filepath.Join(root, "host_integration.go")
	parsed, err := parser.ParseFile(token.NewFileSet(), facadePath, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("ParseFile(%s): %v", facadePath, err)
	}
	forbiddenImplementationImports := map[string]struct{}{
		"fmt":     {},
		"strings": {},
	}
	for _, imp := range parsed.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			t.Fatalf("Unquote(%s): %v", imp.Path.Value, err)
		}
		if _, forbidden := forbiddenImplementationImports[path]; forbidden {
			t.Fatalf("host_integration.go imports implementation package %q; keep root host_integration.go as a public facade and put implementation behind internal/hostintegration", path)
		}
	}
}

func TestArchitectureLayoutHostIntegrationTestsLiveWithModule(t *testing.T) {
	root := repoRoot(t)

	moduleTestPath := filepath.Join(root, "internal", "hostintegration", "integration_test.go")
	if _, err := os.Stat(moduleTestPath); err != nil {
		t.Fatalf("host-integration behavior tests must live at %s: %v", moduleTestPath, err)
	}

	rootTestPath := filepath.Join(root, "host_integration_test.go")
	parsed, err := parser.ParseFile(token.NewFileSet(), rootTestPath, nil, 0)
	if err != nil {
		t.Fatalf("ParseFile(%s): %v", rootTestPath, err)
	}
	for _, decl := range parsed.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !strings.HasPrefix(fn.Name.Name, "Test") {
			continue
		}
		if !strings.HasPrefix(fn.Name.Name, "TestHostIntegrationPublicFacade") {
			t.Fatalf("%s keeps %s in the root package; move pure host-integration behavior tests to internal/hostintegration", rootTestPath, fn.Name.Name)
		}
	}
}

func TestArchitectureLayoutSillyTavernIntegrationLivesWithHostIntegrationModule(t *testing.T) {
	root := repoRoot(t)

	modulePath := filepath.Join(root, "internal", "hostintegration", "sillytavern.go")
	if _, err := os.Stat(modulePath); err != nil {
		t.Fatalf("SillyTavern host-integration implementation must live at %s: %v", modulePath, err)
	}
	moduleTestPath := filepath.Join(root, "internal", "hostintegration", "sillytavern_test.go")
	if _, err := os.Stat(moduleTestPath); err != nil {
		t.Fatalf("SillyTavern host-integration behavior tests must live at %s: %v", moduleTestPath, err)
	}

	facadePath := filepath.Join(root, "sillytavern_mapping.go")
	parsedFacade, err := parser.ParseFile(token.NewFileSet(), facadePath, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("ParseFile(%s): %v", facadePath, err)
	}
	for _, imp := range parsedFacade.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			t.Fatalf("Unquote(%s): %v", imp.Path.Value, err)
		}
		if path == "strings" {
			t.Fatalf("sillytavern_mapping.go imports implementation package %q; keep root sillytavern_mapping.go as a public facade and put implementation behind internal/hostintegration", path)
		}
	}

	rootTestPath := filepath.Join(root, "sillytavern_mapping_test.go")
	parsedTests, err := parser.ParseFile(token.NewFileSet(), rootTestPath, nil, 0)
	if err != nil {
		t.Fatalf("ParseFile(%s): %v", rootTestPath, err)
	}
	for _, decl := range parsedTests.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !strings.HasPrefix(fn.Name.Name, "Test") {
			continue
		}
		if !strings.HasPrefix(fn.Name.Name, "TestSillyTavernPublicFacade") {
			t.Fatalf("%s keeps %s in the root package; move pure SillyTavern host-integration behavior tests to internal/hostintegration", rootTestPath, fn.Name.Name)
		}
	}
}

func TestArchitectureLayoutSearchFilterImplementationLivesBehindInternalModule(t *testing.T) {
	root := repoRoot(t)

	implPath := filepath.Join(root, "internal", "searchfilter", "filter.go")
	if _, err := os.Stat(implPath); err != nil {
		t.Fatalf("search-filter implementation must live at %s: %v", implPath, err)
	}

	filterPath := filepath.Join(root, "filter.go")
	parsed, err := parser.ParseFile(token.NewFileSet(), filterPath, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("ParseFile(%s): %v", filterPath, err)
	}
	forbiddenImplementationImports := map[string]struct{}{
		"fmt":     {},
		"slices":  {},
		"strings": {},
	}
	for _, imp := range parsed.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			t.Fatalf("Unquote(%s): %v", imp.Path.Value, err)
		}
		if _, forbidden := forbiddenImplementationImports[path]; forbidden {
			t.Fatalf("filter.go imports implementation package %q; keep root filter.go as a package facade and put implementation behind internal/searchfilter", path)
		}
	}
}

func TestArchitectureLayoutSearchFilterTestsLiveWithModule(t *testing.T) {
	root := repoRoot(t)

	moduleTestPath := filepath.Join(root, "internal", "searchfilter", "filter_test.go")
	if _, err := os.Stat(moduleTestPath); err != nil {
		t.Fatalf("search-filter grammar/compiler tests must live at %s: %v", moduleTestPath, err)
	}

	rootTestPath := filepath.Join(root, "filter_grammar_test.go")
	parsed, err := parser.ParseFile(token.NewFileSet(), rootTestPath, nil, 0)
	if err != nil {
		t.Fatalf("ParseFile(%s): %v", rootTestPath, err)
	}
	for _, decl := range parsed.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !strings.HasPrefix(fn.Name.Name, "Test") {
			continue
		}
		if !strings.HasPrefix(fn.Name.Name, "TestService_Search") {
			t.Fatalf("%s keeps %s in the root package; move pure search-filter module tests to internal/searchfilter", rootTestPath, fn.Name.Name)
		}
	}
}

func TestArchitectureLayoutWorkspaceDetectionLivesInWorkspacePackage(t *testing.T) {
	root := repoRoot(t)

	for _, rootFile := range []string{"workspace_detection.go", "workspace_detection_test.go"} {
		path := filepath.Join(root, rootFile)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("%s must move out of the root package into the workspace package", path)
		}
	}
	for _, packageFile := range []string{"workspace/workspace.go", "workspace/workspace_test.go"} {
		path := filepath.Join(root, packageFile)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("workspace detection package file missing at %s: %v", path, err)
		}
	}
}

func TestArchitectureLayoutWorkspaceDefaultsHaveSingleOwner(t *testing.T) {
	root := repoRoot(t)

	requireConstExpr(t, filepath.Join(root, "workspace_facade.go"), "DefaultWorkspaceID", "workspacepkg.DefaultWorkspaceID")
	requireConstExpr(t, filepath.Join(root, "workspace_facade.go"), "GlobalWorkspaceID", "workspacepkg.GlobalWorkspaceID")
	forbidConstExpr(t, filepath.Join(root, "topology.go"), "DefaultWorkspaceID")
	requireConstExpr(t, filepath.Join(root, "topology.go"), "EvidenceDefaultWorkspace", `"default_workspace:" + DefaultWorkspaceID`)
	requireConstExpr(t, filepath.Join(root, "integration", "gormes", "adapter.go"), "DefaultWorkspaceID", "goncho.DefaultWorkspaceID")
}

func requireConstExpr(t *testing.T, path, name, want string) {
	t.Helper()
	got, ok := constExpr(t, path, name)
	if !ok {
		t.Fatalf("%s must define const %s as %s", path, name, want)
	}
	if got != want {
		t.Fatalf("%s const %s = %s, want %s", path, name, got, want)
	}
}

func forbidConstExpr(t *testing.T, path, name string) {
	t.Helper()
	if got, ok := constExpr(t, path, name); ok {
		t.Fatalf("%s must not define const %s = %s; keep workspace defaults behind workspace_facade.go", path, name, got)
	}
}

func constExpr(t *testing.T, path, name string) (string, bool) {
	t.Helper()
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatalf("ParseFile(%s): %v", path, err)
	}
	for _, decl := range parsed.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			valueSpec := spec.(*ast.ValueSpec)
			for i, ident := range valueSpec.Names {
				if ident.Name != name {
					continue
				}
				if len(valueSpec.Values) <= i {
					t.Fatalf("%s const %s must have an explicit value", path, name)
				}
				var out bytes.Buffer
				if err := format.Node(&out, fset, valueSpec.Values[i]); err != nil {
					t.Fatalf("format const %s in %s: %v", name, path, err)
				}
				return out.String(), true
			}
		}
	}
	return "", false
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find go.mod above %s", dir)
		}
		dir = parent
	}
}
