package ui

// kanban_zen_gate.go — Zen MCP consensus gate protocol for validating kanban
// column transitions.
//
// The ZenGateChecker implements the GateChecker interface by preparing
// consensus parameters for the zen MCP tool. The actual MCP calls happen in
// the UI layer (home.go message handlers). This module is responsible for:
//
//  1. Gate configuration lookup per transition
//  2. Consensus parameter building (step text, models, stances, files)
//  3. Response parsing (extract per-model verdicts, scores, continuation_id)
//  4. Threshold evaluation (unanimous, majority, both)
//  5. Thinkdeep escalation parameter building for mixed consensus
//  6. Large output handling (>50K chars saved to temp file)

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/asheshgoplani/agent-deck/internal/logging"
	"github.com/asheshgoplani/agent-deck/internal/session"
)

var zenGateLog = logging.ForComponent("zen-gate")

// largeOutputThreshold is the character count above which output is written
// to a temp file instead of being passed inline.
const largeOutputThreshold = 50 * 1024

// --- Gate Configuration Types ---

// GateConfig defines the consensus gate parameters for a column transition.
type GateConfig struct {
	Models       []GateModel
	ThinkingMode string // "medium", "high", "max"
	Threshold    GateThreshold
	Focus        string // evaluation question focus
}

// GateModel describes a single model participating in a consensus gate.
type GateModel struct {
	Model       string // "gemini-3.1-pro-preview", "gpt-5.2", "o3"
	Stance      string // "for", "against", "neutral"
	StancePrompt string // role-specific evaluation prompt
}

// GateThreshold defines the pass criteria for a gate.
type GateThreshold struct {
	Type     string  // "unanimous", "majority", "both"
	MinScore float64 // 7.0 or 8.0
}

// ModelVerdict holds a single model's evaluation outcome.
type ModelVerdict struct {
	Score float64
	Pass  bool
	Notes string
}

// --- Consensus Protocol Types ---

// ConsensusParams holds the parameters needed to invoke mcp__zen__consensus.
type ConsensusParams struct {
	Step             string        // frozen proposal all models evaluate
	StepNumber       int           // always 1 for initial request
	TotalSteps       int           // number of models
	NextStepRequired bool          // true for step 1
	ContinuationID   string        // reused across gates for context revival
	Temperature      float64       // 0.2 for gate decisions
	ThinkingMode     string        // from GateConfig
	RelevantFiles    []string      // artifact file paths
	Models           []GateModel   // models with stances
	CreatedAt        time.Time     // when params were built
}

// ConsensusResponse holds the parsed result from a zen consensus call.
type ConsensusResponse struct {
	ContinuationID string
	ModelResults   []ModelResult
}

// ModelResult holds a single model's response from the consensus call.
type ModelResult struct {
	Model   string
	Score   float64
	Verdict string // "pass" or "fail"
	Notes   string
}

// ThinkdeepParams holds the parameters for a thinkdeep escalation call.
type ThinkdeepParams struct {
	Prompt         string
	ContinuationID string
	ThinkingMode   string
	RelevantFiles  []string
}

// --- Gate Configuration Registry ---

