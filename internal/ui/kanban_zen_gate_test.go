package ui

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/asheshgoplani/agent-deck/internal/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- GateConfig Tests ---

func TestGateConfig_BacklogToDesign(t *testing.T) {
	cfg, ok := LookupGateConfig(session.KanbanBacklog, session.KanbanDesign)
	require.True(t, ok, "should find config for backlog->design")

	assert.Equal(t, "medium", cfg.ThinkingMode)
	assert.Equal(t, GateThreshold{Type: "both", MinScore: 7.0}, cfg.Threshold)
	assert.Len(t, cfg.Models, 2)

	// Verify models and stances
	modelMap := gateModelsToMap(cfg.Models)
	assert.Equal(t, "for", modelMap["gemini-3.1-pro-preview"])
	assert.Equal(t, "against", modelMap["gpt-5.2"])

	assert.Contains(t, cfg.Focus, "well-defined")
}

func TestGateConfig_DesignToPlan(t *testing.T) {
	cfg, ok := LookupGateConfig(session.KanbanDesign, session.KanbanPlan)
	require.True(t, ok)

	assert.Equal(t, "high", cfg.ThinkingMode)
	assert.Equal(t, GateThreshold{Type: "both", MinScore: 7.0}, cfg.Threshold)
	assert.Len(t, cfg.Models, 2)
}

func TestGateConfig_PlanToImplement(t *testing.T) {
	cfg, ok := LookupGateConfig(session.KanbanPlan, session.KanbanImplement)
	require.True(t, ok)

	assert.Equal(t, "high", cfg.ThinkingMode)
	assert.Equal(t, GateThreshold{Type: "majority", MinScore: 7.0}, cfg.Threshold)
	assert.Len(t, cfg.Models, 3) // gemini, gpt, o3

	modelMap := gateModelsToMap(cfg.Models)
	assert.Equal(t, "for", modelMap["gemini-3.1-pro-preview"])
	assert.Equal(t, "against", modelMap["gpt-5.2"])
	assert.Equal(t, "neutral", modelMap["o3"])
}

func TestGateConfig_ImplementToReview(t *testing.T) {
	cfg, ok := LookupGateConfig(session.KanbanImplement, session.KanbanReview)
	require.True(t, ok)

	assert.Equal(t, "high", cfg.ThinkingMode)
	assert.Equal(t, GateThreshold{Type: "both", MinScore: 7.0}, cfg.Threshold)
	assert.Len(t, cfg.Models, 2)
}

func TestGateConfig_ReviewToDone(t *testing.T) {
	cfg, ok := LookupGateConfig(session.KanbanReview, session.KanbanDone)
	require.True(t, ok)

	assert.Equal(t, "max", cfg.ThinkingMode)
	assert.Equal(t, GateThreshold{Type: "unanimous", MinScore: 8.0}, cfg.Threshold)
	assert.Len(t, cfg.Models, 3)

	modelMap := gateModelsToMap(cfg.Models)
	assert.Equal(t, "for", modelMap["gemini-3.1-pro-preview"])
	assert.Equal(t, "against", modelMap["gpt-5.2"])
	assert.Equal(t, "neutral", modelMap["o3"])

	assert.Contains(t, cfg.Focus, "merge")
}

func TestGateConfig_UnknownTransition(t *testing.T) {
	cfg, ok := LookupGateConfig(session.KanbanDone, session.KanbanBacklog)
	assert.False(t, ok)
	assert.Equal(t, GateConfig{}, cfg)
}

// --- GateThreshold Evaluation Tests ---

func TestPassThreshold_Unanimous(t *testing.T) {
	threshold := GateThreshold{Type: "unanimous", MinScore: 8.0}

	t.Run("all pass", func(t *testing.T) {
		verdicts := map[string]ModelVerdict{
			"gemini": {Score: 9.0, Pass: true},
			"gpt":    {Score: 8.5, Pass: true},
			"o3":     {Score: 8.0, Pass: true},
		}
		assert.True(t, EvaluateThreshold(verdicts, threshold))
	})

	t.Run("one fails", func(t *testing.T) {
		verdicts := map[string]ModelVerdict{
			"gemini": {Score: 9.0, Pass: true},
			"gpt":    {Score: 7.5, Pass: false},
			"o3":     {Score: 8.0, Pass: true},
		}
		assert.False(t, EvaluateThreshold(verdicts, threshold))
	})

	t.Run("empty verdicts", func(t *testing.T) {
		verdicts := map[string]ModelVerdict{}
		assert.False(t, EvaluateThreshold(verdicts, threshold))
	})
}

