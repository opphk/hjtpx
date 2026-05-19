package model

import "time"

type ObfuscationTask struct {
	ID              string    `json:"id" gorm:"primaryKey"`
	AppID           string    `json:"app_id" gorm:"index"`
	OriginalCode    string    `json:"original_code" gorm:"type:text"`
	ObfuscatedCode  string    `json:"obfuscated_code" gorm:"type:text"`
	Config          string    `json:"config" gorm:"type:text"`
	Status          string    `json:"status" gorm:"default:'pending'"`
	ErrorMessage    string    `json:"error_message,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	ProcessingTimeMs int64     `json:"processing_time_ms"`
	CodeLength      int       `json:"code_length"`
	ObfuscatedLength int      `json:"obfuscated_length"`
	CompressionRatio float64  `json:"compression_ratio"`
}

type ObfuscationConfig struct {
	EnableVariableObfuscation    bool     `json:"enable_variable_obfuscation"`
	EnableStringEncryption       bool     `json:"enable_string_encryption"`
	EnableCodeCompression        bool     `json:"enable_code_compression"`
	EnableControlFlowFlattening  bool     `json:"enable_control_flow_flattening"`
	EnableDeadCodeInjection      bool     `json:"enable_dead_code_injection"`
	EnableFunctionWrapping       bool     `json:"enable_function_wrapping"`
	StringEncryptionMethod      string   `json:"string_encryption_method"`
	EncryptionRounds            int      `json:"encryption_rounds"`
	SplitFragments              int      `json:"split_fragments"`
	DeadCodeRatio               float64  `json:"dead_code_ratio"`
}

type ObfuscationStats struct {
	TotalTasks          int64   `json:"total_tasks"`
	SuccessCount        int64   `json:"success_count"`
	FailureCount        int64   `json:"failure_count"`
	AverageCodeLength   float64 `json:"average_code_length"`
	AverageObfTimeMs    float64 `json:"average_obfuscation_time_ms"`
	AverageCompression  float64 `json:"average_compression_ratio"`
}

type ObfuscationReport struct {
	TaskID               string    `json:"task_id"`
	OriginalHash         string    `json:"original_hash"`
	ObfuscatedHash       string    `json:"obfuscated_hash"`
	Techniques           []string  `json:"techniques"`
	StrengthScore        float64   `json:"strength_score"`
	EstimatedTime        string    `json:"estimated_time"`
	Recommendations      []string  `json:"recommendations"`
	GeneratedAt          time.Time `json:"generated_at"`
}

type StringEncryptionInfo struct {
	Method          string   `json:"method"`
	Rounds          int      `json:"rounds"`
	EncryptedCount  int      `json:"encrypted_count"`
	TotalCount      int      `json:"total_count"`
	EncryptionRatio float64  `json:"encryption_ratio"`
}

type ControlFlowInfo struct {
	FlattenedFunctions int     `json:"flattened_functions"`
	TotalFunctions     int     `json:"total_functions"`
	FlatteningRatio    float64 `json:"flattening_ratio"`
	StateMachineCount  int     `json:"state_machine_count"`
}

type CodeSplitInfo struct {
	FragmentsCreated int      `json:"fragments_created"`
	AverageFragmentSize float64 `json:"average_fragment_size"`
	SplittingMethod   string   `json:"splitting_method"`
}

type DeadCodeInfo struct {
	LinesInjected   int     `json:"lines_injected"`
	CodeRatio       float64 `json:"code_ratio"`
	TypesUsed       []string `json:"types_used"`
}

type ObfuscationQualityMetrics struct {
	EntropyScore       float64 `json:"entropy_score"`
	LexicalComplexity  float64 `json:"lexical_complexity"`
	StructuralComplexity float64 `json:"structural_complexity"`
	OverallScore       float64 `json:"overall_score"`
}

type BatchObfuscationRequest struct {
	AppID      string   `json:"app_id" binding:"required"`
	CodeBlocks []string `json:"code_blocks" binding:"required"`
	Config     ObfuscationConfig `json:"config"`
}

type BatchObfuscationResponse struct {
	TaskID    string   `json:"task_id"`
	Results   []ObfuscationResult `json:"results"`
	SuccessCount int `json:"success_count"`
	FailureCount int `json:"failure_count"`
}

type ObfuscationResult struct {
	Index            int    `json:"index"`
	OriginalLength   int    `json:"original_length"`
	ObfuscatedCode   string `json:"obfuscated_code"`
	ObfuscatedLength int    `json:"obfuscated_length"`
	Success          bool   `json:"success"`
	Error            string `json:"error,omitempty"`
}

type ObfuscationHistory struct {
	ID              string    `json:"id"`
	AppID           string    `json:"app_id"`
	Configurations  []string  `json:"configurations"`
	TotalRuns       int       `json:"total_runs"`
	LastRunAt       time.Time `json:"last_run_at"`
	MostUsedConfig  string    `json:"most_used_config"`
}

type ObfuscationPreset struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Level       int      `json:"level"`
	Features    []string `json:"features"`
	Config      ObfuscationConfig `json:"config"`
}

var ObfuscationPresets = []ObfuscationPreset{
	{
		Name:        "basic",
		Description: "基础混淆，仅变量名混淆",
		Level:       1,
		Features:    []string{"variable_obfuscation"},
		Config: ObfuscationConfig{
			EnableVariableObfuscation: true,
			EnableStringEncryption:   false,
			EnableControlFlowFlattening: false,
			EnableDeadCodeInjection:   false,
		},
	},
	{
		Name:        "standard",
		Description: "标准混淆，包含字符串加密和控制流扁平化",
		Level:       2,
		Features:    []string{"variable_obfuscation", "string_encryption", "control_flow_flattening"},
		Config: ObfuscationConfig{
			EnableVariableObfuscation:    true,
			EnableStringEncryption:       true,
			EnableCodeCompression:        true,
			EnableControlFlowFlattening:  true,
			EnableDeadCodeInjection:      false,
			EnableFunctionWrapping:      true,
			StringEncryptionMethod:      "aes-gcm",
			EncryptionRounds:            2,
		},
	},
	{
		Name:        "advanced",
		Description: "高级混淆，包含所有保护技术",
		Level:       3,
		Features:    []string{"variable_obfuscation", "string_encryption", "control_flow_flattening", "dead_code_injection", "code_splitting"},
		Config: ObfuscationConfig{
			EnableVariableObfuscation:    true,
			EnableStringEncryption:       true,
			EnableCodeCompression:        true,
			EnableControlFlowFlattening:  true,
			EnableDeadCodeInjection:      true,
			EnableFunctionWrapping:       true,
			StringEncryptionMethod:       "multi-enc",
			EncryptionRounds:            3,
			SplitFragments:              3,
			DeadCodeRatio:               0.3,
		},
	},
	{
		Name:        "maximum",
		Description: "最大混淆，启用所有保护技术",
		Level:       4,
		Features:    []string{"all"},
		Config: ObfuscationConfig{
			EnableVariableObfuscation:    true,
			EnableStringEncryption:       true,
			EnableCodeCompression:        true,
			EnableControlFlowFlattening:  true,
			EnableDeadCodeInjection:      true,
			EnableFunctionWrapping:      true,
			StringEncryptionMethod:       "aes-gcm",
			EncryptionRounds:            5,
			SplitFragments:              5,
			DeadCodeRatio:               0.5,
		},
	},
}

const (
	ObfuscationStatusPending    = "pending"
	ObfuscationStatusProcessing = "processing"
	ObfuscationStatusCompleted  = "completed"
	ObfuscationStatusFailed     = "failed"
)

const (
	EncryptionMethodAESGCM      = "aes-gcm"
	EncryptionMethodRC4         = "rc4"
	EncryptionMethodChaCha20    = "chacha20"
	EncryptionMethodXOR         = "xor"
	EncryptionMethodMultiRound   = "multi-enc"
	EncryptionMethodCustomTable  = "custom-table"
	EncryptionMethodAESBase64    = "aes-base64"
)
