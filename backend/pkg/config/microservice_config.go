package config

import (
	"fmt"
	"os"
	"time"
)

type MicroserviceConfig struct {
	Enabled           bool                  `yaml:"enabled"`
	Services          map[string]ServiceConfig `yaml:"services"`
	GRPC              GRPCConfig            `yaml:"grpc"`
	ServiceDiscovery  ServiceDiscoveryConfig `yaml:"service_discovery"`
	CircuitBreaker   CircuitBreakerConfig  `yaml:"circuit_breaker"`
	GlobalAcceleration GlobalAccelerationConfig `yaml:"global_acceleration"`
	Monitoring        MicroserviceMonitoringConfig `yaml:"monitoring"`
}

type ServiceConfig struct {
	Name        string            `yaml:"name"`
	Host        string            `yaml:"host"`
	Port        int               `yaml:"port"`
	GRPCPort    int               `yaml:"grpc_port"`
	HealthCheck HealthCheckConfig `yaml:"health_check"`
	Limits      LimitsConfig      `yaml:"limits"`
	Dependencies []string         `yaml:"dependencies"`
}

type HealthCheckConfig struct {
	Enabled  bool          `yaml:"enabled"`
	Interval time.Duration `yaml:"interval"`
	Timeout  time.Duration `yaml:"timeout"`
	Retries  int           `yaml:"retries"`
}

type LimitsConfig struct {
	MaxConnections int           `yaml:"max_connections"`
	Timeout        time.Duration `yaml:"timeout"`
	RateLimit      int           `yaml:"rate_limit"`
}

type GRPCConfig struct {
	Enabled     bool           `yaml:"enabled"`
	Port        int            `yaml:"port"`
	MaxMsgSize  int            `yaml:"max_msg_size"`
	KeepAlive   KeepAliveConfig `yaml:"keep_alive"`
	TLS         TLSConfig      `yaml:"tls"`
	Interceptors []string      `yaml:"interceptors"`
}

type KeepAliveConfig struct {
	MaxConnectionIdle     time.Duration `yaml:"max_connection_idle"`
	MaxConnectionAge      time.Duration `yaml:"max_connection_age"`
	MaxConnectionAgeGrace time.Duration `yaml:"max_connection_age_grace"`
	Time                 time.Duration `yaml:"time"`
	Timeout              time.Duration `yaml:"timeout"`
}

type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

type ServiceDiscoveryConfig struct {
	Enabled   bool                    `yaml:"enabled"`
	Provider  string                   `yaml:"provider"`
	Consul    ConsulConfig             `yaml:"consul"`
	Etcd      EtcdConfig               `yaml:"etcd"`
	HealthCheck HealthCheckConfig      `yaml:"health_check"`
}

type ConsulConfig struct {
	Address   string            `yaml:"address"`
	Port     int               `yaml:"port"`
	Datacenter string          `yaml:"datacenter"`
	Token    string            `yaml:"token"`
	Service  ServiceMetaConfig `yaml:"service"`
}

type EtcdConfig struct {
	Endpoints   []string          `yaml:"endpoints"`
	DialTimeout time.Duration   `yaml:"dial_timeout"`
	Username   string            `yaml:"username"`
	Password   string            `yaml:"password"`
	Service    ServiceMetaConfig `yaml:"service"`
}

type ServiceMetaConfig struct {
	Name        string            `yaml:"name"`
	ID          string            `yaml:"id"`
	Tags        []string          `yaml:"tags"`
	Meta        map[string]string `yaml:"meta"`
}

type CircuitBreakerConfig struct {
	Enabled     bool                       `yaml:"enabled"`
	Services    map[string]CircuitBreakerServiceConfig `yaml:"services"`
	Default     CircuitBreakerDefaultConfig `yaml:"default"`
}

type CircuitBreakerServiceConfig struct {
	Enabled           bool    `yaml:"enabled"`
	MaxRequests       int     `yaml:"max_requests"`
	Interval          int     `yaml:"interval"`
	Timeout           int     `yaml:"timeout"`
	FailureThreshold  float64 `yaml:"failure_threshold"`
	SuccessThreshold  float64 `yaml:"success_threshold"`
	HalfOpenMaxReqs   int     `yaml:"half_open_max_requests"`
}

