package service

import (
	"context"
	"testing"
)

func TestTransformerXL(t *testing.T) {
	ctx := context.Background()
	txl := NewTransformerXL(128, 2, 4, 256, 32)
	
	if err := txl.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize TransformerXL: %v", err)
	}
	
	seqLen := 10
	sequence := make([]float64, seqLen*128)
	for i := range sequence {
		sequence[i] = float64(i%10) / 10.0
	}
	
	result, err := txl.Forward(ctx, sequence, seqLen)
	if err != nil {
		t.Fatalf("TransformerXL forward pass failed: %v", err)
	}
	
	if len(result) != len(sequence) {
		t.Errorf("Expected output length %d, got %d", len(sequence), len(result))
	}
	
	txl.ResetMemory()
}

func TestGraphNeuralNetwork(t *testing.T) {
	ctx := context.Background()
	gnn := NewGraphNeuralNetwork(16, 32, 8, 2)
	
	if err := gnn.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize GNN: %v", err)
	}
	
	nodeFeatures := [][]float64{
		{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6},
		{0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7},
		{0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8},
	}
	
	adjacency := [][]float64{
		{0.0, 1.0, 1.0},
		{1.0, 0.0, 1.0},
		{1.0, 1.0, 0.0},
	}
	
	result, err := gnn.Forward(ctx, nodeFeatures, adjacency)
	if err != nil {
		t.Fatalf("GNN forward pass failed: %v", err)
	}
	
	if len(result) != len(nodeFeatures) {
		t.Errorf("Expected %d nodes, got %d", len(nodeFeatures), len(result))
	}
}

func TestSelfSupervisedPretraining(t *testing.T) {
	ctx := context.Background()
	service := NewAIModelV4Service()
	
	if err := service.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize service: %v", err)
	}
	
	seqLen := 16
	dModel := 512
	sequence := make([]float64, seqLen*dModel)
	for i := range sequence {
		sequence[i] = float64(i%100) / 100.0
	}
	
	predicted, loss, err := service.PretrainStep(ctx, sequence, seqLen)
	if err != nil {
		t.Fatalf("Pretrain step failed: %v", err)
	}
	
	if len(predicted) != len(sequence) {
		t.Errorf("Expected prediction length %d, got %d", len(sequence), len(predicted))
	}
	
	if loss < 0 {
		t.Errorf("Loss should be non-negative, got %f", loss)
	}
	
	sequences := [][]float64{sequence, sequence, sequence, sequence}
	contrastiveLoss, err := service.ContrastiveLearning(ctx, sequences, seqLen, 0.1)
	if err != nil {
		t.Fatalf("Contrastive learning failed: %v", err)
	}
	
	if contrastiveLoss < 0 {
		t.Errorf("Contrastive loss should be non-negative, got %f", contrastiveLoss)
	}
}
