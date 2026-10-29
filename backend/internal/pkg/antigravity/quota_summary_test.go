//go:build unit

package antigravity

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRetrieveUserQuotaSummary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1internal:retrieveUserQuotaSummary" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer quota-token" || r.Header.Get("User-Agent") != "antigravity/acp/1.3.0" {
			t.Error("missing quota client credentials/identity")
		}
		var body FetchAvailableModelsRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Project != "quota-project" {
			t.Errorf("unexpected request body: %+v (%v)", body, err)
		}
		_, _ = fmt.Fprint(w, `{"groups":[{"displayName":"Claude and GPT models","buckets":[{"bucketId":"3p-5h","window":"5h","remainingFraction":0},{"bucketId":"3p-weekly","window":"weekly"}]}]}`)
	}))
	defer server.Close()
	withMockBaseURLs(t, []string{server.URL})
	client, err := NewClient("")
	require.NoError(t, err)
	summary, err := client.RetrieveUserQuotaSummary(context.Background(), "quota-token", "quota-project", 4096)
	require.NoError(t, err)
	require.Len(t, summary.Groups, 1)
	require.NotNil(t, summary.Groups[0].Buckets[0].RemainingFraction)
	require.Zero(t, *summary.Groups[0].Buckets[0].RemainingFraction)
	require.Nil(t, summary.Groups[0].Buckets[1].RemainingFraction)
}

func TestRetrieveUserQuotaSummaryFallbackAndValidation(t *testing.T) {
	for _, tc := range []struct {
		name     string
		status   int
		body     string
		limit    int64
		fallback bool
	}{
		{"server error", 503, "", 4096, true},
		{"empty summary", 200, `{}`, 4096, true},
		{"forbidden", 403, "sensitive provider error", 4096, false},
		{"unauthorized", 401, "sensitive provider error", 4096, false},
		{"malformed", 200, `{`, 4096, false},
		{"oversized", 200, strings.Repeat(" ", 100), 10, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fallbackCalls := 0
			fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				fallbackCalls++
				_, _ = fmt.Fprint(w, `{"buckets":[{"bucketId":"gemini-weekly","remainingFraction":1}]}`)
			}))
			defer fallback.Close()
			first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(tc.status); _, _ = fmt.Fprint(w, tc.body) }))
			defer first.Close()
			withMockBaseURLs(t, []string{first.URL, fallback.URL})
			client, err := NewClient("")
			require.NoError(t, err)
			_, err = client.RetrieveUserQuotaSummary(context.Background(), "token", "project", tc.limit)
			if tc.fallback {
				require.NoError(t, err)
				require.Equal(t, 1, fallbackCalls)
			} else {
				require.Error(t, err)
				require.NotContains(t, err.Error(), "sensitive provider error")
				require.Zero(t, fallbackCalls)
			}
		})
	}
}

func TestRetrieveUserQuotaSummaryDoesNotFollowRedirect(t *testing.T) {
	redirected := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect-target" {
			redirected = true
			return
		}
		http.Redirect(w, r, "/redirect-target", http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	withMockBaseURLs(t, []string{server.URL})
	client, err := NewClient("")
	require.NoError(t, err)
	_, err = client.RetrieveUserQuotaSummary(context.Background(), "token", "project", 4096)
	require.ErrorContains(t, err, "307")
	require.False(t, redirected)
}

func TestModelQuotaInfoMissingFraction(t *testing.T) {
	for _, tc := range []struct {
		raw   string
		known bool
	}{
		{`{}`, false}, {`{"remainingFraction":null}`, false}, {`{"remainingFraction":0}`, true},
		{`{"remainingFraction":1}`, true}, {`{"remainingFraction":-0.1}`, false}, {`{"remainingFraction":1.1}`, false},
	} {
		var quota ModelQuotaInfo
		require.NoError(t, json.Unmarshal([]byte(tc.raw), &quota))
		require.Equal(t, tc.known, quota.HasRemainingFraction(), tc.raw)
	}
}