type CircuitBreakerDefaultConfig struct {
	MaxRequests       int     `yaml:"max_requests"`
	Interval          int     `yaml:"interval"`
	Timeout           int     `yaml:"timeout"`
	FailureThreshold  float64 `yaml:"failure_threshold"`
	SuccessThreshold  float64 `yaml:"success_threshold"`
	HalfOpenMaxReqs   int     `yaml:"half_open_max_requests"`
}

type GlobalAccelerationConfig struct {
	Enabled         bool                      `yaml:"enabled"`
	CDN             CDNConfig                 `yaml:"cdn"`
	MultiRegion     MultiRegionConfig         `yaml:"multi_region"`
	SmartRouting    SmartRoutingConfig        `yaml:"smart_routing"`
	EdgeComputing   EdgeComputingConfig       `yaml:"edge_computing"`
}

type CDNConfig struct {
	Enabled        bool              `yaml:"enabled"`
	Provider       string            `yaml:"provider"`
	Endpoints      []CDNEndpoint    `yaml:"endpoints"`
	CachePolicy    CachePolicyConfig `yaml:"cache_policy"`
	SSLEnabled     bool              `yaml:"ssl_enabled"`
}

type CDNEndpoint struct {
	Name     string `yaml:"name"`
	URL      string `yaml:"url"`
	Region   string `yaml:"region"`
	Priority int    `yaml:"priority"`
}

type CachePolicyConfig struct {
	DefaultTTL     int      `yaml:"default_ttl"`
	MaxTTL         int      `yaml:"max_ttl"`
	MinTTL         int      `yaml:"min_ttl"`
	StaleTTL       int      `yaml:"stale_ttl"`
	CacheTypes     []string `yaml:"cache_types"`
	NoCacheHeaders []string `yaml:"no_cache_headers"`
}

type MultiRegionConfig struct {
	Enabled     bool                `yaml:"enabled"`
	Regions     []RegionConfig      `yaml:"regions"`
	Replication ReplicationConfig   `yaml:"replication"`
}

type RegionConfig struct {
	Name         string            `yaml:"name"`
	ID           string            `yaml:"id"`
	Endpoint     string            `yaml:"endpoint"`
	Priority     int               `yaml:"priority"`
	Weight       int               `yaml:"weight"`
	GeoTargeting bool              `yaml:"geo_targeting"`
	Meta         map[string]string `yaml:"meta"`
}

type ReplicationConfig struct {
	Mode          string `yaml:"mode"`
	AsyncEnabled  bool   `yaml:"async_enabled"`
	SyncInterval  int    `yaml:"sync_interval_secs"`
	ConflictRes   string `yaml:"conflict_resolution"`
}

type SmartRoutingConfig struct {
	Enabled          bool               `yaml:"enabled"`
	Strategy         string             `yaml:"strategy"`
	LatencyThreshold int                `yaml:"latency_threshold_ms"`
	HealthCheck      SmartHealthCheck   `yaml:"health_check"`
}

type SmartHealthCheck struct {
	Enabled    bool          `yaml:"enabled"`
	Interval   time.Duration `yaml:"interval"`
	Timeout    time.Duration `yaml:"timeout"`
	Retries    int           `yaml:"retries"`
}

type EdgeComputingConfig struct {
	Enabled     bool                 `yaml:"enabled"`
	Nodes       []EdgeNodeConfig     `yaml:"nodes"`
	Functions   []EdgeFunctionConfig `yaml:"functions"`
}

type EdgeNodeConfig struct {
	ID       string `yaml:"id"`
	Name     string `yaml:"name"`
	Region   string `yaml:"region"`
	Endpoint string `yaml:"endpoint"`
	Capacity int    `yaml:"capacity"`
}

type EdgeFunctionConfig struct {
	Name    string   `yaml:"name"`
	Type    string   `yaml:"type"`
	Code    string   `yaml:"code"`
	Runtime string   `yaml:"runtime"`
	Triggers []string `yaml:"triggers"`
}

