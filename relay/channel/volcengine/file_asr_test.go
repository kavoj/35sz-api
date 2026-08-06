package volcengine

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileASRHeadersNewAndLegacyAuth(t *testing.T) {
	ark := fileASRHeaders("ark-test", asrDefaultResourceID, "req-1")
	assert.Equal(t, "ark-test", ark.Get("X-Api-Key"))
	assert.Equal(t, asrDefaultResourceID, ark.Get("X-Api-Resource-Id"))
	assert.Equal(t, "req-1", ark.Get("X-Api-Request-Id"))

	legacy := fileASRHeaders("appid|token", fileASRBigModelResourceID, "req-2")
	assert.Equal(t, "appid", legacy.Get("X-Api-App-Key"))
	assert.Equal(t, "token", legacy.Get("X-Api-Access-Key"))
	assert.Equal(t, fileASRBigModelResourceID, legacy.Get("X-Api-Resource-Id"))
}

func TestFileASRResourceID(t *testing.T) {
	assert.Equal(t, fileASRBigModelResourceID, fileASRResourceID("volc.bigasr.auc"))
	assert.Equal(t, asrDefaultResourceID, fileASRResourceID("doubao-seed-asr-2.0"))
}

func TestQueryFileASRTreatsEmptyBodyAsProcessing(t *testing.T) {
	oldURL := fileASRQueryURL
	defer func() { fileASRQueryURL = oldURL }()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()
	fileASRQueryURL = server.URL

	result, done, err := queryFileASR(context.Background(), server.Client(), "ark-test", asrDefaultResourceID, "request-1")
	require.NoError(t, err)
	assert.False(t, done)
	assert.Nil(t, result)
}

func TestPollFileASRUsesSameRequestIDAndMapsWords(t *testing.T) {
	oldURL := fileASRQueryURL
	defer func() { fileASRQueryURL = oldURL }()
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		assert.Equal(t, "request-1", r.Header.Get("X-Api-Request-Id"))
		assert.Equal(t, "request-1", r.Header.Get("X-Api-Connect-Id"))
		if calls == 1 {
			_, _ = w.Write([]byte("{}"))
			return
		}
		_, _ = w.Write([]byte(`{"text":"hello","utterances":[{"text":"hello","start_time":1,"end_time":2,"words":[{"text":"hello","start_time":1,"end_time":2}]}]}`))
	}))
	defer server.Close()
	fileASRQueryURL = server.URL

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result, err := pollFileASR(ctx, server.Client(), "ark-test", asrDefaultResourceID, "request-1")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "hello", result.Text)
	require.Len(t, result.Utterances, 1)
	require.Len(t, result.Utterances[0].Words, 1)
}
