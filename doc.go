// Package goncho provides high-trust local memory for Go-native AI agent
// runtimes.
//
// Goncho is designed as an embedded memory kernel rather than a hosted memory
// service. It stores local evidence, derives scoped recall, assembles context,
// records review signals, and helps callers verify remembered claims before an
// agent acts on them.
//
// The core operating rule is evidence before belief: memory can orient an
// agent, but current evidence decides whether an action is safe.
package goncho
