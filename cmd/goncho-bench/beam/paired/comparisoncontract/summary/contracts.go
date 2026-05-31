package summary

import "github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/paired/comparisoncontract/report"

// Report preserves the summary package report API while sharing the focused
// report contract with root compatibility and comparison helpers.
type Report = report.Report

// Stats preserves the summary package stats API while sharing the focused
// report contract with root compatibility and comparison helpers.
type Stats = report.Stats

// Row preserves the summary package row API while sharing the focused report
// contract with root compatibility and comparison helpers.
type Row = report.Row
