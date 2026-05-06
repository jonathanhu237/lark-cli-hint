package retrieval

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

type Retriever interface {
	Retrieve(ctx context.Context, queries []string) ([]Source, Status, error)
}

type Status string

const (
	StatusOK      Status = "ok"
	StatusPartial Status = "partial"
	StatusFailed  Status = "failed"
)

type Source struct {
	Type      string
	Title     string
	URL       string
	ID        string
	Content   string
	Summary   string
	ChatName  string
	Sender    string
	Timestamp string
	Fetched   bool
}

type jsonRunner interface {
	RunJSON(ctx context.Context, args ...string) (map[string]any, error)
}

func (s Source) CitationLabel() string {
	switch s.Type {
	case "im":
		parts := []string{"群聊"}
		if s.ChatName != "" {
			parts = append(parts, s.ChatName)
		}
		if s.Sender != "" {
			parts = append(parts, s.Sender)
		}
		if s.Timestamp != "" {
			parts = append(parts, s.Timestamp)
		}
		return strings.Join(parts, " / ")
	default:
		if s.Title != "" {
			return s.Title
		}
		if s.URL != "" {
			return s.URL
		}
		return s.ID
	}
}

type LarkRetriever struct {
	client jsonRunner
}

func NewLarkRetriever(client jsonRunner) *LarkRetriever {
	return &LarkRetriever{client: client}
}

func (r *LarkRetriever) Retrieve(ctx context.Context, queries []string) ([]Source, Status, error) {
	var all []Source
	var errs []string
	seen := map[string]bool{}

	for _, query := range queries {
		docs, err := r.searchDocs(ctx, query)
		if err != nil {
			errs = append(errs, err.Error())
		}
		for _, doc := range docs {
			key := "doc:" + firstNonEmpty(doc.ID, doc.URL, doc.Title)
			if seen[key] {
				continue
			}
			seen[key] = true
			fetched, err := r.fetchDoc(ctx, doc)
			if err != nil {
				errs = append(errs, err.Error())
				continue
			}
			all = append(all, fetched)
		}

		messages, err := r.searchMessages(ctx, query)
		if err != nil {
			errs = append(errs, err.Error())
		}
		for _, msg := range messages {
			key := "im:" + firstNonEmpty(msg.ID, msg.Content)
			if seen[key] {
				continue
			}
			seen[key] = true
			all = append(all, msg)
		}
	}

	if len(all) == 0 && len(errs) > 0 {
		return nil, StatusFailed, errors.New(strings.Join(errs, "; "))
	}
	if len(all) > 0 && len(errs) > 0 {
		return all, StatusPartial, errors.New(strings.Join(errs, "; "))
	}
	return all, StatusOK, nil
}

func (r *LarkRetriever) searchDocs(ctx context.Context, query string) ([]Source, error) {
	out, err := r.client.RunJSON(ctx, "docs", "+search", "--query", query, "--page-size", "5", "--format", "json")
	if err != nil {
		return nil, err
	}
	results := arrayAt(out, "data", "results")
	var sources []Source
	for _, item := range results {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		meta, _ := obj["result_meta"].(map[string]any)
		source := Source{
			Type:    "doc",
			Title:   cleanHighlight(stringAt(obj, "title_highlighted")),
			Summary: cleanHighlight(stringAt(obj, "summary_highlighted")),
			URL:     stringAt(meta, "url"),
			ID:      stringAt(meta, "token"),
		}
		if source.Title == "" {
			source.Title = firstNonEmpty(source.ID, source.URL)
		}
		sources = append(sources, source)
	}
	return sources, nil
}

func (r *LarkRetriever) fetchDoc(ctx context.Context, source Source) (Source, error) {
	docRef := firstNonEmpty(source.URL, source.ID)
	if docRef == "" {
		return source, fmt.Errorf("doc source missing url/token")
	}
	out, err := r.client.RunJSON(ctx, "docs", "+fetch", "--doc", docRef, "--format", "json")
	if err != nil {
		return source, err
	}
	data, _ := out["data"].(map[string]any)
	source.Title = firstNonEmpty(stringAt(data, "title"), source.Title)
	source.Content = stringAt(data, "markdown")
	source.ID = firstNonEmpty(stringAt(data, "doc_id"), source.ID)
	source.Fetched = source.Content != ""
	return source, nil
}

func (r *LarkRetriever) searchMessages(ctx context.Context, query string) ([]Source, error) {
	out, err := r.client.RunJSON(ctx, "im", "+messages-search", "--query", query, "--page-size", "5", "--format", "json")
	if err != nil {
		return nil, err
	}
	messages := arrayAt(out, "data", "messages")
	var sources []Source
	for _, item := range messages {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		sender, _ := obj["sender"].(map[string]any)
		source := Source{
			Type:      "im",
			ID:        stringAt(obj, "message_id"),
			Content:   stringAt(obj, "content"),
			ChatName:  stringAt(obj, "chat_name"),
			Sender:    stringAt(sender, "name"),
			Timestamp: stringAt(obj, "create_time"),
			Fetched:   stringAt(obj, "content") != "",
		}
		sources = append(sources, source)
	}
	return sources, nil
}

func arrayAt(obj map[string]any, path ...string) []any {
	var cur any = obj
	for _, part := range path {
		next, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = next[part]
	}
	values, _ := cur.([]any)
	return values
}

func stringAt(obj map[string]any, key string) string {
	if obj == nil {
		return ""
	}
	value, ok := obj[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case json.Number:
		return typed.String()
	default:
		return fmt.Sprint(typed)
	}
}

var tagRE = regexp.MustCompile(`<[^>]+>`)

func cleanHighlight(value string) string {
	return strings.TrimSpace(tagRE.ReplaceAllString(value, ""))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
