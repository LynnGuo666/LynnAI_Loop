package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"loop/internal/models"
)

type requestMeta struct {
	Model    string
	Stream   bool
	Endpoint string
}

type streamMetrics struct {
	Usage        parsedUsage
	FirstTokenMs int64
}

type probeRequest struct {
	Path string
	Body []byte
}

type protocolAdapter interface {
	Name() string
	MatchProxyPath(path string) bool
	RequestMeta(r *http.Request, body []byte) requestMeta
	UpstreamPath(meta requestMeta) (string, error)
	ApplyHeaders(req *http.Request, inbound http.Header, key string)
	ParseNonStreamUsage(body []byte) parsedUsage
	ObserveStreamData(data []byte, metrics *streamMetrics, elapsedMs int64)
	ProbeModelsPath() string
	ParseModels(body []byte) ([]string, error)
	BuildProbeRequest(modelID string) probeRequest
}

func adapterForChannel(ch *models.Channel) protocolAdapter {
	switch strings.TrimSpace(ch.Protocol) {
	case models.ProtocolOpenAIChatCompletions:
		return openAIChatAdapter{}
	case models.ProtocolOpenAIResponses:
		return openAIResponsesAdapter{}
	case models.ProtocolGeminiGenerateContent:
		return geminiAdapter{}
	default:
		return anthropicAdapter{}
	}
}

func normalizeIncomingPath(path string) string {
	if strings.HasPrefix(path, "/channel/") {
		parts := strings.SplitN(strings.TrimPrefix(path, "/channel/"), "/", 2)
		if len(parts) == 2 {
			return "/" + parts[1]
		}
	}
	return path
}

func unmarshalObject(body []byte) map[string]json.RawMessage {
	var obj map[string]json.RawMessage
	if json.Unmarshal(body, &obj) != nil {
		return map[string]json.RawMessage{}
	}
	return obj
}

func rawString(obj map[string]json.RawMessage, key string) string {
	var value string
	if raw, ok := obj[key]; ok && json.Unmarshal(raw, &value) == nil {
		return strings.TrimSpace(value)
	}
	return ""
}

func rawBool(obj map[string]json.RawMessage, key string) bool {
	var value bool
	if raw, ok := obj[key]; ok && json.Unmarshal(raw, &value) == nil {
		return value
	}
	return false
}

type anthropicAdapter struct{}

func (anthropicAdapter) Name() string { return models.ProtocolAnthropicMessages }

func (anthropicAdapter) MatchProxyPath(path string) bool {
	return path == "/v1/messages"
}

func (anthropicAdapter) RequestMeta(_ *http.Request, body []byte) requestMeta {
	obj := unmarshalObject(body)
	return requestMeta{
		Model:    rawString(obj, "model"),
		Stream:   rawBool(obj, "stream"),
		Endpoint: "/v1/messages",
	}
}

func (anthropicAdapter) UpstreamPath(_ requestMeta) (string, error) {
	return "/v1/messages", nil
}

func (anthropicAdapter) ApplyHeaders(req *http.Request, inbound http.Header, key string) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("x-api-key", key)
	if v := inbound.Get("anthropic-version"); v != "" {
		req.Header.Set("anthropic-version", v)
	} else {
		req.Header.Set("anthropic-version", "2023-06-01")
	}
	if beta := inbound.Get("anthropic-beta"); beta != "" {
		req.Header.Set("anthropic-beta", beta)
	}
}

func (anthropicAdapter) ParseNonStreamUsage(body []byte) parsedUsage {
	return extractUsageFromBody(body, anthropicUsageParser)
}

func (anthropicAdapter) ObserveStreamData(data []byte, metrics *streamMetrics, elapsedMs int64) {
	var event map[string]json.RawMessage
	if json.Unmarshal(data, &event) != nil {
		return
	}
	if metrics.FirstTokenMs == 0 && anthropicEventHasTextDelta(event) {
		metrics.FirstTokenMs = elapsedMs
	}
	if t, ok := event["type"]; ok {
		var eventType string
		json.Unmarshal(t, &eventType)
		switch eventType {
		case "message_start":
			extractTokensFromMessageStart(event, &metrics.Usage.InputTokens, &metrics.Usage.CacheCreationTokens, &metrics.Usage.CacheReadTokens)
		case "message_delta":
			extractTokensFromMessageDelta(event, &metrics.Usage)
		}
	}
}

func (anthropicAdapter) ProbeModelsPath() string { return "/v1/models" }

func (anthropicAdapter) ParseModels(body []byte) ([]string, error) {
	return parseOpenAIStyleModels(body)
}

func (anthropicAdapter) BuildProbeRequest(modelID string) probeRequest {
	body, _ := json.Marshal(map[string]interface{}{
		"model":      modelID,
		"max_tokens": 1,
		"messages":   []map[string]string{{"role": "user", "content": "hi"}},
	})
	return probeRequest{Path: "/v1/messages", Body: body}
}

type openAIBaseAdapter struct{}