func TestPassThreshold_Majority(t *testing.T) {
	threshold := GateThreshold{Type: "majority", MinScore: 7.0}

	t.Run("2 of 3 pass", func(t *testing.T) {
		verdicts := map[string]ModelVerdict{
			"gemini": {Score: 8.0, Pass: true},
			"gpt":    {Score: 5.0, Pass: false},
			"o3":     {Score: 7.5, Pass: true},
		}
		assert.True(t, EvaluateThreshold(verdicts, threshold))
	})

	t.Run("1 of 3 pass", func(t *testing.T) {
		verdicts := map[string]ModelVerdict{
			"gemini": {Score: 8.0, Pass: true},
			"gpt":    {Score: 5.0, Pass: false},
			"o3":     {Score: 6.0, Pass: false},
		}
		assert.False(t, EvaluateThreshold(verdicts, threshold))
	})

	t.Run("all pass", func(t *testing.T) {
		verdicts := map[string]ModelVerdict{
			"gemini": {Score: 8.0, Pass: true},
			"gpt":    {Score: 7.5, Pass: true},
			"o3":     {Score: 9.0, Pass: true},
		}
		assert.True(t, EvaluateThreshold(verdicts, threshold))
	})

	t.Run("empty verdicts", func(t *testing.T) {
		verdicts := map[string]ModelVerdict{}
		assert.False(t, EvaluateThreshold(verdicts, threshold))
	})
}

func TestPassThreshold_Both(t *testing.T) {
	threshold := GateThreshold{Type: "both", MinScore: 7.0}

	t.Run("both pass", func(t *testing.T) {
		verdicts := map[string]ModelVerdict{
			"gemini": {Score: 8.0, Pass: true},
			"gpt":    {Score: 7.5, Pass: true},
		}
		assert.True(t, EvaluateThreshold(verdicts, threshold))
	})

	t.Run("one fails", func(t *testing.T) {
		verdicts := map[string]ModelVerdict{
			"gemini": {Score: 8.0, Pass: true},
			"gpt":    {Score: 6.5, Pass: false},
		}
		assert.False(t, EvaluateThreshold(verdicts, threshold))
	})
}

// --- ConsensusParams Building Tests ---

func TestBuildConsensusParams(t *testing.T) {
	cfg := GateConfig{
		Models: []GateModel{
			{Model: "gemini-3.1-pro-preview", Stance: "for", StancePrompt: "Evaluate as advocate"},
			{Model: "gpt-5.2", Stance: "against", StancePrompt: "Evaluate as critic"},
		},
		ThinkingMode: "high",
		Threshold:    GateThreshold{Type: "both", MinScore: 7.0},
		Focus:        "Work items well-defined with testable AC",
	}

	artifacts := []string{"/tmp/session-output.md", "/tmp/plan.md"}
	contID := "cont-abc"

	params := BuildConsensusParams(cfg, "sess-1", artifacts, contID)

	assert.Contains(t, params.Step, "Work items well-defined with testable AC")
	assert.Equal(t, 1, params.StepNumber)
	assert.Equal(t, 2, params.TotalSteps) // number of models
	assert.True(t, params.NextStepRequired)
	assert.Equal(t, "cont-abc", params.ContinuationID)
	assert.Equal(t, 0.2, params.Temperature)
	assert.Equal(t, "high", params.ThinkingMode)
	assert.Equal(t, artifacts, params.RelevantFiles)
	assert.Len(t, params.Models, 2)

	// Verify model details
	assert.Equal(t, "gemini-3.1-pro-preview", params.Models[0].Model)
	assert.Equal(t, "for", params.Models[0].Stance)
	assert.Equal(t, "Evaluate as advocate", params.Models[0].StancePrompt)
}

