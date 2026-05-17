package metrics

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogSearchService(t *testing.T) {
	lss := NewLogSearchService("http://loki:3100")
	require.NotNil(t, lss)
	assert.Equal(t, "http://loki:3100", lss.lokiURL)
	assert.NotNil(t, lss.client)
}

func TestBuildQuery(t *testing.T) {
	lss := NewLogSearchService("http://loki:3100")

	query := lss.BuildQuery(
		map[string]string{"job": "api", "env": "production"},
		[]string{"error", "warning"},
		"failed",
	)
	assert.Contains(t, query, `job="api"`)
	assert.Contains(t, query, `env="production"`)
	assert.Contains(t, query, `level="error"`)
	assert.Contains(t, query, `level="warning"`)
	assert.Contains(t, query, "failed")
}

func TestBuildQueryEmptyFilters(t *testing.T) {
	lss := NewLogSearchService("http://loki:3100")

	query := lss.BuildQuery(nil, nil, "")
	assert.Equal(t, "{}", query)

	query = lss.BuildQuery(map[string]string{"job": ""}, nil, "")
	assert.Contains(t, query, `job=""`)
}

func TestBuildQueryWithLevels(t *testing.T) {
	lss := NewLogSearchService("http://loki:3100")

	query := lss.BuildQuery(
		map[string]string{"job": "test"},
		[]string{"info", "debug"},
		"",
	)
	assert.Contains(t, query, `job="test"`)
	assert.Contains(t, query, `level="info"`)
	assert.Contains(t, query, `level="debug"`)
}

func TestBuildQueryWithSearch(t *testing.T) {
	lss := NewLogSearchService("http://loki:3100")

	query := lss.BuildQuery(
		map[string]string{"job": "api"},
		nil,
		"database error",
	)
	assert.Contains(t, query, `job="api"`)
	assert.Contains(t, query, "database error")
}

func TestParseLogEntry(t *testing.T) {
	lss := NewLogSearchService("http://loki:3100")

	stream := LogStream{
		Stream: map[string]string{
			"job":       "api",
			"level":     "error",
			"component": "database",
		},
	}

	value := [2]string{
		"2024-01-15T10:30:00.123456789Z",
		"connection refused",
	}

	entry := lss.ParseLogEntry(stream, value)

	assert.Equal(t, "connection refused", entry.Message)
	assert.Equal(t, "error", entry.Level)
	assert.Equal(t, "database", entry.Component)
	assert.Equal(t, "api", entry.Stream["job"])
}

func TestParseLogEntryInvalidTimestamp(t *testing.T) {
	lss := NewLogSearchService("http://loki:3100")

	stream := LogStream{
		Stream: map[string]string{"job": "test"},
	}

	value := [2]string{
		"invalid-timestamp",
		"test message",
	}

	entry := lss.ParseLogEntry(stream, value)
	assert.Equal(t, "test message", entry.Message)
}

func TestParseLogEntryEmptyValue(t *testing.T) {
	lss := NewLogSearchService("http://loki:3100")

	stream := LogStream{
		Stream: map[string]string{"job": "test"},
	}

	entry := lss.ParseLogEntry(stream, [2]string{})
	assert.Empty(t, entry.Message)
}

func TestLogQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/loki/api/v1/query_range", r.URL.Path)
		assert.Contains(t, r.URL.RawQuery, "query=")
		assert.Contains(t, r.URL.RawQuery, "start=")
		assert.Contains(t, r.URL.RawQuery, "end=")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "streams",
				"result": []map[string]interface{}{
					{
						"stream": map[string]string{"job": "test"},
						"values": [][2]string{
							{"2024-01-15T10:30:00.000000000Z", "test message"},
						},
					},
				},
				"stats": QueryStats{
					LinesMatched: 1,
					LinesSent:    1,
					BytesMatched: 100,
					ExecTime:     0.05,
				},
			},
		})
	}))
	defer server.Close()

	lss := NewLogSearchService(server.URL)

	ctx := context.Background()
	result, err := lss.Query(ctx, LogQuery{
		Query:     `{job="test"}`,
		Start:     time.Now().Add(-1 * time.Hour),
		End:       time.Now(),
		Limit:     100,
		Direction: "backward",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Streams, 1)
}