func (openAIBaseAdapter) ApplyHeaders(req *http.Request, _ http.Header, key string) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("Authorization", "Bearer "+key)
}

func (openAIBaseAdapter) ProbeModelsPath() string { return "/v1/models" }

func (openAIBaseAdapter) ParseModels(body []byte) ([]string, error) {
	return parseOpenAIStyleModels(body)
}

func (openAIBaseAdapter) ParseNonStreamUsage(body []byte) parsedUsage {
	return extractUsageFromBody(body, openAIUsageParser)
}

func (openAIBaseAdapter) observeOpenAIStreamData(data []byte, metrics *streamMetrics, elapsedMs int64) {
	var event map[string]json.RawMessage
	if json.Unmarshal(data, &event) != nil {
		return
	}
	if metrics.FirstTokenMs == 0 && openAIEventHasTextDelta(event) {
		metrics.FirstTokenMs = elapsedMs
	}
	if raw, ok := event["usage"]; ok {
		mergeUsage(&metrics.Usage, openAIUsageParser(raw))
	}
	if raw, ok := event["response"]; ok {
		var response struct {
			Usage json.RawMessage `json:"usage"`
		}
		if json.Unmarshal(raw, &response) == nil && len(response.Usage) > 0 {
			mergeUsage(&metrics.Usage, openAIUsageParser(response.Usage))
		}
	}
}

type openAIChatAdapter struct {
	openAIBaseAdapter
}

func (openAIChatAdapter) Name() string { return models.ProtocolOpenAIChatCompletions }

func (openAIChatAdapter) MatchProxyPath(path string) bool {
	return path == "/v1/chat/completions"
}

func (openAIChatAdapter) RequestMeta(_ *http.Request, body []byte) requestMeta {
	obj := unmarshalObject(body)
	return requestMeta{
		Model:    rawString(obj, "model"),
		Stream:   rawBool(obj, "stream"),
		Endpoint: "/v1/chat/completions",
	}
}

func (openAIChatAdapter) UpstreamPath(_ requestMeta) (string, error) {
	return "/v1/chat/completions", nil
}

func (a openAIChatAdapter) ObserveStreamData(data []byte, metrics *streamMetrics, elapsedMs int64) {
	a.observeOpenAIStreamData(data, metrics, elapsedMs)
}

func (openAIChatAdapter) BuildProbeRequest(modelID string) probeRequest {
	body, _ := json.Marshal(map[string]interface{}{
		"model":      modelID,
		"max_tokens": 1,
		"messages":   []map[string]string{{"role": "user", "content": "hi"}},
	})
	return probeRequest{Path: "/v1/chat/completions", Body: body}
}

type openAIResponsesAdapter struct {
	openAIBaseAdapter
}

func (openAIResponsesAdapter) Name() string { return models.ProtocolOpenAIResponses }

func (openAIResponsesAdapter) MatchProxyPath(path string) bool {
	return path == "/v1/responses"
}

func (openAIResponsesAdapter) RequestMeta(_ *http.Request, body []byte) requestMeta {
	obj := unmarshalObject(body)
	return requestMeta{
		Model:    rawString(obj, "model"),
		Stream:   rawBool(obj, "stream"),
		Endpoint: "/v1/responses",
	}
}

func (openAIResponsesAdapter) UpstreamPath(_ requestMeta) (string, error) {
	return "/v1/responses", nil
}

func (a openAIResponsesAdapter) ObserveStreamData(data []byte, metrics *streamMetrics, elapsedMs int64) {
	a.observeOpenAIStreamData(data, metrics, elapsedMs)
}

func (openAIResponsesAdapter) BuildProbeRequest(modelID string) probeRequest {
	body, _ := json.Marshal(map[string]interface{}{
		"model":             modelID,
		"max_output_tokens": 1,
		"input":             "hi",
	})
	return probeRequest{Path: "/v1/responses", Body: body}
}

type geminiAdapter struct{}

func (geminiAdapter) Name() string { return models.ProtocolGeminiGenerateContent }

func (geminiAdapter) MatchProxyPath(path string) bool {
	return strings.HasPrefix(path, "/v1beta/models/") &&
		(strings.HasSuffix(path, ":generateContent") || strings.HasSuffix(path, ":streamGenerateContent"))
}

func (geminiAdapter) RequestMeta(r *http.Request, body []byte) requestMeta {
	path := normalizeIncomingPath(r.URL.Path)
	model := geminiModelFromPath(path)
	stream := strings.HasSuffix(path, ":streamGenerateContent")
	if model == "" {
		model = rawString(unmarshalObject(body), "model")
	}
	return requestMeta{
		Model:    model,
		Stream:   stream,
		Endpoint: path,
	}
}

func (geminiAdapter) UpstreamPath(meta requestMeta) (string, error) {
	if meta.Endpoint == "" {
		return "", fmt.Errorf("missing gemini endpoint")
	}
	return meta.Endpoint, nil
}

