package eval

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"lark-cue/internal/card"
	"lark-cue/internal/llm"
)

type Record struct {
	Type                  string          `json:"type"`
	CardID                string          `json:"card_id"`
	Command               string          `json:"command"`
	Scenario              string          `json:"scenario"`
	Reason                string          `json:"reason,omitempty"`
	ShouldRetrieve        *bool           `json:"should_retrieve,omitempty"`
	RetrievalStatus       string          `json:"retrieval_status"`
	RetrievalError        string          `json:"retrieval_error,omitempty"`
	Sources               []card.Citation `json:"sources"`
	Confidence            string          `json:"confidence,omitempty"`
	LatencyMS             int64           `json:"latency_ms"`
	QueryCount            int             `json:"query_count"`
	Feedback              string          `json:"feedback"`
	OpenClawAttempted     *bool           `json:"openclaw_attempted,omitempty"`
	OpenClawSucceeded     *bool           `json:"openclaw_succeeded,omitempty"`
	OpenClawSkippedReason string          `json:"openclaw_skipped_reason,omitempty"`
	OpenClawTimedOut      bool            `json:"openclaw_timed_out,omitempty"`
	OpenClawExitCode      *int            `json:"openclaw_exit_code,omitempty"`
	OpenClawError         string          `json:"openclaw_error,omitempty"`
	OpenClawLatencyMS     *int64          `json:"openclaw_latency_ms,omitempty"`
	CreatedAt             time.Time       `json:"created_at"`
}

func FromCard(k card.KnowledgeCard) Record {
	sources := k.Citations
	if sources == nil {
		sources = []card.Citation{}
	}
	attempted := k.OpenClaw.Attempted
	succeeded := k.OpenClaw.Succeeded
	latency := k.OpenClaw.LatencyMS
	record := Record{
		Type:            "cue",
		CardID:          k.ID,
		Command:         k.Command,
		Scenario:        k.Scenario,
		RetrievalStatus: string(k.RetrievalStatus),
		RetrievalError:  k.RetrievalError,
		Sources:         sources,
		Confidence:      string(k.Confidence),
		LatencyMS:       k.LatencyMS,
		QueryCount:      k.QueryCount,
		Feedback:        k.Feedback,
		CreatedAt:       k.CreatedAt,
	}
	record.OpenClawAttempted = &attempted
	record.OpenClawSkippedReason = k.OpenClaw.SkippedReason
	if attempted {
		record.OpenClawSucceeded = &succeeded
		record.OpenClawLatencyMS = &latency
		record.OpenClawTimedOut = k.OpenClaw.TimedOut
		record.OpenClawError = k.OpenClaw.Error
		exitCode := k.OpenClaw.ExitCode
		record.OpenClawExitCode = &exitCode
	}
	return record
}

func FromPlanner(command string, decision llm.PlanDecision, latencyMS int64) Record {
	shouldRetrieve := decision.ShouldRetrieve
	return Record{
		Type:           "planner",
		Command:        command,
		Scenario:       decision.Scenario,
		Reason:         decision.Reason,
		ShouldRetrieve: &shouldRetrieve,
		Sources:        []card.Citation{},
		LatencyMS:      latencyMS,
		QueryCount:     len(decision.Queries),
		CreatedAt:      time.Now(),
	}
}

func Append(path string, record Record) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	encoded, err := json.Marshal(record)
	if err != nil {
		return err
	}
	_, err = file.Write(append(encoded, '\n'))
	return err
}

func AppendFeedback(path, cardID, feedback string) error {
	return Append(path, Record{
		Type:      "feedback_update",
		CardID:    cardID,
		Sources:   []card.Citation{},
		Feedback:  feedback,
		CreatedAt: time.Now(),
	})
}
