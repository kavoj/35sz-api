package doubao

import (
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

func TestConvertToRequestPayloadUsesTopLevelAgentPlanVideoFields(t *testing.T) {
	generateAudio := true
	watermark := false
	req := &relaycommon.TaskSubmitReq{
		Model:         "doubao-seedance-2.0",
		GenerateAudio: &generateAudio,
		Ratio:         "adaptive",
		Duration:      5,
		Watermark:     &watermark,
		Content: []map[string]interface{}{
			{"type": "text", "text": "女孩抱着狐狸，镜头缓缓拉出"},
			{"type": "image_url", "image_url": map[string]interface{}{"url": "https://example.com/first.png"}},
		},
	}

	adaptor := &TaskAdaptor{}
	payload, err := adaptor.convertToRequestPayload(req)
	if err != nil {
		t.Fatalf("convertToRequestPayload returned error: %v", err)
	}

	if payload.GenerateAudio == nil || !bool(*payload.GenerateAudio) {
		t.Fatalf("GenerateAudio = %#v, want true", payload.GenerateAudio)
	}
	if payload.Ratio != "adaptive" {
		t.Fatalf("Ratio = %q, want adaptive", payload.Ratio)
	}
	if payload.Duration == nil || int(*payload.Duration) != 5 {
		t.Fatalf("Duration = %#v, want 5", payload.Duration)
	}
	if payload.Watermark == nil || bool(*payload.Watermark) {
		t.Fatalf("Watermark = %#v, want false", payload.Watermark)
	}
	if len(payload.Content) != 2 {
		t.Fatalf("Content len = %d, want 2: %#v", len(payload.Content), payload.Content)
	}
	if payload.Content[0].Type != "text" || payload.Content[0].Text == "" {
		t.Fatalf("text content not preserved: %#v", payload.Content[0])
	}
	if payload.Content[1].Type != "image_url" || payload.Content[1].ImageURL == nil || payload.Content[1].ImageURL.URL == "" {
		t.Fatalf("image content not preserved: %#v", payload.Content[1])
	}
}
