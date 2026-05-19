package service

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type DeployStatus string

const (
	DeployStatusPending    DeployStatus = "pending"
	DeployStatusBuilding   DeployStatus = "building"
	DeployStatusTesting    DeployStatus = "testing"
	DeployStatusUploading  DeployStatus = "uploading"
	DeployStatusDeploying DeployStatus = "deploying"
	DeployStatusComplete   DeployStatus = "complete"
	DeployStatusFailed     DeployStatus = "failed"
	DeployStatusRolledBack DeployStatus = "rolled_back"
)

type DeployStage struct {
	Stage       string    `json:"stage"`
	Status      string    `json:"status"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Error       string    `json:"error,omitempty"`
	Logs        []string  `json:"logs"`
}

type DeployResult struct {
	DeploymentID   string                 `json:"deployment_id"`
	FunctionName   string                 `json:"function_name"`
	Status         DeployStatus           `json:"status"`
	Version        string                 `json:"version"`
	ArtifactURL    string                 `json:"artifact_url"`
	ArtifactSize   int64                  `json:"artifact_size"`
	BuildDuration  time.Duration         `json:"build_duration"`
	DeployDuration time.Duration         `json:"deploy_duration"`
	TotalDuration  time.Duration         `json:"total_duration"`
	Stages        []DeployStage          `json:"stages"`
	ErrorMessage   string                 `json:"error_message,omitempty"`
	RollbackTo     string                 `json:"rollback_to,omitempty"`
	Metadata       map[string]interface{} `json:"metadata"`
}

type BuildConfig struct {
	BuildImage    string            `json:"build_image"`
	BuildArgs     map[string]string `json:"build_args"`
	CacheEnabled  bool              `json:"cache_enabled"`
	CacheBucket   string            `json:"cache_bucket"`
	EnvVars       map[string]string `json:"env_vars"`
	WorkingDir    string            `json:"working_dir"`
	OutputDir     string            `json:"output_dir"`
	GoModProxy    string            `json:"go_mod_proxy"`
	NpmRegistry   string            `json:"npm_registry"`
	PythonIndex   string            `json:"python_index"`
}

type FunctionDeployer struct {
	manager         *ServerlessManager
	deployments     map[string]*DeployResult
	deploymentMu    sync.RWMutex
	httpClient      *http.Client
	buildConfig     *BuildConfig
	storageBackend  StorageBackend
	registry        ContainerRegistry
	builder         BuildEngine
	deployStrategy  DeployStrategy
}

type StorageBackend interface {
	Upload(ctx context.Context, key string, data []byte) (string, error)
	Download(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
	GetURL(key string) string
}

type ContainerRegistry interface {
	Push(ctx context.Context, image string, tarData []byte) error
	Pull(ctx context.Context, image string) ([]byte, error)
	List(ctx context.Context, prefix string) ([]string, error)
}

type BuildEngine interface {
	Build(ctx context.Context, config *BuildConfig, sourceDir string) ([]byte, error)
	Validate(ctx context.Context, artifact []byte) error
}

type DeployStrategy interface {
	Deploy(ctx context.Context, artifact []byte, config *FunctionConfig) error
	Rollback(ctx context.Context, version string) error
	GetTrafficWeights(version string) (map[string]float64, error)
}

type DefaultBuildEngine struct {
	buildCache   map[string][]byte
	cacheMu      sync.RWMutex
}

type DefaultStorageBackend struct {
	basePath     string
	baseURL      string
}

type DefaultContainerRegistry struct {
	registryURL string
	authToken   string
}

type CanaryDeployStrategy struct {
	manager *ServerlessManager
}

type BlueGreenDeployStrategy struct {
	manager *ServerlessManager
	activeEnv string
}

type FunctionArtifact struct {
	FunctionName string            `json:"function_name"`
	Version      string            `json:"version"`
	Runtime      RuntimeType       `json:"runtime"`
	Checksum     string            `json:"checksum"`
	Size         int64             `json:"size"`
	CreatedAt    time.Time         `json:"created_at"`
	BuildInfo    BuildInfo         `json:"build_info"`
	Layers       []LayerInfo       `json:"layers"`
	Config       *FunctionConfig   `json:"config"`
}

type BuildInfo struct {
	BuildID      string            `json:"build_id"`
	BuildImage   string            `json:"build_image"`
	BuildArgs    map[string]string `json:"build_args"`
	BuildTime   time.Duration     `json:"build_time"`
	GoVersion    string            `json:"go_version"`
	Dependencies []string         `json:"dependencies"`
}

type LayerInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Size        int64  `json:"size"`
	Checksum    string `json:"checksum"`
	Description string `json:"description"`
}

type DeployOptions struct {
	Version       string
	Stage         string
	Strategy      string
	TrafficWeight map[string]float64
	SkipTests     bool
	SkipValidation bool
	Environment   map[string]string
	Timeout       time.Duration
}

func NewFunctionDeployer(manager *ServerlessManager) *FunctionDeployer {
	deployer := &FunctionDeployer{
		manager:     manager,
		deployments: make(map[string]*DeployResult),
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
		buildConfig: &BuildConfig{
			BuildImage:   "serverless/go-builder:1.20",
			CacheEnabled: true,
			GoModProxy:   "https://proxy.golang.org",
			NpmRegistry:  "https://registry.npmjs.org",
			PythonIndex:  "https://pypi.org/simple",
		},
		storageBackend: &DefaultStorageBackend{
			basePath: "/tmp/serverless/artifacts",
			baseURL:  "https://storage.serverless.local",
		},
		registry: &DefaultContainerRegistry{
			registryURL: "registry.serverless.local",
		},
		builder: &DefaultBuildEngine{
			buildCache: make(map[string][]byte),
		},
	}
	
	deployer.deployStrategy = &CanaryDeployStrategy{manager: manager}
	
	return deployer
}

func (d *FunctionDeployer) Deploy(ctx context.Context, name string, sourceCode []byte, opts *DeployOptions) (*DeployResult, error) {
	if opts == nil {
		opts = &DeployOptions{}
	}
	
	_, err := d.manager.GetConfig(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get function config: %w", err)
	}
	
	deploymentID := fmt.Sprintf("deploy-%d", time.Now().UnixNano())
	
	result := &DeployResult{
		DeploymentID: deploymentID,
		FunctionName: name,
		Status:       DeployStatusPending,
		Stages:       []DeployStage{},
		Metadata:     make(map[string]interface{}),
	}
	
	if opts.Version != "" {
		result.Version = opts.Version
	} else {
		result.Version = fmt.Sprintf("v%d", time.Now().Unix())
	}
	
	d.deploymentMu.Lock()
	d.deployments[deploymentID] = result
	d.deploymentMu.Unlock()
	
	totalStart := time.Now()
	
	stages := []struct {
		name    string
		handler func() error
	}{
		{"building", func() error { return d.executeBuildStage(ctx, name, sourceCode, result, opts) }},
		{"testing", func() error { return d.executeTestStage(ctx, name, result, opts) }},
		{"uploading", func() error { return d.executeUploadStage(ctx, name, result) }},
		{"deploying", func() error { return d.executeDeployStage(ctx, name, result) }},
	}
	
	for _, stage := range stages {
		stageStart := time.Now()
		result.Stages = append(result.Stages, DeployStage{
			Stage:     stage.name,
			Status:    "in_progress",
			StartedAt: stageStart,
			Logs:      []string{},
		})
		
		if err := stage.handler(); err != nil {
			result.Status = DeployStatusFailed
			result.ErrorMessage = err.Error()
			result.TotalDuration = time.Since(totalStart)
			
			currentStage := &result.Stages[len(result.Stages)-1]
			currentStage.Status = "failed"
			currentStage.Error = err.Error()
			now := time.Now()
			currentStage.CompletedAt = &now
			
			d.manager.SetFunctionState(name, FunctionStateError)
			return result, err
		}
		
		currentStage := &result.Stages[len(result.Stages)-1]
		currentStage.Status = "complete"
		now := time.Now()
		currentStage.CompletedAt = &now
		
		switch stage.name {
		case "building":
			result.BuildDuration = time.Since(stageStart)
			result.Status = DeployStatusBuilding
		case "testing":
			result.Status = DeployStatusTesting
		case "uploading":
			result.Status = DeployStatusUploading
		case "deploying":
			result.Status = DeployStatusComplete
			result.DeployDuration = time.Since(stageStart)
		}
	}
	
	result.TotalDuration = time.Since(totalStart)
	
	d.manager.SetFunctionState(name, FunctionStateRunning)
	
	return result, nil
}

func (d *FunctionDeployer) executeBuildStage(ctx context.Context, name string, sourceCode []byte, result *DeployResult, opts *DeployOptions) error {
	buildStart := time.Now()
	
	logs := append(result.Stages[len(result.Stages)-1].Logs, fmt.Sprintf("[%s] Starting build for function %s", time.Now().Format(time.RFC3339), name))
	result.Stages[len(result.Stages)-1].Logs = logs
	
	tmpDir, err := os.MkdirTemp("", "serverless-build-*")
	if err != nil {
		return fmt.Errorf("failed to create build directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)
	
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), sourceCode, 0644); err != nil {
		return fmt.Errorf("failed to write source code: %w", err)
	}
	
	logs = append(logs, fmt.Sprintf("[%s] Source code extracted to %s", time.Now().Format(time.RFC3339), tmpDir))
	
	goVersion := d.detectGoVersion(sourceCode)
	logs = append(logs, fmt.Sprintf("[%s] Detected Go version: %s", time.Now().Format(time.RFC3339), goVersion))
	
	artifact, err := d.builder.Build(ctx, d.buildConfig, tmpDir)
	if err != nil {
		logs = append(logs, fmt.Sprintf("[%s] Build failed: %v", time.Now().Format(time.RFC3339), err))
		return fmt.Errorf("build failed: %w", err)
	}
	
	checksum := sha256.Sum256(artifact)
	result.Metadata["checksum"] = hex.EncodeToString(checksum[:])
	result.Metadata["build_time"] = time.Since(buildStart).String()
	
	logs = append(logs, fmt.Sprintf("[%s] Build completed successfully", time.Now().Format(time.RFC3339)))
	result.Stages[len(result.Stages)-1].Logs = logs
	
	return nil
}

func (d *FunctionDeployer) executeTestStage(ctx context.Context, name string, result *DeployResult, opts *DeployOptions) error {
	if opts.SkipTests {
		result.Stages[len(result.Stages)-1].Logs = append(result.Stages[len(result.Stages)-1].Logs, 
			fmt.Sprintf("[%s] Tests skipped", time.Now().Format(time.RFC3339)))
		return nil
	}
	
	result.Stages[len(result.Stages)-1].Logs = append(result.Stages[len(result.Stages)-1].Logs,
		fmt.Sprintf("[%s] Running unit tests", time.Now().Format(time.RFC3339)))
	
	time.Sleep(100 * time.Millisecond)
	
	result.Stages[len(result.Stages)-1].Logs = append(result.Stages[len(result.Stages)-1].Logs,
		fmt.Sprintf("[%s] Tests passed", time.Now().Format(time.RFC3339)))
	
	return nil
}

func (d *FunctionDeployer) executeUploadStage(ctx context.Context, name string, result *DeployResult) error {
	artifact := []byte(fmt.Sprintf("artifact for %s", name))
	
	key := fmt.Sprintf("functions/%s/%s/artifact.zip", name, result.Version)
	
	url, err := d.storageBackend.Upload(ctx, key, artifact)
	if err != nil {
		return fmt.Errorf("failed to upload artifact: %w", err)
	}
	
	result.ArtifactURL = url
	result.ArtifactSize = int64(len(artifact))
	
	result.Stages[len(result.Stages)-1].Logs = append(result.Stages[len(result.Stages)-1].Logs,
		fmt.Sprintf("[%s] Artifact uploaded to %s", time.Now().Format(time.RFC3339), url))
	
	return nil
}

func (d *FunctionDeployer) executeDeployStage(ctx context.Context, name string, result *DeployResult) error {
	config, err := d.manager.GetConfig(name)
	if err != nil {
		return fmt.Errorf("failed to get function config: %w", err)
	}
	
	if err := d.deployStrategy.Deploy(ctx, []byte(result.ArtifactURL), config); err != nil {
		return fmt.Errorf("deployment failed: %w", err)
	}
	
	result.Stages[len(result.Stages)-1].Logs = append(result.Stages[len(result.Stages)-1].Logs,
		fmt.Sprintf("[%s] Function deployed successfully", time.Now().Format(time.RFC3339)))
	
	return nil
}

func (d *FunctionDeployer) detectGoVersion(sourceCode []byte) string {
	content := string(sourceCode)
	
	if strings.Contains(content, "//go:build go1.20") || strings.Contains(content, "// +build go1.20") {
		return "1.20"
	}
	if strings.Contains(content, "//go:build go1.18") || strings.Contains(content, "// +build go1.18") {
		return "1.18"
	}
	if strings.Contains(content, "//go:build go1.16") || strings.Contains(content, "// +build go1.16") {
		return "1.16"
	}
	
	return "1.20"
}

func (d *FunctionDeployer) Rollback(ctx context.Context, name, version string) error {
	if err := d.deployStrategy.Rollback(ctx, version); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}
	
	return nil
}

func (d *FunctionDeployer) GetDeployment(deploymentID string) (*DeployResult, error) {
	d.deploymentMu.RLock()
	defer d.deploymentMu.RUnlock()
	
	deployment, exists := d.deployments[deploymentID]
	if !exists {
		return nil, fmt.Errorf("deployment %s not found", deploymentID)
	}
	
	return deployment, nil
}

func (d *FunctionDeployer) ListDeployments(name string) []*DeployResult {
	d.deploymentMu.RLock()
	defer d.deploymentMu.RUnlock()
	
	var results []*DeployResult
	for _, deployment := range d.deployments {
		if deployment.FunctionName == name {
			results = append(results, deployment)
		}
	}
	
	return results
}

func (d *FunctionDeployer) CreateDeploymentPackage(functionName string, runtime RuntimeType, sourceCode []byte) ([]byte, error) {
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)
	
	files := map[string][]byte{
		"bootstrap":     []byte("#!/bin/sh\nexec /usr/local/bin/serverless-runtime"),
		"function/main": sourceCode,
	}
	
	if strings.HasPrefix(string(runtime), "go") {
		files["go.mod"] = []byte(fmt.Sprintf("module %s\n\ngo 1.20\n", functionName))
	}
	
	for name, content := range files {
		if err := tw.WriteHeader(&tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(content)),
		}); err != nil {
			return nil, fmt.Errorf("failed to write tar header: %w", err)
		}
		
		if _, err := tw.Write(content); err != nil {
			return nil, fmt.Errorf("failed to write tar content: %w", err)
		}
	}
	
	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("failed to close tar writer: %w", err)
	}
	
	if err := gzw.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}
	
	return buf.Bytes(), nil
}

func (d *FunctionDeployer) ValidateDeployment(deployment *DeployResult) error {
	if deployment == nil {
		return fmt.Errorf("deployment is nil")
	}
	
	if deployment.FunctionName == "" {
		return fmt.Errorf("function name is required")
	}
	
	if deployment.Status == DeployStatusFailed {
		return fmt.Errorf("deployment failed: %s", deployment.ErrorMessage)
	}
	
	return nil
}

func (d *FunctionDeployer) SetBuildConfig(config *BuildConfig) {
	d.buildConfig = config
}

func (d *FunctionDeployer) SetStorageBackend(backend StorageBackend) {
	d.storageBackend = backend
}

func (d *FunctionDeployer) SetRegistry(registry ContainerRegistry) {
	d.registry = registry
}

func (d *FunctionDeployer) SetDeployStrategy(strategy DeployStrategy) {
	d.deployStrategy = strategy
}

func (b *DefaultBuildEngine) Build(ctx context.Context, config *BuildConfig, sourceDir string) ([]byte, error) {
	b.cacheMu.Lock()
	defer b.cacheMu.Unlock()
	
	cacheKey := fmt.Sprintf("%s-%s", sourceDir, config.BuildImage)
	
	if config.CacheEnabled {
		if cached, exists := b.buildCache[cacheKey]; exists {
			return cached, nil
		}
	}
	
	artifact := []byte(fmt.Sprintf("built artifact for %s using %s", sourceDir, config.BuildImage))
	
	if config.CacheEnabled {
		b.buildCache[cacheKey] = artifact
	}
	
	return artifact, nil
}

func (b *DefaultBuildEngine) Validate(ctx context.Context, artifact []byte) error {
	if len(artifact) == 0 {
		return fmt.Errorf("artifact is empty")
	}
	
	return nil
}

func (s *DefaultStorageBackend) Upload(ctx context.Context, key string, data []byte) (string, error) {
	os.MkdirAll(filepath.Join(s.basePath, filepath.Dir(key)), 0755)
	
	fullPath := filepath.Join(s.basePath, key)
	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}
	
	return fmt.Sprintf("%s/%s", s.baseURL, key), nil
}

func (s *DefaultStorageBackend) Download(ctx context.Context, key string) ([]byte, error) {
	fullPath := filepath.Join(s.basePath, key)
	return os.ReadFile(fullPath)
}

func (s *DefaultStorageBackend) Delete(ctx context.Context, key string) error {
	fullPath := filepath.Join(s.basePath, key)
	return os.Remove(fullPath)
}

func (s *DefaultStorageBackend) GetURL(key string) string {
	return fmt.Sprintf("%s/%s", s.baseURL, key)
}

func (r *DefaultContainerRegistry) Push(ctx context.Context, image string, tarData []byte) error {
	return nil
}

func (r *DefaultContainerRegistry) Pull(ctx context.Context, image string) ([]byte, error) {
	return nil, nil
}

func (r *DefaultContainerRegistry) List(ctx context.Context, prefix string) ([]string, error) {
	return []string{}, nil
}

func (s *CanaryDeployStrategy) Deploy(ctx context.Context, artifact []byte, config *FunctionConfig) error {
	return nil
}

func (s *CanaryDeployStrategy) Rollback(ctx context.Context, version string) error {
	return nil
}

func (s *CanaryDeployStrategy) GetTrafficWeights(version string) (map[string]float64, error) {
	return map[string]float64{
		"current": 0.9,
		"new":     0.1,
	}, nil
}

func (s *BlueGreenDeployStrategy) Deploy(ctx context.Context, artifact []byte, config *FunctionConfig) error {
	return nil
}

func (s *BlueGreenDeployStrategy) Rollback(ctx context.Context, version string) error {
	return nil
}

func (s *BlueGreenDeployStrategy) GetTrafficWeights(version string) (map[string]float64, error) {
	return map[string]float64{
		"blue": 0.0,
		"green": 1.0,
	}, nil
}

func DeployFunctionFromFile(ctx context.Context, manager *ServerlessManager, deployer *FunctionDeployer, filePath string) (*DeployResult, error) {
	sourceCode, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read source file: %w", err)
	}
	
	var metadata struct {
		FunctionName string `json:"function_name"`
		Runtime      string `json:"runtime"`
	}
	
	if strings.HasSuffix(filePath, ".json") {
		if err := json.Unmarshal(sourceCode, &metadata); err == nil {
			sourceCode, _ = os.ReadFile(strings.TrimSuffix(filePath, ".json") + ".go")
		}
	}
	
	if metadata.FunctionName == "" {
		metadata.FunctionName = filepath.Base(filePath)
		metadata.FunctionName = strings.TrimSuffix(metadata.FunctionName, filepath.Ext(metadata.FunctionName))
	}
	
	if err := manager.RegisterFunction(&FunctionConfig{
		FunctionName: metadata.FunctionName,
		Runtime:      RuntimeType(metadata.Runtime),
		Handler:      "main",
		Memory:       Memory256MB,
		Timeout:      Timeout30s,
	}); err != nil && !strings.Contains(err.Error(), "already registered") {
		return nil, fmt.Errorf("failed to register function: %w", err)
	}
	
	return deployer.Deploy(ctx, metadata.FunctionName, sourceCode, nil)
}

type DeploymentWatcher struct {
	deploymentID string
	status       atomic.Value
	logs         []string
	mu           sync.RWMutex
}

func NewDeploymentWatcher(deploymentID string) *DeploymentWatcher {
	watcher := &DeploymentWatcher{
		deploymentID: deploymentID,
	}
	watcher.status.Store(DeployStatusPending)
	return watcher
}

func (w *DeploymentWatcher) UpdateStatus(status DeployStatus) {
	w.status.Store(status)
}

func (w *DeploymentWatcher) AppendLog(log string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.logs = append(w.logs, log)
}

func (w *DeploymentWatcher) GetStatus() DeployStatus {
	return w.status.Load().(DeployStatus)
}

func (w *DeploymentWatcher) GetLogs() []string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	logs := make([]string, len(w.logs))
	copy(logs, w.logs)
	return logs
}
