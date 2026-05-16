package config

type HAConfig struct {
	Enabled              bool          `yaml:"enabled" json:"enabled"`
	NodeID              string        `yaml:"node_id" json:"node_id"`
	ClusterName         string        `yaml:"cluster_name" json:"cluster_name"`
	Role                string        `yaml:"role" json:"role"`
	BindAddress         string        `yaml:"bind_address" json:"bind_address"`
	AdvertiseAddress    string        `yaml:"advertise_address" json:"advertise_address"`
	Port                int           `yaml:"port" json:"port"`
	GossipPort          int           `yaml:"gossip_port" json:"gossip_port"`
	InitialNodes        []string      `yaml:"initial_nodes" json:"initial_nodes"`
	HealthCheck         HealthCheckConfig `yaml:"health_check" json:"health_check"`
	Failover            FailoverConfig   `yaml:"failover" json:"failover"`
	Sync                SyncConfig       `yaml:"sync" json:"sync"`
	Backup             BackupConfig      `yaml:"backup" json:"backup"`
	LoadBalancer        LBConfig         `yaml:"load_balancer" json:"load_balancer"`
}

type HealthCheckConfig struct {
	Enabled        bool          `yaml:"enabled" json:"enabled"`
	Interval       string        `yaml:"interval" json:"interval"`
	Timeout        string        `yaml:"timeout" json:"timeout"`
	Endpoint       string        `yaml:"endpoint" json:"endpoint"`
	MaxRetries     int           `yaml:"max_retries" json:"max_retries"`
	SuccessThreshold int         `yaml:"success_threshold" json:"success_threshold"`
}

type FailoverConfig struct {
	Enabled              bool   `yaml:"enabled" json:"enabled"`
	FailureThreshold     int    `yaml:"failure_threshold" json:"failure_threshold"`
	RecoveryThreshold    int    `yaml:"recovery_threshold" json:"recovery_threshold"`
	FailoverTimeout      string `yaml:"failover_timeout" json:"failover_timeout"`
	RecoveryTimeout       string `yaml:"recovery_timeout" json:"recovery_timeout"`
	MaxFailoverAttempts  int    `yaml:"max_failover_attempts" json:"max_failover_attempts"`
	Strategy             string `yaml:"strategy" json:"strategy"`
	EnableAutoRecovery   bool   `yaml:"enable_auto_recovery" json:"enable_auto_recovery"`
}

type SyncConfig struct {
	Enabled           bool   `yaml:"enabled" json:"enabled"`
	Strategy          string `yaml:"strategy" json:"strategy"`
	Interval          string `yaml:"interval" json:"interval"`
	Timeout           string `yaml:"timeout" json:"timeout"`
	RetryAttempts     int    `yaml:"retry_attempts" json:"retry_attempts"`
	BatchSize         int    `yaml:"batch_size" json:"batch_size"`
	EnableCompression bool   `yaml:"enable_compression" json:"enable_compression"`
}

type BackupConfig struct {
	Enabled              bool   `yaml:"enabled" json:"enabled"`
	Interval             string `yaml:"interval" json:"interval"`
	RetentionDays        int    `yaml:"retention_days" json:"retention_days"`
	MaxBackups           int    `yaml:"max_backups" json:"max_backups"`
	BackupPath           string `yaml:"backup_path" json:"backup_path"`
	EnableCompression    bool   `yaml:"enable_compression" json:"enable_compression"`
	EnableEncryption     bool   `yaml:"enable_encryption" json:"enable_encryption"`
	HealthCheckBeforeBackup bool `yaml:"health_check_before_backup" json:"health_check_before_backup"`
	TargetLocations      []string `yaml:"target_locations" json:"target_locations"`
}

type LBConfig struct {
	Strategy           string   `yaml:"strategy" json:"strategy"`
	HealthCheckEnabled bool     `yaml:"health_check_enabled" json:"health_check_enabled"`
	HealthCheckInterval string   `yaml:"health_check_interval" json:"health_check_interval"`
	MaxRetries         int      `yaml:"max_retries" json:"max_retries"`
	Timeout            string   `yaml:"timeout" json:"timeout"`
	Backends           []string `yaml:"backends" json:"backends"`
}

func DefaultHAConfig() *HAConfig {
	return &HAConfig{
		Enabled:           true,
		ClusterName:       "hjtpx-cluster",
		Role:              "standalone",
		BindAddress:       "0.0.0.0",
		Port:              7946,
		GossipPort:        7946,
		InitialNodes:      []string{},
		HealthCheck: HealthCheckConfig{
			Enabled:         true,
			Interval:        "10s",
			Timeout:         "5s",
			Endpoint:        "/health",
			MaxRetries:      3,
			SuccessThreshold: 2,
		},
		Failover: FailoverConfig{
			Enabled:             true,
			FailureThreshold:    3,
			RecoveryThreshold:   2,
			FailoverTimeout:     "30s",
			RecoveryTimeout:     "60s",
			MaxFailoverAttempts: 3,
			Strategy:            "automatic",
			EnableAutoRecovery:  true,
		},
		Sync: SyncConfig{
			Enabled:           true,
			Strategy:          "eventual",
			Interval:          "5s",
			Timeout:           "30s",
			RetryAttempts:     3,
			BatchSize:         100,
			EnableCompression: true,
		},
		Backup: BackupConfig{
			Enabled:               true,
			Interval:              "1h",
			RetentionDays:         7,
			MaxBackups:           10,
			BackupPath:            "/var/backups/hjtpx",
			EnableCompression:    true,
			EnableEncryption:     false,
			HealthCheckBeforeBackup: true,
			TargetLocations:      []string{},
		},
		LoadBalancer: LBConfig{
			Strategy:            "round_robin",
			HealthCheckEnabled:  true,
			HealthCheckInterval: "10s",
			MaxRetries:          3,
			Timeout:             "30s",
			Backends:            []string{},
		},
	}
}