func TestBuildConsensusParams_EmptyArtifacts(t *testing.T) {
	cfg := GateConfig{
		Models: []GateModel{
			{Model: "gemini-3.1-pro-preview", Stance: "for"},
		},
		ThinkingMode: "medium",
		Threshold:    GateThreshold{Type: "both", MinScore: 7.0},
		Focus:        "Check items",
	}

	params := BuildConsensusParams(cfg, "sess-2", nil, "")

	assert.Empty(t, params.RelevantFiles)
	assert.Empty(t, params.ContinuationID)
	assert.Equal(t, 1, params.TotalSteps)
}

// --- ParseConsensusResponse Tests ---

func TestParseConsensusResponse(t *testing.T) {
	t.Run("all pass", func(t *testing.T) {
		resp := ConsensusResponse{
			ContinuationID: "cont-xyz",
			ModelResults: []ModelResult{
				{Model: "gemini-3.1-pro-preview", Score: 8.5, Verdict: "pass", Notes: "looks good"},
				{Model: "gpt-5.2", Score: 7.5, Verdict: "pass", Notes: "acceptable"},
			},
		}

		verdicts, contID := ParseConsensusResponse(resp)
		assert.Equal(t, "cont-xyz", contID)
		assert.Len(t, verdicts, 2)

		assert.Equal(t, 8.5, verdicts["gemini-3.1-pro-preview"].Score)
		assert.True(t, verdicts["gemini-3.1-pro-preview"].Pass)
		assert.Equal(t, "looks good", verdicts["gemini-3.1-pro-preview"].Notes)

		assert.Equal(t, 7.5, verdicts["gpt-5.2"].Score)
		assert.True(t, verdicts["gpt-5.2"].Pass)
	})

	t.Run("mixed results", func(t *testing.T) {
		resp := ConsensusResponse{
			ContinuationID: "cont-123",
			ModelResults: []ModelResult{
				{Model: "gemini", Score: 9.0, Verdict: "pass"},
				{Model: "gpt", Score: 4.0, Verdict: "fail"},
			},
		}

		verdicts, contID := ParseConsensusResponse(resp)
		assert.Equal(t, "cont-123", contID)
		assert.True(t, verdicts["gemini"].Pass)
		assert.False(t, verdicts["gpt"].Pass)
	})

	t.Run("empty response", func(t *testing.T) {
		resp := ConsensusResponse{}
		verdicts, contID := ParseConsensusResponse(resp)
		assert.Empty(t, contID)
		assert.Empty(t, verdicts)
	})
}

// --- ZenGateChecker Tests ---

func TestZenGateChecker_Pass(t *testing.T) {
	checker := NewZenGateChecker()

	result, err := checker.CheckGate(
		context.Background(),
		"sess-1",
		session.KanbanBacklog,
		session.KanbanDesign,
		"cont-start",
	)

	require.NoError(t, err)
	// The checker returns a pending result with consensus params
	assert.NotNil(t, result)
	assert.Equal(t, "cont-start", result.ContinuationID)

	// The result should contain consensus request params
	req := result.ConsensusRequest
	assert.NotNil(t, req)
	assert.Contains(t, req.Step, "well-defined")
	assert.Equal(t, "medium", req.ThinkingMode)
	assert.Len(t, req.Models, 2)
}

func TestZenGateChecker_ReviewToDone(t *testing.T) {
	checker := NewZenGateChecker()

	result, err := checker.CheckGate(
		context.Background(),
		"sess-review",
		session.KanbanReview,
		session.KanbanDone,
		"",
	)

	require.NoError(t, err)
	req := result.ConsensusRequest
	assert.NotNil(t, req)
	assert.Equal(t, "max", req.ThinkingMode)
	assert.Len(t, req.Models, 3) // gemini, gpt, o3
}

func TestZenGateChecker_UnknownTransition(t *testing.T) {
	checker := NewZenGateChecker()

	result, err := checker.CheckGate(
		context.Background(),
		"sess-1",
		session.KanbanDone,
		session.KanbanBacklog,
		"",
	)

	require.NoError(t, err)
	// Unknown transitions use default config
	assert.NotNil(t, result.ConsensusRequest)
	assert.Equal(t, "medium", result.ConsensusRequest.ThinkingMode)
}

func TestZenGateChecker_Timeout(t *testing.T) {
	checker := NewZenGateChecker()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := checker.CheckGate(
		ctx,
		"sess-1",
		session.KanbanBacklog,
		session.KanbanDesign,
		"",
	)

	assert.ErrorIs(t, err, context.Canceled)
}

