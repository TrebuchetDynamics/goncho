package main

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	goncho "github.com/TrebuchetDynamics/goncho/service"
)

func TestSecurityReportMarksLeasesServerModeOnly(t *testing.T) {
	var stdout bytes.Buffer
	if err := run(context.Background(), config{Command: "security", ServerMode: goncho.ServerModeLocalOnly, Stdout: &stdout}); err != nil {
		t.Fatalf("run security: %v", err)
	}
	var report goncho.ServerModeSecurityRequirement
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if report.CapabilityMode != goncho.ServerModeLocalOnly || report.LeasesEnabled || report.SignalsEnabled || len(report.ServerModeOnly) == 0 {
		t.Fatalf("security report = %+v, want leases/signals server-mode only", report)
	}

	stdout.Reset()
	if err := run(context.Background(), config{Command: "security", ServerMode: goncho.ServerModeTeamEnabled, Stdout: &stdout}); err != nil {
		t.Fatalf("run security team-enabled: %v", err)
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode team report: %v", err)
	}
	if !report.LeasesEnabled || !report.SignalsEnabled || !report.EnforcementEnabled {
		t.Fatalf("team-enabled report = %+v, want distributed capabilities enabled", report)
	}
}