type MicroserviceMonitoringConfig struct {
	Enabled           bool                  `yaml:"enabled"`
	OpenTelemetry     OpenTelemetryConfig   `yaml:"opentelemetry"`
	Profiling         ProfilingConfig       `yaml:"profiling"`
	LogAggregation    LogAggregationConfig  `yaml:"log_aggregation"`
	AlertAggregation  AlertAggregationConfig `yaml:"alert_aggregation"`
}

type OpenTelemetryConfig struct {
	Enabled     bool             `yaml:"enabled"`
	Endpoint    string           `yaml:"endpoint"`
	Insecure    bool             `yaml:"insecure"`
	ServiceName string           `yaml:"service_name"`
	ExportType  string           `yaml:"export_type"`
	Headers     map[string]string `yaml:"headers"`
	SamplingRate float64         `yaml:"sampling_rate"`
}

type ProfilingConfig struct {
	Enabled     bool     `yaml:"enabled"`
	Endpoints   []string `yaml:"endpoints"`
	Interval    int      `yaml:"interval_secs"`
	Types       []string `yaml:"types"`
}

type LogAggregationConfig struct {
	Enabled     bool       `yaml:"enabled"`
	Provider    string     `yaml:"provider"`
	Endpoints   []string  `yaml:"endpoints"`
	Aggregation AggregationPolicy `yaml:"aggregation_policy"`
}

type AggregationPolicy struct {
	ByService    bool `yaml:"by_service"`
	BySeverity   bool `yaml:"by_severity"`
	ByTimeWindow int  `yaml:"by_time_window_secs"`
	MaxBatchSize int  `yaml:"max_batch_size"`
	FlushInterval int `yaml:"flush_interval_secs"`
}

type AlertAggregationConfig struct {
	Enabled        bool           `yaml:"enabled"`
	Provider       string         `yaml:"provider"`
	GroupBy        []string       `yaml:"group_by"`
	TimeWindow     int            `yaml:"time_window_secs"`
	Threshold      int            `yaml:"threshold"`
	Deduplication  DeduplicationConfig `yaml:"deduplication"`
}

type DeduplicationConfig struct {
	Enabled    bool  `yaml:"enabled"`
	WindowSecs int   `yaml:"window_secs"`
	MaxCount   int   `yaml:"max_count"`
}

func (c *MicroserviceConfig) GetServiceConfig(serviceName string) (*ServiceConfig, error) {
	if service, ok := c.Services[serviceName]; ok {
		return &service, nil
	}
	return nil, fmt.Errorf("service %s not found in config", serviceName)
}

func (c *MicroserviceConfig) IsServiceEnabled(serviceName string) bool {
	if service, ok := c.Services[serviceName]; ok {
		return c.Enabled && service.Name != ""
	}
	return c.Enabled
}