// gateConfigs maps column transitions to their gate configurations.
// This is the source of truth for gate behavior.
var gateConfigs = map[[2]session.KanbanColumn]GateConfig{
	{session.KanbanBacklog, session.KanbanDesign}: {
		Models: []GateModel{
			{Model: "gemini-3.1-pro-preview", Stance: "for", StancePrompt: "Evaluate as advocate: Are these work items well-defined with clear, testable acceptance criteria?"},
			{Model: "gpt-5.2", Stance: "against", StancePrompt: "Evaluate as critic: Are there ambiguities, missing edge cases, or untestable requirements in these work items?"},
		},
		ThinkingMode: "medium",
		Threshold:    GateThreshold{Type: "both", MinScore: 7.0},
		Focus:        "Work items well-defined with testable AC",
	},
	{session.KanbanDesign, session.KanbanPlan}: {
		Models: []GateModel{
			{Model: "gemini-3.1-pro-preview", Stance: "for", StancePrompt: "Evaluate as advocate: Is this design feasible, complete, and ready for planning?"},
			{Model: "gpt-5.2", Stance: "against", StancePrompt: "Evaluate as critic: Are there design gaps, scalability concerns, or missing considerations?"},
		},
		ThinkingMode: "high",
		Threshold:    GateThreshold{Type: "both", MinScore: 7.0},
		Focus:        "Design feasible, complete, ready for planning",
	},
	{session.KanbanPlan, session.KanbanImplement}: {
		Models: []GateModel{
			{Model: "gemini-3.1-pro-preview", Stance: "for", StancePrompt: "Evaluate as advocate: Is this plan well-decomposed with clear tasks and dependencies?"},
			{Model: "gpt-5.2", Stance: "against", StancePrompt: "Evaluate as critic: Are there missing tasks, incorrect dependencies, or underestimated complexity?"},
			{Model: "o3", Stance: "neutral", StancePrompt: "Evaluate as neutral arbiter: Objectively assess plan decomposition quality and readiness for implementation."},
		},
		ThinkingMode: "high",
		Threshold:    GateThreshold{Type: "majority", MinScore: 7.0},
		Focus:        "Plan decomposed correctly with clear tasks and dependencies",
	},
	{session.KanbanImplement, session.KanbanReview}: {
		Models: []GateModel{
			{Model: "gemini-3.1-pro-preview", Stance: "for", StancePrompt: "Evaluate as advocate: Is this implementation complete and consistent with the plan?"},
			{Model: "gpt-5.2", Stance: "against", StancePrompt: "Evaluate as critic: Are there missing features, incomplete tests, or deviations from the plan?"},
		},
		ThinkingMode: "high",
		Threshold:    GateThreshold{Type: "both", MinScore: 7.0},
		Focus:        "Implementation complete against plan",
	},
	{session.KanbanReview, session.KanbanDone}: {
		Models: []GateModel{
			{Model: "gemini-3.1-pro-preview", Stance: "for", StancePrompt: "Evaluate as advocate: Is this work safe to merge? Are tests passing, code quality high, and no regressions?"},
			{Model: "gpt-5.2", Stance: "against", StancePrompt: "Evaluate as critic: Are there security issues, performance regressions, missing tests, or code quality concerns?"},
			{Model: "o3", Stance: "neutral", StancePrompt: "Evaluate as neutral arbiter: Objectively assess merge readiness including test coverage, code quality, and risk."},
		},
		ThinkingMode: "max",
		Threshold:    GateThreshold{Type: "unanimous", MinScore: 8.0},
		Focus:        "Safe to merge: tests passing, quality high, no regressions",
	},
}

// defaultGateConfig is used when no specific gate config is found for a transition.
var defaultGateConfig = GateConfig{
	Models: []GateModel{
		{Model: "gemini-3.1-pro-preview", Stance: "for", StancePrompt: "Evaluate as advocate"},
		{Model: "gpt-5.2", Stance: "against", StancePrompt: "Evaluate as critic"},
	},
	ThinkingMode: "medium",
	Threshold:    GateThreshold{Type: "both", MinScore: 7.0},
	Focus:        "Transition quality check",
}

// LookupGateConfig returns the gate configuration for a column transition.
// Returns a deep copy to prevent mutation of the registry.
// The second return value is false if no specific config exists for the transition.
func LookupGateConfig(from, to session.KanbanColumn) (GateConfig, bool) {
	key := [2]session.KanbanColumn{from, to}
	cfg, ok := gateConfigs[key]
	if !ok {
		return GateConfig{}, false
	}
	return copyGateConfig(cfg), true
}

// --- ZenGateChecker ---

// ZenGateChecker implements GateChecker by preparing consensus parameters
// for the zen MCP consensus tool. It does not make actual MCP calls; instead
// it returns a GateResult with ConsensusRequest populated for the UI layer.
type ZenGateChecker struct {
	artifactPaths []string
}

// ZenGateOption configures a ZenGateChecker.
type ZenGateOption func(*ZenGateChecker)

// WithArtifactPaths sets the artifact file paths to include in gate checks.
func WithArtifactPaths(paths []string) ZenGateOption {
	return func(z *ZenGateChecker) {
		copied := make([]string, len(paths))
		copy(copied, paths)
		z.artifactPaths = copied
	}
}

