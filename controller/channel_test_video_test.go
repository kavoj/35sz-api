package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShouldUseTaskChannelTestForVolcEngineVideoModel(t *testing.T) {
	channel := &model.Channel{
		Type:   constant.ChannelTypeVolcEngine,
		Models: "doubao-seedance-2.0",
	}

	assert.True(t, shouldUseTaskChannelTest(channel, "doubao-seedance-2.0"))
}

func TestBuildAgentPlanVideoTestRequestUsesDocumentedFields(t *testing.T) {
	payload := buildAgentPlanVideoTestRequest("doubao-seedance-2.0")

	assert.Equal(t, "doubao-seedance-2.0", payload["model"])
	assert.NotContains(t, payload, "draft")
	assert.Equal(t, false, payload["generate_audio"])
	assert.Equal(t, "adaptive", payload["ratio"])
	assert.Equal(t, 5, payload["duration"])
	assert.Equal(t, false, payload["watermark"])

	content, ok := payload["content"].([]map[string]interface{})
	require.True(t, ok)
	require.Len(t, content, 1)
	assert.Equal(t, "text", content[0]["type"])
	assert.NotEmpty(t, content[0]["text"])
}
