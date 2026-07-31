package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/dto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildTestRequestAutoVolcSeedreamUsesImageRequest(t *testing.T) {
	channel := &model.Channel{Type: constant.ChannelTypeVolcEngine}
	request := buildTestRequest("doubao-seedream-4-5-251128", "", channel, false)
	imageRequest, ok := request.(*dto.ImageRequest)
	require.True(t, ok)
	assert.Equal(t, "2K", imageRequest.Size)
}

func TestIsAudioTranscriptionModel(t *testing.T) {
	tests := map[string]bool{
		"volc.seedasr.sauc.duration": true,
		"doubao-seed-asr-2.0":        true,
		"volc.bigasr.sauc.duration":  true,
		"ASR":                        true,
		"whisper-1":                  false,
		"seed-tts-2.0":               false,
		"asratchet-v1":               false,
		"disaster-model":             false,
	}

	for modelName, want := range tests {
		assert.Equalf(t, want, isAudioTranscriptionModel(modelName), "model=%q", modelName)
	}
}

func TestBuildTestRequestASRModelUsesAudioRequest(t *testing.T) {
	channel := &model.Channel{Type: constant.ChannelTypeVolcEngine}
	request := buildTestRequest("volc.seedasr.sauc.duration", "", channel, false)
	audioRequest, ok := request.(*dto.AudioRequest)
	require.True(t, ok)
	assert.Equal(t, "volc.seedasr.sauc.duration", audioRequest.Model)
}
