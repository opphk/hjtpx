package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type LogSearchService struct {
	lokiURL string
	client  *http.Client
}

type LogQuery struct {
	Query       string            `json:"query"`
	Start       time.Time         `json:"start"`
	End         time.Time         `json:"end"`
	Limit       int               `json:"limit"`
	Direction   string            `json:"direction"`
	Regex       bool              `json:"regex"`
	Filter      map[string]string `json:"filter,omitempty"`
}

type LogQueryResult struct {
	Streams    []LogStream `json:"streams"`
	Stats      QueryStats  `json:"stats"`
}

type LogStream struct {
	Stream    map[string]string `json:"stream"`
	Values    [][2]string      `json:"values"`
	Timestamp time.Time         `json:"-"`
}

type LogEntryParsed struct {
	Timestamp  time.Time         `json:"timestamp"`
	Message    string            `json:"message"`
	Stream     map[string]string `json:"stream"`
	Labels     map[string]string `json:"labels,omitempty"`
	Level      string            `json:"level,omitempty"`
	Component  string            `json:"component,omitempty"`
	TraceID    string            `json:"trace_id,omitempty"`
	Error      string            `json:"error,omitempty"`
	Duration   time.Duration     `json:"duration,omitempty"`
	Metadata   map[string]any    `json:"metadata,omitempty"`
}

type QueryStats struct {
	LinesMatched  int     `json:"linesMatched"`
	LinesSent    int     `json:"linesSent"`
	BytesMatched int     `json:"bytesMatched"`
	ExecTime     float64 `json:"execTime"`
}

type LogSeries struct {
	Series []SeriesInfo `json:"series"`
}

type SeriesInfo struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels"`
}

type LogLabelValues struct {
	Label   string   `json:"label"`
	Values  []string `json:"values"`
}

type LogStatsResult struct {
	Ingester LogStats `json:"ingester"`
}

type LogStats struct {
	Store struct {
		TotalChunksRef        int     `json:"totalChunksRef"`
		TotalChunksDownloaded int     `json:"totalChunksDownloaded"`
		ChunksDownloaded      int     `json:"chunksDownloaded"`
		ChunksLookups         int     `json:"chunksLookups"`
		ChunksDownloadTime    float64 `json:"chunksDownloadTime"`
		HeadChunksLookups     int     `json:"headChunksLookups"`
		HeadChunksFound       int     `json:"headChunksFound"`
	} `json:"store"`
	Ingester struct {
		TotalReached          int `json:"totalReached"`
		TotalChunksMatched    int `json:"totalChunksMatched"`
		TotalBatches          int `json:"totalBatches"`
		TotalLinesSent        int `json:"totalLinesSent"`
		StoreChunksDownloaded int `json:"storeChunksDownloaded"`
	} `json:"ingester"`
}

func NewLogSearchService(lokiURL string) *LogSearchService {
	return &LogSearchService{
		lokiURL: lokiURL,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (lss *LogSearchService) Query(ctx context.Context, query LogQuery) (*LogQueryResult, error) {
	if query.Limit == 0 {
		query.Limit = 100
	}
	if query.Direction == "" {
		query.Direction = "backward"
	}

	params := fmt.Sprintf("?query=%s&start=%d&end=%d&limit=%d&direction=%s",
		query.Query,
		query.Start.UnixNano(),
		query.End.UnixNano(),
		query.Limit,
		query.Direction,
	)

	if query.Regex {
		params += "&regex=true"
	}

	url := lss.lokiURL + "/loki/api/v1/query_range" + params
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := lss.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("loki query failed: %s", string(body))
	}

	var rawResult struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Stream map[string]string `json:"stream"`
				Values [][2]string      `json:"values"`
			} `json:"result"`
			Stats QueryStats `json:"stats"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rawResult); err != nil {
		return nil, err
	}

	result := &LogQueryResult{
		Streams: make([]LogStream, 0, len(rawResult.Data.Result)),
		Stats:   rawResult.Data.Stats,
	}

	for _, stream := range rawResult.Data.Result {
		logStream := LogStream{
			Stream: stream.Stream,
			Values: stream.Values,
		}
		if len(stream.Values) > 0 {
			ts, _ := time.Parse("2006-01-02T15:04:05.999999999Z", stream.Values[0][0])
			logStream.Timestamp = ts
		}
		result.Streams = append(result.Streams, logStream)
	}

	return result, nil
}

func (lss *LogSearchService) QueryInstant(ctx context.Context, query string) (*LogQueryResult, error) {
	url := fmt.Sprintf("%s/loki/api/v1/query?query=%s", lss.lokiURL, query)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := lss.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("loki instant query failed: %s", string(body))
	}

	var rawResult struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Stream map[string]string `json:"stream"`
				Values [][2]string      `json:"values"`
			} `json:"result"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rawResult); err != nil {
		return nil, err
	}

	result := &LogQueryResult{
		Streams: make([]LogStream, 0, len(rawResult.Data.Result)),
	}

	for _, stream := range rawResult.Data.Result {
		logStream := LogStream{
			Stream: stream.Stream,
			Values: stream.Values,
		}
		if len(stream.Values) > 0 {
			ts, _ := time.Parse("2006-01-02T15:04:05.999999999Z", stream.Values[0][0])
			logStream.Timestamp = ts
		}
		result.Streams = append(result.Streams, logStream)
	}

	return result, nil
}