func (geminiAdapter) ApplyHeaders(req *http.Request, _ http.Header, key string) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("x-goog-api-key", key)
	if strings.HasSuffix(req.URL.Path, ":streamGenerateContent") {
		q := req.URL.Query()
		if q.Get("alt") == "" {
			q.Set("alt", "sse")
			req.URL.RawQuery = q.Encode()
		}
	}
}

func (geminiAdapter) ParseNonStreamUsage(body []byte) parsedUsage {
	return geminiUsageFromBody(body)
}

func (geminiAdapter) ObserveStreamData(data []byte, metrics *streamMetrics, elapsedMs int64) {
	if metrics.FirstTokenMs == 0 && bytes.Contains(data, []byte(`"text"`)) {
		metrics.FirstTokenMs = elapsedMs
	}
	mergeUsage(&metrics.Usage, geminiUsageFromBody(data))
}

func (geminiAdapter) ProbeModelsPath() string { return "/v1beta/models" }

func (geminiAdapter) ParseModels(body []byte) ([]string, error) {
	var parsed struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	modelIDs := make([]string, 0, len(parsed.Models))
	for _, item := range parsed.Models {
		id := strings.TrimSpace(strings.TrimPrefix(item.Name, "models/"))
		if id != "" {
			modelIDs = append(modelIDs, id)
		}
	}
	return modelIDs, nil
}

func (geminiAdapter) BuildProbeRequest(modelID string) probeRequest {
	body, _ := json.Marshal(map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": "hi"},
				},
			},
		},
	})
	escapedModel := url.PathEscape(modelID)
	return probeRequest{Path: "/v1beta/models/" + escapedModel + ":generateContent", Body: body}
}

func geminiModelFromPath(path string) string {
	prefix := "/v1beta/models/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	value := strings.TrimPrefix(path, prefix)
	value = strings.TrimSuffix(value, ":generateContent")
	value = strings.TrimSuffix(value, ":streamGenerateContent")
	decoded, err := url.PathUnescape(value)
	if err == nil {
		value = decoded
	}
	return strings.TrimSpace(value)
}

func parseOpenAIStyleModels(body []byte) ([]string, error) {
	var parsed struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	modelIDs := make([]string, 0, len(parsed.Data))
	for _, item := range parsed.Data {
		if id := strings.TrimSpace(item.ID); id != "" {
			modelIDs = append(modelIDs, id)
		}
	}
	return modelIDs, nil
}

type usageParser func(json.RawMessage) parsedUsage

func extractUsageFromBody(body []byte, parser usageParser) parsedUsage {
	var envelope struct {
		Usage json.RawMessage `json:"usage"`
	}
	if json.Unmarshal(body, &envelope) != nil || len(envelope.Usage) == 0 {
		return parsedUsage{}
	}
	return parser(envelope.Usage)
}

func openAIUsageParser(raw json.RawMessage) parsedUsage {
	var u struct {
		PromptTokens     int64 `json:"prompt_tokens"`
		CompletionTokens int64 `json:"completion_tokens"`
		InputTokens      int64 `json:"input_tokens"`
		OutputTokens     int64 `json:"output_tokens"`
	}
	if json.Unmarshal(raw, &u) != nil {
		return parsedUsage{}
	}
	input := u.InputTokens
	if input == 0 {
		input = u.PromptTokens
	}
	output := u.OutputTokens
	if output == 0 {
		output = u.CompletionTokens
	}
	return parsedUsage{InputTokens: input, OutputTokens: output}
}

func geminiUsageFromBody(body []byte) parsedUsage {
	var envelope struct {
		UsageMetadata struct {
			PromptTokenCount     int64 `json:"promptTokenCount"`
			CandidatesTokenCount int64 `json:"candidatesTokenCount"`
		} `json:"usageMetadata"`
	}
	if json.Unmarshal(body, &envelope) != nil {
		return parsedUsage{}
	}
	return parsedUsage{
		InputTokens:  envelope.UsageMetadata.PromptTokenCount,
		OutputTokens: envelope.UsageMetadata.CandidatesTokenCount,
	}
}

func openAIEventHasTextDelta(event map[string]json.RawMessage) bool {
	if raw, ok := event["choices"]; ok {
		var choices []struct {
			Delta struct {
				Content interface{} `json:"content"`
			} `json:"delta"`
		}
		if json.Unmarshal(raw, &choices) == nil {
			for _, choice := range choices {
				if textLikeContent(choice.Delta.Content) {
					return true
				}
			}
		}
	}
	if raw, ok := event["delta"]; ok {
		var delta string
		if json.Unmarshal(raw, &delta) == nil && strings.TrimSpace(delta) != "" {
			return true
		}
	}
	if raw, ok := event["response"]; ok && bytes.Contains(raw, []byte(`"output"`)) {
		return true
	}
	return false
}

func textLikeContent(value interface{}) bool {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v) != ""
	case []interface{}:
		return len(v) > 0
	default:
		return v != nil
	}
}
