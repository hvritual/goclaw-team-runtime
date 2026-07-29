package gateway

import (
	"encoding/json"
	"strings"
	"testing"

	dev "github.com/smallnest/goclaw/orchestratorlite"
	"github.com/smallnest/goclaw/teamcontrol"
	"github.com/smallnest/goclaw/workstation"
)

func TestDevelopmentExecutionPackRequiresCompleteWaveBinding(t *testing.T) {
	task := dev.Task{
		ID:               "task-wave-pack",
		ProjectID:        "project-pilot",
		RepositoryID:     "repo-pilot",
		PolicyBundleHash: strings.Repeat("a", 64),
		Title:            "Wave pack",
		Request: dev.RequestFrame{
			RawRequest: "Build a complete immutable execution pack.",
		},
		Goal: dev.GoalSpec{Objective: "Validate the Wave binding."},
		Plan: dev.PlanSpec{Milestones: []dev.Milestone{{
			ID: "pilot", Title: "Pilot", WorkItems: []dev.WorkItem{{
				ID:           "work-pilot",
				Title:        "Pilot",
				Instructions: "Validate.",
			}},
		}}},
		EvidencePlan: dev.EvidencePlan{Commands: []dev.CommandSpec{{
			Name: "verify",
			Argv: []string{"true"},
		}}},
		Scope: dev.ScopePolicy{AllowedPaths: []string{"gateway/**"}},
		Compile: dev.CompileRecord{
			Revision:            3,
			BaseRef:             "main",
			BaseCommit:          strings.Repeat("b", 40),
			ExecutionBundleHash: strings.Repeat("c", 64),
		},
		Wave: &dev.WaveBinding{
			WaveID:         "PILOT-W00",
			PlanRevision:   1,
			StepID:         "PILOT-W00-S03",
			PlanPath:       "docs/waves/pilot/plan-r001.md",
			RegistrySHA256: strings.Repeat("d", 64),
			PlanSHA256:     strings.Repeat("e", 64),
		},
	}
	pack, err := buildDevelopmentExecutionPack(task, teamcontrol.Repository{
		ID:        task.RepositoryID,
		RemoteURL: "https://example.invalid/pilot.git",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := validateDevelopmentExecutionPackWave(task, pack); err != nil {
		t.Fatalf("valid pack rejected: %v", err)
	}

	keys := []string{
		"wave_id",
		"wave_revision",
		"wave_step",
		"wave_plan_path",
		"wave_registry_sha256",
		"wave_plan_sha256",
	}
	for _, key := range keys {
		t.Run("missing-"+key, func(t *testing.T) {
			candidate := cloneExecutionPack(t, pack)
			delete(candidate.Metadata, key)
			if err := validateDevelopmentExecutionPackWave(
				task,
				candidate,
			); err == nil || !strings.Contains(err.Error(), key) {
				t.Fatalf("missing %s did not fail closed: %v", key, err)
			}
		})
	}
	t.Run("payload binding missing", func(t *testing.T) {
		candidate := cloneExecutionPack(t, pack)
		candidate.Payload = json.RawMessage(`{"wave":null}`)
		if err := validateDevelopmentExecutionPackWave(
			task,
			candidate,
		); err == nil || !strings.Contains(err.Error(), "Wave payload") {
			t.Fatalf("missing payload Wave did not fail closed: %v", err)
		}
	})
	t.Run("task binding missing", func(t *testing.T) {
		candidateTask := task
		candidateTask.Wave = nil
		if err := validateDevelopmentExecutionPackWave(
			candidateTask,
			pack,
		); err == nil || !strings.Contains(err.Error(), "missing its Wave") {
			t.Fatalf("missing task Wave did not fail closed: %v", err)
		}
	})
}

func cloneExecutionPack(
	t *testing.T,
	pack workstation.ExecutionPack,
) workstation.ExecutionPack {
	t.Helper()
	data, err := json.Marshal(pack)
	if err != nil {
		t.Fatal(err)
	}
	var result workstation.ExecutionPack
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestValidateTeamExecutionProfileFailsClosed(t *testing.T) {
	noRule := teamcontrol.ResolvedPolicy{Rules: map[string]json.RawMessage{}}
	if err := validateTeamExecutionProfile(
		noRule,
		workstation.ExecutionProfileStrict,
	); err != nil {
		t.Fatalf("default strict rejected: %v", err)
	}
	if err := validateTeamExecutionProfile(
		noRule,
		workstation.ExecutionProfileCodexDelegated,
	); err == nil {
		t.Fatal("delegated profile passed without an explicit policy")
	}

	allowed := teamcontrol.ResolvedPolicy{Rules: map[string]json.RawMessage{
		"runner.execution_profiles": json.RawMessage(
			`["strict","codex-delegated"]`,
		),
	}}
	for _, profile := range []workstation.ExecutionProfile{
		workstation.ExecutionProfileStrict,
		workstation.ExecutionProfileCodexDelegated,
	} {
		if err := validateTeamExecutionProfile(allowed, profile); err != nil {
			t.Fatalf("allowed profile %q rejected: %v", profile, err)
		}
	}

	for name, raw := range map[string]json.RawMessage{
		"wrong type": json.RawMessage(`"codex-delegated"`),
		"unknown":    json.RawMessage(`["future-profile"]`),
		"empty":      json.RawMessage(`[]`),
	} {
		t.Run(name, func(t *testing.T) {
			policy := teamcontrol.ResolvedPolicy{Rules: map[string]json.RawMessage{
				"runner.execution_profiles": raw,
			}}
			if err := validateTeamExecutionProfile(
				policy,
				workstation.ExecutionProfileCodexDelegated,
			); err == nil {
				t.Fatal("invalid policy unexpectedly allowed execution")
			}
		})
	}
}

func TestResolveRunnerLifecyclePolicyFailsClosed(t *testing.T) {
	policy := teamcontrol.ResolvedPolicy{Rules: map[string]json.RawMessage{
		"runner.target_version":    json.RawMessage(`"0.9.0"`),
		"runner.target_release_id": json.RawMessage(`"release-090"`),
		"runner.release_channel":   json.RawMessage(`"pilot"`),
		"runner.rollout_paused":    json.RawMessage(`true`),
	}}
	got, err := resolveRunnerLifecyclePolicy(policy)
	if err != nil {
		t.Fatal(err)
	}
	if got.TargetVersion != "0.9.0" ||
		got.TargetReleaseID != "release-090" ||
		got.ReleaseChannel != "pilot" ||
		!got.Paused {
		t.Fatalf("lifecycle policy = %#v", got)
	}
	for name, raw := range map[string]json.RawMessage{
		"empty target": json.RawMessage(`""`),
		"wrong type":   json.RawMessage(`false`),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := resolveRunnerLifecyclePolicy(
				teamcontrol.ResolvedPolicy{
					Rules: map[string]json.RawMessage{
						"runner.target_version": raw,
					},
				},
			)
			if err == nil {
				t.Fatal("invalid lifecycle policy passed")
			}
		})
	}
}
