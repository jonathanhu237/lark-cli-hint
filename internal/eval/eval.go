package eval

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"lark-cue/internal/card"
)

type Record struct {
	Type            string          `json:"type"`
	CardID          string          `json:"card_id"`
	Command         string          `json:"command"`
	Scenario        string          `json:"scenario"`
	RetrievalStatus string          `json:"retrieval_status"`
	RetrievalError  string          `json:"retrieval_error,omitempty"`
	Sources         []card.Citation `json:"sources"`
	LatencyMS       int64           `json:"latency_ms"`
	QueryCount      int             `json:"query_count"`
	Feedback        string          `json:"feedback"`
	CreatedAt       time.Time       `json:"created_at"`
}

func FromCard(k card.KnowledgeCard) Record {
	sources := k.Citations
	if sources == nil {
		sources = []card.Citation{}
	}
	return Record{
		Type:            "cue",
		CardID:          k.ID,
		Command:         k.Command,
		Scenario:        k.Scenario,
		RetrievalStatus: string(k.RetrievalStatus),
		RetrievalError:  k.RetrievalError,
		Sources:         sources,
		LatencyMS:       k.LatencyMS,
		QueryCount:      k.QueryCount,
		Feedback:        k.Feedback,
		CreatedAt:       k.CreatedAt,
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
