package volcengine

import (
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SeedASR nests the transcription under "result", and sends no event bits on
// responses. Parsing the OpenAI-facing text therefore depends on reading
// result.text; a top-level "text" lookup silently yields "".
func TestASRServerResponseParsing(t *testing.T) {
	msg, err := NewMessage(MsgTypeFullServerResponse, MsgTypeFlagPositiveSeq)
	require.NoError(t, err)
	msg.Payload = []byte(`{"audio_info":{"duration":1000},"result":{"text":"你好，今天天气很好。","utterances":[{"text":"你好，今天天气很好。","definite":true}]}}`)

	payload, err := decodeASRPayload(msg)
	require.NoError(t, err)

	var resp asrServerResponse
	require.NoError(t, common.Unmarshal(payload, &resp))
	assert.Equal(t, "你好，今天天气很好。", resp.Result.Text)
	require.Len(t, resp.Result.Utterances, 1)
	assert.True(t, resp.Result.Utterances[0].Definite)
}

// A gzip-compressed frame must be inflated before JSON parsing; the server sets
// the compression bit independently of the request.
func TestDecodeASRPayloadGzip(t *testing.T) {
	body := []byte(`{"result":{"text":"压缩结果"}}`)
	compressed, err := gzipCompress(body)
	require.NoError(t, err)

	msg, err := NewMessage(MsgTypeFullServerResponse, MsgTypeFlagPositiveSeq)
	require.NoError(t, err)
	msg.Compression = CompressionGzip
	msg.Payload = compressed

	got, err := decodeASRPayload(msg)
	require.NoError(t, err)
	assert.Equal(t, body, got)

	// Uncompressed frames pass through untouched.
	plain, err := NewMessage(MsgTypeFullServerResponse, MsgTypeFlagPositiveSeq)
	require.NoError(t, err)
	plain.Payload = body
	got, err = decodeASRPayload(plain)
	require.NoError(t, err)
	assert.Equal(t, body, got)
}

// The ASR resource id travels as a header and is distinct from the protocol
// model name in the init payload. Ark console keys have no "|" separator and
// must be sent verbatim as X-Api-Key; legacy keys keep the App/Access pair.
func TestBuildWebSocketRequestHeaderAuthSchemes(t *testing.T) {
	arkHeader, err := buildWebSocketRequestHeader("ark-abc-123", asrDefaultResourceID)
	require.NoError(t, err)
	assert.Equal(t, "ark-abc-123", arkHeader.Get("X-Api-Key"))
	assert.Equal(t, asrDefaultResourceID, arkHeader.Get("X-Api-Resource-Id"))
	assert.NotEmpty(t, arkHeader.Get("X-Api-Connect-Id"))
	assert.Empty(t, arkHeader.Get("X-Api-App-Key"))
	// The TTS scheme is rejected by SeedASR with 401 Invalid X-Api-Key.
	assert.Empty(t, arkHeader.Get("Authorization"))

	legacyHeader, err := buildWebSocketRequestHeader("6543210|tok-xyz", asrDefaultResourceID)
	require.NoError(t, err)
	assert.Equal(t, "6543210", legacyHeader.Get("X-Api-App-Key"))
	assert.Equal(t, "tok-xyz", legacyHeader.Get("X-Api-Access-Key"))
	assert.Empty(t, legacyHeader.Get("X-Api-Key"))

	_, err = buildWebSocketRequestHeader("   ", asrDefaultResourceID)
	assert.Error(t, err)
}

func TestASRResourceID(t *testing.T) {
	assert.Equal(t, "volc.seedasr.sauc.duration", asrResourceID("volc.seedasr.sauc.duration"))
	assert.Equal(t, asrDefaultResourceID, asrResourceID("doubao-seed-asr-2.0"))
	assert.Equal(t, asrDefaultResourceID, asrResourceID(""))
	// The init payload always carries the protocol model, not the resource id.
	assert.Equal(t, "bigmodel", buildASRInitPayload(asrProtocolModelName).Request.ModelName)
}