func TestLogQueryInstant(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/loki/api/v1/query")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "streams",
				"result": []map[string]interface{}{
					{
						"stream": map[string]string{"job": "api"},
						"values": [][2]string{
							{"2024-01-15T10:30:00.000000000Z", "api request"},
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	lss := NewLogSearchService(server.URL)

	ctx := context.Background()
	result, err := lss.QueryInstant(ctx, `{job="api"}`)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Streams, 1)
}

func TestLogQueryError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad Request"))
	}))
	defer server.Close()

	lss := NewLogSearchService(server.URL)

	ctx := context.Background()
	result, err := lss.Query(ctx, LogQuery{
		Query: "invalid query",
		Start: time.Now().Add(-1 * time.Hour),
		End:   time.Now(),
	})

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestGetSeries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/loki/api/v1/series")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"series": []map[string]interface{}{
				{
					"name":   "logs",
					"labels": map[string]string{"job": "test"},
				},
			},
		})
	}))
	defer server.Close()

	lss := NewLogSearchService(server.URL)

	ctx := context.Background()
	result, err := lss.GetSeries(ctx, time.Now().Add(-1*time.Hour), time.Now())

	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestGetLabelValues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/loki/api/v1/label/")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"data":   []string{"info", "warning", "error"},
		})
	}))
	defer server.Close()

	lss := NewLogSearchService(server.URL)

	ctx := context.Background()
	result, err := lss.GetLabelValues(ctx, "level", time.Now().Add(-1*time.Hour), time.Now())

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "level", result.Label)
}

func TestGetStats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"stats": map[string]interface{}{
					"ingester": map[string]interface{}{
						"store": map[string]interface{}{
							"totalChunksRef": 100,
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	lss := NewLogSearchService(server.URL)

	ctx := context.Background()
	_, err := lss.GetStats(ctx, `{job="test"}`, time.Now().Add(-1*time.Hour), time.Now())
	assert.NoError(t, err)
}

func TestSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "streams",
				"result": []map[string]interface{}{
					{
						"stream": map[string]string{"job": "api", "level": "error"},
						"values": [][2]string{
							{"2024-01-15T10:30:00.000000000Z", "error message"},
						},
					},
				},
				"stats": QueryStats{
					LinesMatched: 1,
				},
			},
		})
	}))
	defer server.Close()

	lss := NewLogSearchService(server.URL)

	ctx := context.Background()
	entries, err := lss.Search(ctx, LogQuery{
		Query:     `{job="api"}`,
		Start:     time.Now().Add(-1 * time.Hour),
		End:       time.Now(),
		Limit:     100,
		Direction: "backward",
	})

	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "error message", entries[0].Message)
}

func TestLogQueryDefaultLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "streams",
				"result":     []map[string]interface{}{},
				"stats":     QueryStats{},
			},
		})
	}))
	defer server.Close()

	lss := NewLogSearchService(server.URL)

	ctx := context.Background()
	_, err := lss.Query(ctx, LogQuery{
		Query: `{job="test"}`,
		Start: time.Now().Add(-1 * time.Hour),
		End:   time.Now(),
	})

	require.NoError(t, err)
}

func TestLogQueryDefaultDirection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "streams",
				"result":     []map[string]interface{}{},
				"stats":     QueryStats{},
			},
		})
	}))
	defer server.Close()

	lss := NewLogSearchService(server.URL)

	ctx := context.Background()
	_, err := lss.Query(ctx, LogQuery{
		Query: `{job="test"}`,
		Start: time.Now().Add(-1 * time.Hour),
		End:   time.Now(),
		Limit: 100,
	})

	require.NoError(t, err)
}