func TestZenGateChecker_WithArtifacts(t *testing.T) {
	dir := t.TempDir()

	artifact1 := dir + "/output.md"
	err := os.WriteFile(artifact1, []byte("# Output\nAll tests pass"), 0o644)
	require.NoError(t, err)

	checker := NewZenGateChecker(WithArtifactPaths([]string{artifact1}))

	result, err := checker.CheckGate(
		context.Background(),
		"sess-1",
		session.KanbanImplement,
		session.KanbanReview,
		"",
	)

	require.NoError(t, err)
	assert.Contains(t, result.ConsensusRequest.RelevantFiles, artifact1)
}

// --- Large Output Handling Tests ---

func TestLargeOutputHandling(t *testing.T) {
	t.Run("small content returned as-is", func(t *testing.T) {
		content := "small output"
		path, cleanup, err := HandleLargeOutput(content, "test-gate-")
		require.NoError(t, err)
		defer cleanup()

		assert.Empty(t, path, "small content should not create a temp file")
	})

	t.Run("large content written to temp file", func(t *testing.T) {
		// 51K chars
		content := strings.Repeat("x", 51*1024)

		path, cleanup, err := HandleLargeOutput(content, "conductor-gate-")
		require.NoError(t, err)
		defer cleanup()

		assert.NotEmpty(t, path, "large content should create a temp file")
		assert.Contains(t, path, "conductor-gate-")

		data, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		assert.Equal(t, content, string(data))
	})
}

// --- BuildGateResult Tests ---

func TestBuildGateResult(t *testing.T) {
	t.Run("pass with unanimous threshold", func(t *testing.T) {
		threshold := GateThreshold{Type: "unanimous", MinScore: 8.0}
		verdicts := map[string]ModelVerdict{
			"gemini": {Score: 9.0, Pass: true, Notes: "good"},
			"gpt":    {Score: 8.5, Pass: true, Notes: "ok"},
			"o3":     {Score: 8.0, Pass: true, Notes: "fine"},
		}

		result := BuildGateResult(verdicts, threshold, "cont-new")
		assert.True(t, result.Pass)
		assert.Equal(t, "cont-new", result.ContinuationID)
		assert.Len(t, result.Models, 3)
		assert.Equal(t, "pass (9.0)", result.Models["gemini"])
	})

	t.Run("fail with majority threshold", func(t *testing.T) {
		threshold := GateThreshold{Type: "majority", MinScore: 7.0}
		verdicts := map[string]ModelVerdict{
			"gemini": {Score: 8.0, Pass: true},
			"gpt":    {Score: 4.0, Pass: false},
			"o3":     {Score: 5.0, Pass: false},
		}

		result := BuildGateResult(verdicts, threshold, "cont-456")
		assert.False(t, result.Pass)
		assert.Equal(t, "cont-456", result.ContinuationID)
	})
}

// --- ThinkdeepParams Building Tests ---

func TestBuildThinkdeepParams(t *testing.T) {
	verdicts := map[string]ModelVerdict{
		"gemini": {Score: 8.0, Pass: true, Notes: "good quality"},
		"gpt":    {Score: 5.0, Pass: false, Notes: "missing tests"},
	}

	params := BuildThinkdeepParams(
		"sess-1",
		session.KanbanPlan,
		session.KanbanImplement,
		verdicts,
		"cont-escalate",
		[]string{"/tmp/plan.md"},
	)

	assert.Contains(t, params.Prompt, "sess-1")
	assert.Contains(t, params.Prompt, "plan")
	assert.Contains(t, params.Prompt, "implement")
	assert.Equal(t, "cont-escalate", params.ContinuationID)
	assert.Equal(t, "high", params.ThinkingMode)
	assert.Equal(t, []string{"/tmp/plan.md"}, params.RelevantFiles)
}

// --- Integration: Full Gate Flow ---