// NewZenGateChecker creates a new ZenGateChecker with the given options.
func NewZenGateChecker(opts ...ZenGateOption) *ZenGateChecker {
	z := &ZenGateChecker{}
	for _, opt := range opts {
		opt(z)
	}
	return z
}

// CheckGate prepares consensus parameters for the given column transition.
// Returns a GateResult with ConsensusRequest populated. The actual MCP call
// is made by the UI layer.
//
// Returns context.Canceled or context.DeadlineExceeded if the context is done.
func (z *ZenGateChecker) CheckGate(
	ctx context.Context,
	sessionID string,
	from, to session.KanbanColumn,
	continuationID string,
) (GateResult, error) {
	if err := ctx.Err(); err != nil {
		return GateResult{}, err
	}

	cfg, ok := LookupGateConfig(from, to)
	if !ok {
		zenGateLog.Info("gate_config_fallback",
			slog.String("session", sessionID),
			slog.String("from", from.String()),
			slog.String("to", to.String()),
		)
		cfg = copyGateConfig(defaultGateConfig)
	}

	params := BuildConsensusParams(cfg, sessionID, z.artifactPaths, continuationID)

	zenGateLog.Info("gate_check_prepared",
		slog.String("session", sessionID),
		slog.String("from", from.String()),
		slog.String("to", to.String()),
		slog.String("thinking", cfg.ThinkingMode),
		slog.Int("models", len(cfg.Models)),
	)

	return GateResult{
		Pass:             false, // pending; actual result comes after MCP call
		Models:           make(map[string]string),
		ContinuationID:   continuationID,
		ConsensusRequest: &params,
	}, nil
}

// --- Parameter Building ---

// BuildConsensusParams constructs the parameters for a zen consensus call.
func BuildConsensusParams(
	cfg GateConfig,
	sessionID string,
	artifacts []string,
	continuationID string,
) ConsensusParams {
	step := buildStepText(cfg.Focus, sessionID)

	models := make([]GateModel, len(cfg.Models))
	copy(models, cfg.Models)

	files := make([]string, len(artifacts))
	copy(files, artifacts)

	return ConsensusParams{
		Step:             step,
		StepNumber:       1,
		TotalSteps:       len(cfg.Models),
		NextStepRequired: true,
		ContinuationID:   continuationID,
		Temperature:      0.2,
		ThinkingMode:     cfg.ThinkingMode,
		RelevantFiles:    files,
		Models:           models,
		CreatedAt:        time.Now(),
	}
}

// buildStepText constructs the frozen proposal text that all models evaluate.
// Session ID is sanitized to prevent prompt injection via crafted session names.
func buildStepText(focus, sessionID string) string {
	safeID := sanitizePromptField(sessionID)
	return fmt.Sprintf(
		"Gate evaluation for session %s: %s\n\n"+
			"Score this transition 1-10 based on the criteria above. "+
			"A score >= the threshold means PASS. Provide your score, verdict (pass/fail), "+
			"and brief notes explaining your reasoning.",
		safeID, focus,
	)
}

// sanitizePromptField strips newlines and control characters to prevent prompt injection.
func sanitizePromptField(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	// Limit length to prevent oversized session IDs from dominating the prompt
	const maxFieldLen = 200
	if len(s) > maxFieldLen {
		s = s[:maxFieldLen]
	}
	return s
}

// --- Response Parsing ---

// ParseConsensusResponse extracts per-model verdicts and continuation ID
// from a consensus response.
func ParseConsensusResponse(resp ConsensusResponse) (map[string]ModelVerdict, string) {
	verdicts := make(map[string]ModelVerdict, len(resp.ModelResults))
	for _, mr := range resp.ModelResults {
		verdicts[mr.Model] = ModelVerdict{
			Score: mr.Score,
			Pass:  mr.Verdict == "pass",
			Notes: mr.Notes,
		}
	}
	return verdicts, resp.ContinuationID
}

// --- Threshold Evaluation ---

// EvaluateThreshold determines whether the model verdicts meet the gate threshold.
func EvaluateThreshold(verdicts map[string]ModelVerdict, threshold GateThreshold) bool {
	if len(verdicts) == 0 {
		return false
	}

	switch threshold.Type {
	case "unanimous", "both":
		return evaluateUnanimous(verdicts, threshold.MinScore)
	case "majority":
		return evaluateMajority(verdicts, threshold.MinScore)
	default:
		zenGateLog.Warn("unknown_threshold_type",
			slog.String("type", threshold.Type),
		)
		return evaluateUnanimous(verdicts, threshold.MinScore)
	}
}