func (lss *LogSearchService) GetSeries(ctx context.Context, start, end time.Time) (*LogSeries, error) {
	url := fmt.Sprintf("%s/loki/api/v1/series?start=%d&end=%d",
		lss.lokiURL,
		start.UnixNano(),
		end.UnixNano(),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := lss.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("loki series query failed: %s", string(body))
	}

	var series LogSeries
	if err := json.NewDecoder(resp.Body).Decode(&series); err != nil {
		return nil, err
	}

	return &series, nil
}

func (lss *LogSearchService) GetLabelValues(ctx context.Context, labelName string, start, end time.Time) (*LogLabelValues, error) {
	url := fmt.Sprintf("%s/loki/api/v1/label/%s?start=%d&end=%d",
		lss.lokiURL,
		labelName,
		start.UnixNano(),
		end.UnixNano(),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := lss.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("loki label values query failed: %s", string(body))
	}

	var rawResult struct {
		Status string   `json:"status"`
		Data   []string `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rawResult); err != nil {
		return nil, err
	}

	return &LogLabelValues{
		Label:  labelName,
		Values: rawResult.Data,
	}, nil
}

func (lss *LogSearchService) GetStats(ctx context.Context, query string, start, end time.Time) (*LogStatsResult, error) {
	url := fmt.Sprintf("%s/loki/api/v1/query_range?query=%s&start=%d&end=%d&limit=0",
		lss.lokiURL,
		query,
		start.UnixNano(),
		end.UnixNano(),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := lss.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("loki stats query failed: %s", string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &LogStatsResult{
		Ingester: LogStats{
			Store: struct {
				TotalChunksRef        int     `json:"totalChunksRef"`
				TotalChunksDownloaded int     `json:"totalChunksDownloaded"`
				ChunksDownloaded      int     `json:"chunksDownloaded"`
				ChunksLookups         int     `json:"chunksLookups"`
				ChunksDownloadTime    float64 `json:"chunksDownloadTime"`
				HeadChunksLookups     int     `json:"headChunksLookups"`
				HeadChunksFound       int     `json:"headChunksFound"`
			}{},
			Ingester: struct {
				TotalReached          int `json:"totalReached"`
				TotalChunksMatched    int `json:"totalChunksMatched"`
				TotalBatches          int `json:"totalBatches"`
				TotalLinesSent        int `json:"totalLinesSent"`
				StoreChunksDownloaded int `json:"storeChunksDownloaded"`
			}{},
		},
	}, nil
}

func (lss *LogSearchService) ParseLogEntry(stream LogStream, value [2]string) LogEntryParsed {
	entry := LogEntryParsed{
		Stream: stream.Stream,
	}

	if len(value) >= 2 {
		ts, err := time.Parse("2006-01-02T15:04:05.999999999Z", value[0])
		if err == nil {
			entry.Timestamp = ts
		}
		entry.Message = value[1]
	}

	if level, ok := stream.Stream["level"]; ok {
		entry.Level = level
	}
	if component, ok := stream.Stream["component"]; ok {
		entry.Component = component
	}

	return entry
}

func (lss *LogSearchService) Search(ctx context.Context, query LogQuery) ([]LogEntryParsed, error) {
	result, err := lss.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	entries := make([]LogEntryParsed, 0)
	for _, stream := range result.Streams {
		for _, value := range stream.Values {
			entry := lss.ParseLogEntry(stream, value)
			entries = append(entries, entry)
		}
	}

	return entries, nil
}

func (lss *LogSearchService) BuildQuery(filters map[string]string, levels []string, search string) string {
	queryParts := make([]string, 0)

	for key, value := range filters {
		queryParts = append(queryParts, fmt.Sprintf(`%s="%s"`, key, value))
	}

	if len(levels) > 0 {
		levelQuery := "{"
		for i, level := range levels {
			if i > 0 {
				levelQuery += ","
			}
			levelQuery += fmt.Sprintf(`level="%s"`, level)
		}
		levelQuery += "}"
		queryParts = append(queryParts, levelQuery)
	}

	if search != "" {
		queryParts = append(queryParts, fmt.Sprintf(`"%s"`, search))
	}

	if len(queryParts) == 0 {
		return "{}"
	}

	query := queryParts[0]
	for i := 1; i < len(queryParts); i++ {
		query += " | " + queryParts[i]
	}

	return query
}