func TestZenGateChecker_FullFlow(t *testing.T) {
	// This test simulates the full gate check flow:
	// 1. ZenGateChecker prepares consensus params
	// 2. "UI layer" processes the consensus response
	// 3. BuildGateResult produces the final GateResult

	checker := NewZenGateChecker()

	// Step 1: Get consensus params
	prepResult, err := checker.CheckGate(
		context.Background(),
		"sess-flow",
		session.KanbanDesign,
		session.KanbanPlan,
		"cont-flow",
	)
	require.NoError(t, err)
	assert.NotNil(t, prepResult.ConsensusRequest)

	// Step 2: Simulate consensus response from zen MCP
	response := ConsensusResponse{
		ContinuationID: "cont-flow-updated",
		ModelResults: []ModelResult{
			{Model: "gemini-3.1-pro-preview", Score: 8.0, Verdict: "pass", Notes: "design is solid"},
			{Model: "gpt-5.2", Score: 7.5, Verdict: "pass", Notes: "feasible"},
		},
	}

	// Step 3: Parse and evaluate
	verdicts, contID := ParseConsensusResponse(response)
	cfg, _ := LookupGateConfig(session.KanbanDesign, session.KanbanPlan)
	result := BuildGateResult(verdicts, cfg.Threshold, contID)

	assert.True(t, result.Pass)
	assert.Equal(t, "cont-flow-updated", result.ContinuationID)
	assert.Len(t, result.Models, 2)
}

func TestZenGateChecker_MixedConsensus(t *testing.T) {
	// When models disagree, we should be able to build thinkdeep params for escalation

	checker := NewZenGateChecker()

	// Step 1: Get consensus params
	prepResult, err := checker.CheckGate(
		context.Background(),
		"sess-mixed",
		session.KanbanPlan,
		session.KanbanImplement,
		"cont-mixed",
	)
	require.NoError(t, err)
	require.NotNil(t, prepResult.ConsensusRequest)

	// Step 2: Simulate mixed consensus response
	response := ConsensusResponse{
		ContinuationID: "cont-mixed-2",
		ModelResults: []ModelResult{
			{Model: "gemini-3.1-pro-preview", Score: 8.0, Verdict: "pass", Notes: "plan is good"},
			{Model: "gpt-5.2", Score: 5.0, Verdict: "fail", Notes: "missing edge cases"},
			{Model: "o3", Score: 7.0, Verdict: "pass", Notes: "borderline acceptable"},
		},
	}

	// Step 3: Parse results
	verdicts, contID := ParseConsensusResponse(response)
	cfg, _ := LookupGateConfig(session.KanbanPlan, session.KanbanImplement)

	// Majority threshold: 2/3 pass, this should pass
	result := BuildGateResult(verdicts, cfg.Threshold, contID)
	assert.True(t, result.Pass) // 2 of 3 >= 7.0

	// But if we want to escalate anyway for the disagreement, build thinkdeep params
	tdParams := BuildThinkdeepParams(
		"sess-mixed",
		session.KanbanPlan,
		session.KanbanImplement,
		verdicts,
		contID,
		nil,
	)
	assert.Contains(t, tdParams.Prompt, "sess-mixed")
	assert.Equal(t, "cont-mixed-2", tdParams.ContinuationID)
}

// --- GateConfig Immutability Test ---

func TestGateConfig_Immutability(t *testing.T) {
	cfg1, ok := LookupGateConfig(session.KanbanBacklog, session.KanbanDesign)
	require.True(t, ok)

	// Modify returned config
	cfg1.ThinkingMode = "max"
	cfg1.Models = append(cfg1.Models, GateModel{Model: "rogue"})

	// Fetch again, should be unchanged
	cfg2, _ := LookupGateConfig(session.KanbanBacklog, session.KanbanDesign)
	assert.Equal(t, "medium", cfg2.ThinkingMode)
	assert.Len(t, cfg2.Models, 2)
}

// --- ConsensusParams Time Tests ---

func TestConsensusParams_HasTimestamp(t *testing.T) {
	cfg := GateConfig{
		Models: []GateModel{
			{Model: "gemini-3.1-pro-preview", Stance: "for"},
		},
		ThinkingMode: "medium",
		Focus:        "test focus",
	}

	before := time.Now()
	params := BuildConsensusParams(cfg, "sess-ts", nil, "")
	after := time.Now()

	assert.False(t, params.CreatedAt.Before(before))
	assert.False(t, params.CreatedAt.After(after))
}

// --- Helper ---

func gateModelsToMap(models []GateModel) map[string]string {
	m := make(map[string]string, len(models))
	for _, gm := range models {
		m[gm.Model] = gm.Stance
	}
	return m
}