// evaluateUnanimous returns true only if ALL model scores meet the minimum.
func evaluateUnanimous(verdicts map[string]ModelVerdict, minScore float64) bool {
	for _, v := range verdicts {
		if v.Score < minScore {
			return false
		}
	}
	return true
}

// evaluateMajority returns true if more than 50% of model scores meet the minimum.
func evaluateMajority(verdicts map[string]ModelVerdict, minScore float64) bool {
	passing := 0
	for _, v := range verdicts {
		if v.Score >= minScore {
			passing++
		}
	}
	return passing*2 > len(verdicts)
}

// --- GateResult Building ---

// BuildGateResult constructs a GateResult from parsed verdicts and threshold.
func BuildGateResult(
	verdicts map[string]ModelVerdict,
	threshold GateThreshold,
	continuationID string,
) GateResult {
	pass := EvaluateThreshold(verdicts, threshold)

	models := make(map[string]string, len(verdicts))
	for name, v := range verdicts {
		verdict := "fail"
		if v.Pass {
			verdict = "pass"
		}
		models[name] = fmt.Sprintf("%s (%.1f)", verdict, v.Score)
	}

	return GateResult{
		Pass:           pass,
		Models:         models,
		ContinuationID: continuationID,
	}
}

// --- Thinkdeep Escalation ---

// BuildThinkdeepParams constructs parameters for a thinkdeep escalation call
// when models disagree during consensus.
func BuildThinkdeepParams(
	sessionID string,
	from, to session.KanbanColumn,
	verdicts map[string]ModelVerdict,
	continuationID string,
	artifacts []string,
) ThinkdeepParams {
	var summary strings.Builder
	safeID := sanitizePromptField(sessionID)
	summary.WriteString(fmt.Sprintf(
		"Mixed consensus for session %s (%s -> %s).\n\nModel verdicts:\n",
		safeID, from.String(), to.String(),
	))
	for model, v := range verdicts {
		verdict := "FAIL"
		if v.Pass {
			verdict = "PASS"
		}
		summary.WriteString(fmt.Sprintf("- %s: %s (%.1f) — %s\n", sanitizePromptField(model), verdict, v.Score, sanitizePromptField(v.Notes)))
	}
	summary.WriteString("\nRecommend: advance (with warning), retry (with feedback), or pause.\n")

	files := make([]string, len(artifacts))
	copy(files, artifacts)

	return ThinkdeepParams{
		Prompt:         summary.String(),
		ContinuationID: continuationID,
		ThinkingMode:   "high",
		RelevantFiles:  files,
	}
}

// --- Large Output Handling ---

// HandleLargeOutput checks if content exceeds the large output threshold.
// If so, it writes the content to a temp file and returns the path.
// The cleanup function should be deferred to remove the temp file.
// If content is small, path is empty and cleanup is a no-op.
func HandleLargeOutput(content, prefix string) (path string, cleanup func(), err error) {
	noop := func() {}

	if len(content) <= largeOutputThreshold {
		return "", noop, nil
	}

	f, err := os.CreateTemp("", prefix)
	if err != nil {
		return "", noop, fmt.Errorf("create temp file for large output: %w", err)
	}

	if _, err := f.WriteString(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", noop, fmt.Errorf("write large output to temp file: %w", err)
	}

	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return "", noop, fmt.Errorf("close temp file: %w", err)
	}

	cleanupFn := func() {
		os.Remove(f.Name())
	}
	return f.Name(), cleanupFn, nil
}

// --- Internal Helpers ---

// copyGateConfig returns a deep copy of a GateConfig to prevent mutation.
func copyGateConfig(cfg GateConfig) GateConfig {
	models := make([]GateModel, len(cfg.Models))
	copy(models, cfg.Models)
	return GateConfig{
		Models:       models,
		ThinkingMode: cfg.ThinkingMode,
		Threshold:    cfg.Threshold,
		Focus:        cfg.Focus,
	}
}