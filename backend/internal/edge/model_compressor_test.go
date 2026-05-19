package edge

import (
	"context"
	"testing"
)

func TestModelCompressor_NewModelCompressor(t *testing.T) {
	compressor := NewModelCompressor(nil)

	if compressor == nil {
		t.Fatal("Expected compressor to not be nil")
	}

	if len(compressor.compressedModels) != 0 {
		t.Errorf("Expected 0 compressed models initially, got %d", len(compressor.compressedModels))
	}
}

func TestModelCompressor_Quantize(t *testing.T) {
	compressor := NewModelCompressor(nil)
	ctx := context.Background()

	modelData := make([]float32, 1000)
	for i := range modelData {
		modelData[i] = float32(i) / 100.0
	}

	config := &CompressionConfig{
		Method:           MethodQuantization,
		QuantizationType: QuantINT8,
		PreserveAccuracy: true,
	}

	result, err := compressor.Compress(ctx, "test-model", modelData, config)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected compression to succeed")
	}

	if result.CompressionRatio <= 0 {
		t.Error("Expected positive compression ratio")
	}
}

func TestModelCompressor_Prune(t *testing.T) {
	compressor := NewModelCompressor(nil)
	ctx := context.Background()

	modelData := make([]float32, 1000)
	for i := range modelData {
		modelData[i] = float32(i) / 100.0
	}

	config := &CompressionConfig{
		Method:       MethodPruning,
		PruningType:  PruningMagnitude,
		PruningRatio: 0.5,
	}

	result, err := compressor.Compress(ctx, "test-model", modelData, config)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected compression to succeed")
	}
}

func TestModelCompressor_Factorization(t *testing.T) {
	compressor := NewModelCompressor(nil)
	ctx := context.Background()

	modelData := make([]float32, 1000)
	for i := range modelData {
		modelData[i] = float32(i) / 100.0
	}

	config := &CompressionConfig{
		Method: MethodFactorization,
	}

	result, err := compressor.Compress(ctx, "test-model", modelData, config)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected compression to succeed")
	}
}

func TestModelCompressor_KnowledgeDistillation(t *testing.T) {
	compressor := NewModelCompressor(nil)
	ctx := context.Background()

	modelData := make([]float32, 1000)
	for i := range modelData {
		modelData[i] = float32(i) / 100.0
	}

	config := &CompressionConfig{
		Method: MethodKnowledgeDist,
	}

	result, err := compressor.Compress(ctx, "test-model", modelData, config)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected compression to succeed")
	}
}

func TestModelCompressor_GetCompressedModel(t *testing.T) {
	compressor := NewModelCompressor(nil)
	ctx := context.Background()

	modelData := make([]float32, 1000)
	for i := range modelData {
		modelData[i] = float32(i) / 100.0
	}

	config := &CompressionConfig{
		Method:           MethodQuantization,
		QuantizationType: QuantFP16,
	}

	result, err := compressor.Compress(ctx, "test-model", modelData, config)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	compressed, err := compressor.GetCompressedModel(result.CompressedModel.ID)
	if err != nil {
		t.Fatalf("GetCompressedModel failed: %v", err)
	}

	if compressed.OriginalID != "test-model" {
		t.Errorf("Expected original ID 'test-model', got '%s'", compressed.OriginalID)
	}
}

func TestModelCompressor_ListCompressedModels(t *testing.T) {
	compressor := NewModelCompressor(nil)
	ctx := context.Background()

	modelData := make([]float32, 1000)

	config := &CompressionConfig{
		Method:           MethodQuantization,
		QuantizationType: QuantFP16,
	}

	compressor.Compress(ctx, "model-1", modelData, config)
	compressor.Compress(ctx, "model-2", modelData, config)

	models := compressor.ListCompressedModels()
	if len(models) != 2 {
		t.Errorf("Expected 2 compressed models, got %d", len(models))
	}
}

func TestModelCompressor_GetMetrics(t *testing.T) {
	compressor := NewModelCompressor(nil)

	metrics := compressor.GetMetrics()

	if metrics == nil {
		t.Fatal("Expected metrics to not be nil")
	}

	if metrics.TotalCompressions != 0 {
		t.Errorf("Expected 0 total compressions, got %d", metrics.TotalCompressions)
	}
}

func TestModelCompressor_Decompress(t *testing.T) {
	compressor := NewModelCompressor(nil)
	ctx := context.Background()

	modelData := make([]float32, 1000)
	for i := range modelData {
		modelData[i] = float32(i) / 100.0
	}

	config := &CompressionConfig{
		Method:           MethodQuantization,
		QuantizationType: QuantINT8,
	}

	result, err := compressor.Compress(ctx, "test-model", modelData, config)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	decompressed, err := compressor.Decompress(result.CompressedModel)
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}

	if decompressed == nil {
		t.Error("Expected decompressed model to not be nil")
	}
}

func TestModelCompressor_Calibrate(t *testing.T) {
	compressor := NewModelCompressor(nil)

	modelData := make([]float32, 100)
	samples := make([]interface{}, 10)
	for i := range samples {
		samples[i] = float32(i)
	}

	calibration, err := compressor.Calibrate(modelData, samples)
	if err != nil {
		t.Fatalf("Calibrate failed: %v", err)
	}

	if calibration == nil {
		t.Fatal("Expected calibration data to not be nil")
	}

	if len(calibration.Samples) != 10 {
		t.Errorf("Expected 10 samples, got %d", len(calibration.Samples))
	}

	if calibration.Stats == nil {
		t.Fatal("Expected stats to not be nil")
	}
}

func TestModelCompressor_AutoCompress(t *testing.T) {
	compressor := NewModelCompressor(nil)
	ctx := context.Background()

	modelData := make([]float32, 10000)

	result, err := compressor.AutoCompress(ctx, "test-model", modelData, 1.0)
	if err != nil {
		t.Fatalf("AutoCompress failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected auto compression to succeed")
	}

	if result.CompressedSizeMB > 10.0 {
		t.Logf("Compressed size: %f MB", result.CompressedSizeMB)
	}
}

func TestModelCompressor_GetCompressionRatio(t *testing.T) {
	compressor := NewModelCompressor(nil)
	ctx := context.Background()

	modelData := make([]float32, 1000)

	config := &CompressionConfig{
		Method:           MethodQuantization,
		QuantizationType: QuantINT8,
	}

	result, err := compressor.Compress(ctx, "test-model", modelData, config)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	ratio, err := compressor.GetCompressionRatio(result.CompressedModel.ID)
	if err != nil {
		t.Fatalf("GetCompressionRatio failed: %v", err)
	}

	if ratio <= 0 {
		t.Error("Expected positive compression ratio")
	}
}