func LoadMicroserviceConfig() *MicroserviceConfig {
	cfg := &MicroserviceConfig{
		Enabled: getEnvAsBoolMS("MICROSERVICE_ENABLED", false),
		Services: map[string]ServiceConfig{
			"captcha": {
				Name: "captcha-service",
				Host: getEnvMS("CAPTCHA_SERVICE_HOST", "localhost"),
				Port: getEnvAsIntMS("CAPTCHA_SERVICE_PORT", 8081),
				GRPCPort: getEnvAsIntMS("CAPTCHA_GRPC_PORT", 9091),
				HealthCheck: HealthCheckConfig{
					Enabled:  true,
					Interval: 10 * time.Second,
					Timeout:  5 * time.Second,
					Retries:  3,
				},
				Limits: LimitsConfig{
					MaxConnections: getEnvAsIntMS("CAPTCHA_MAX_CONN", 1000),
					Timeout:        30 * time.Second,
					RateLimit:      getEnvAsIntMS("CAPTCHA_RATE_LIMIT", 10000),
				},
				Dependencies: []string{"redis"},
			},
			"behavior": {
				Name: "behavior-service",
				Host: getEnvMS("BEHAVIOR_SERVICE_HOST", "localhost"),
				Port: getEnvAsIntMS("BEHAVIOR_SERVICE_PORT", 8082),
				GRPCPort: getEnvAsIntMS("BEHAVIOR_GRPC_PORT", 9092),
				HealthCheck: HealthCheckConfig{
					Enabled:  true,
					Interval: 10 * time.Second,
					Timeout:  5 * time.Second,
					Retries:  3,
				},
				Limits: LimitsConfig{
					MaxConnections: getEnvAsIntMS("BEHAVIOR_MAX_CONN", 500),
					Timeout:        60 * time.Second,
					RateLimit:      getEnvAsIntMS("BEHAVIOR_RATE_LIMIT", 5000),
				},
				Dependencies: []string{"captcha", "redis", "postgres"},
			},
			"analytics": {
				Name: "analytics-service",
				Host: getEnvMS("ANALYTICS_SERVICE_HOST", "localhost"),
				Port: getEnvAsIntMS("ANALYTICS_SERVICE_PORT", 8083),
				GRPCPort: getEnvAsIntMS("ANALYTICS_GRPC_PORT", 9093),
				HealthCheck: HealthCheckConfig{
					Enabled:  true,
					Interval: 15 * time.Second,
					Timeout:  5 * time.Second,
					Retries:  3,
				},
				Limits: LimitsConfig{
					MaxConnections: getEnvAsIntMS("ANALYTICS_MAX_CONN", 200),
					Timeout:        120 * time.Second,
					RateLimit:      getEnvAsIntMS("ANALYTICS_RATE_LIMIT", 1000),
				},
				Dependencies: []string{"postgres"},
			},
		},
		GRPC: GRPCConfig{
			Enabled:    getEnvAsBoolMS("GRPC_ENABLED", true),
			Port:       getEnvAsIntMS("GRPC_PORT", 9090),
			MaxMsgSize: getEnvAsIntMS("GRPC_MAX_MSG_SIZE", 4*1024*1024),
			KeepAlive: KeepAliveConfig{
				MaxConnectionIdle:    5 * time.Minute,
				MaxConnectionAge:      2 * time.Hour,
				MaxConnectionAgeGrace: 30 * time.Second,
				Time:                  1 * time.Minute,
				Timeout:              20 * time.Second,
			},
			TLS: TLSConfig{
				Enabled:  getEnvAsBoolMS("GRPC_TLS_ENABLED", false),
				CertFile: getEnvMS("GRPC_CERT_FILE", "/etc/grpc/certs/server.crt"),
				KeyFile:  getEnvMS("GRPC_KEY_FILE", "/etc/grpc/certs/server.key"),
			},
		},
		ServiceDiscovery: ServiceDiscoveryConfig{
			Enabled:  getEnvAsBoolMS("SERVICE_DISCOVERY_ENABLED", false),
			Provider: getEnvMS("SERVICE_DISCOVERY_PROVIDER", "consul"),
			Consul: ConsulConfig{
				Address:   getEnvMS("CONSUL_ADDRESS", "localhost"),
				Port:      getEnvAsIntMS("CONSUL_PORT", 8500),
				Datacenter: getEnvMS("CONSUL_DATACENTER", "dc1"),
				Token:     getEnvMS("CONSUL_TOKEN", ""),
				Service: ServiceMetaConfig{
					Name: getEnvMS("SERVICE_NAME", "hjtpx"),
					ID:   getEnvMS("SERVICE_ID", "hjtpx-1"),
					Tags: []string{"captcha", "behavior", "verification"},
					Meta: map[string]string{
						"version":      "17.0",
						"environment": getEnvMS("ENVIRONMENT", "production"),
					},
				},
			},
			Etcd: EtcdConfig{
				Endpoints:   []string{getEnvMS("ETCD_ENDPOINTS", "localhost:2379")},
				DialTimeout: 10 * time.Second,
				Username:   getEnvMS("ETCD_USERNAME", ""),
				Password:   getEnvMS("ETCD_PASSWORD", ""),
			},
		},
		CircuitBreaker: CircuitBreakerConfig{
			Enabled: getEnvAsBoolMS("CIRCUIT_BREAKER_ENABLED", true),
			Services: map[string]CircuitBreakerServiceConfig{
				"captcha": {
					MaxRequests:       5,
					Interval:          10,
					Timeout:           30,
					FailureThreshold:  0.5,
					SuccessThreshold:  2,
					HalfOpenMaxReqs:   3,
				},
				"behavior": {
					MaxRequests:       3,
					Interval:          10,
					Timeout:           60,
					FailureThreshold:  0.6,
					SuccessThreshold:  2,
					HalfOpenMaxReqs:   2,
				},
				"analytics": {
					MaxRequests:       5,
					Interval:          10,
					Timeout:           120,
					FailureThreshold:  0.7,
					SuccessThreshold:  2,
					HalfOpenMaxReqs:   3,
				},
			},
			Default: CircuitBreakerDefaultConfig{
				MaxRequests:       5,
				Interval:          10,
				Timeout:           30,
				FailureThreshold:  0.5,
				SuccessThreshold:  2,
				HalfOpenMaxReqs:   3,
			},
		},
		GlobalAcceleration: GlobalAccelerationConfig{
			Enabled: getEnvAsBoolMS("GLOBAL_ACCELERATION_ENABLED", false),
			CDN: CDNConfig{
				Enabled:  getEnvAsBoolMS("CDN_ENABLED", false),
				Provider: getEnvMS("CDN_PROVIDER", "cloudflare"),
				Endpoints: []CDNEndpoint{
					{Name: "us-east", URL: "https://cdn-us-east.hjtpx.com", Region: "us-east-1", Priority: 1},
					{Name: "eu-west", URL: "https://cdn-eu-west.hjtpx.com", Region: "eu-west-1", Priority: 1},
					{Name: "ap-east", URL: "https://cdn-ap-east.hjtpx.com", Region: "ap-east-1", Priority: 1},
				},
				CachePolicy: CachePolicyConfig{
					DefaultTTL:     getEnvAsIntMS("CDN_DEFAULT_TTL", 3600),
					MaxTTL:         getEnvAsIntMS("CDN_MAX_TTL", 86400),
					MinTTL:         getEnvAsIntMS("CDN_MIN_TTL", 60),
					StaleTTL:       getEnvAsIntMS("CDN_STALE_TTL", 300),
					CacheTypes:     []string{"image", "js", "css", "font"},
					NoCacheHeaders: []string{"Authorization", "X-Internal-Token"},
				},
				SSLEnabled: true,
			},
			MultiRegion: MultiRegionConfig{
				Enabled: getEnvAsBoolMS("MULTI_REGION_ENABLED", false),
				Regions: []RegionConfig{
					{Name: "North America", ID: "na-east", Endpoint: "https://na-east.hjtpx.com", Priority: 1, Weight: 100, GeoTargeting: true},
					{Name: "Europe", ID: "eu-west", Endpoint: "https://eu-west.hjtpx.com", Priority: 1, Weight: 100, GeoTargeting: true},
					{Name: "Asia Pacific", ID: "ap-east", Endpoint: "https://ap-east.hjtpx.com", Priority: 1, Weight: 100, GeoTargeting: true},
				},
				Replication: ReplicationConfig{
					Mode:         getEnvMS("REPLICATION_MODE", "sync"),
					AsyncEnabled: getEnvAsBoolMS("ASYNC_REPLICATION_ENABLED", false),
					SyncInterval: getEnvAsIntMS("SYNC_INTERVAL_SECS", 60),
					ConflictRes:  getEnvMS("CONFLICT_RESOLUTION", "last_write_wins"),
				},
			},
			SmartRouting: SmartRoutingConfig{
				Enabled:         getEnvAsBoolMS("SMART_ROUTING_ENABLED", true),
				Strategy:        getEnvMS("ROUTING_STRATEGY", "latency"),
				LatencyThreshold: getEnvAsIntMS("LATENCY_THRESHOLD_MS", 100),
				HealthCheck: SmartHealthCheck{
					Enabled:  true,
					Interval: 5 * time.Second,
					Timeout:  2 * time.Second,
					Retries:  3,
				},
			},
			EdgeComputing: EdgeComputingConfig{
				Enabled: getEnvAsBoolMS("EDGE_COMPUTING_ENABLED", false),
				Nodes: []EdgeNodeConfig{
					{ID: "edge-us-east", Name: "US East Edge", Region: "us-east-1", Endpoint: "https://edge-us-east.hjtpx.com", Capacity: 1000},
					{ID: "edge-eu-west", Name: "EU West Edge", Region: "eu-west-1", Endpoint: "https://edge-eu-west.hjtpx.com", Capacity: 1000},
					{ID: "edge-ap-east", Name: "AP East Edge", Region: "ap-east-1", Endpoint: "https://edge-ap-east.hjtpx.com", Capacity: 1000},
				},
				Functions: []EdgeFunctionConfig{
					{Name: "captcha-validate", Type: "validation", Runtime: "wasm"},
					{Name: "fingerprint-process", Type: "processing", Runtime: "wasm"},
					{Name: "rate-limit-check", Type: "security", Runtime: "wasm"},
				},
			},
		},
		Monitoring: MicroserviceMonitoringConfig{
			Enabled: getEnvAsBoolMS("MONITORING_ENABLED", true),
			OpenTelemetry: OpenTelemetryConfig{
				Enabled:     getEnvAsBoolMS("OTEL_ENABLED", true),
				Endpoint:    getEnvMS("OTEL_ENDPOINT", "localhost:4317"),
				Insecure:    getEnvAsBoolMS("OTEL_INSECURE", true),
				ServiceName: getEnvMS("OTEL_SERVICE_NAME", "hjtpx"),
				ExportType:  getEnvMS("OTEL_EXPORT_TYPE", "grpc"),
				Headers:     map[string]string{},
				SamplingRate: getEnvAsFloatMS("OTEL_SAMPLING_RATE", 1.0),
			},
			Profiling: ProfilingConfig{
				Enabled:   getEnvAsBoolMS("PROFILING_ENABLED", false),
				Endpoints: []string{getEnvMS("PYROSCOPE_ENDPOINT", "http://localhost:4040")},
				Interval:  getEnvAsIntMS("PROFILING_INTERVAL", 10),
				Types:     []string{"cpu", "heap", "goroutine", "mutex"},
			},
			LogAggregation: LogAggregationConfig{
				Enabled:   getEnvAsBoolMS("LOG_AGGREGATION_ENABLED", true),
				Provider:  getEnvMS("LOG_PROVIDER", "loki"),
				Endpoints: []string{getEnvMS("LOKI_ENDPOINT", "http://localhost:3100")},
				Aggregation: AggregationPolicy{
					ByService:    true,
					BySeverity:   true,
					ByTimeWindow: getEnvAsIntMS("LOG_TIME_WINDOW_SECS", 60),
					MaxBatchSize: getEnvAsIntMS("LOG_MAX_BATCH_SIZE", 1000),
					FlushInterval: getEnvAsIntMS("LOG_FLUSH_INTERVAL", 5),
				},
			},
			AlertAggregation: AlertAggregationConfig{
				Enabled:  getEnvAsBoolMS("ALERT_AGGREGATION_ENABLED", true),
				Provider: getEnvMS("ALERT_PROVIDER", "alertmanager"),
				GroupBy:  []string{"service", "severity", "alertname"},
				TimeWindow: getEnvAsIntMS("ALERT_TIME_WINDOW_SECS", 300),
				Threshold: getEnvAsIntMS("ALERT_THRESHOLD", 10),
				Deduplication: DeduplicationConfig{
					Enabled:    true,
					WindowSecs: getEnvAsIntMS("ALERT_DEDUP_WINDOW_SECS", 300),
					MaxCount:   getEnvAsIntMS("ALERT_DEDUP_MAX_COUNT", 100),
				},
			},
		},
	}

	return cfg
}

func getEnvAsIntMS(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		var result int
		_, err := fmt.Sscanf(value, "%d", &result)
		if err == nil {
			return result
		}
	}
	return defaultValue
}

func getEnvAsBoolMS(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		return value == "true" || value == "1"
	}
	return defaultValue
}

func getEnvAsFloatMS(key string, defaultValue float64) float64 {
	if value, exists := os.LookupEnv(key); exists {
		var result float64
		_, err := fmt.Sscanf(value, "%f", &result)
		if err == nil {
			return result
		}
	}
	return defaultValue
}

func getEnvMS(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

var microserviceConfig *MicroserviceConfig

func GetMicroserviceConfig() *MicroserviceConfig {
	if microserviceConfig == nil {
		microserviceConfig = LoadMicroserviceConfig()
	}
	return microserviceConfig
}
