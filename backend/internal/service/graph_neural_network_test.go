package service

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestNewUserGraph(t *testing.T) {
	graph := NewUserGraph()
	if graph == nil {
		t.Fatal("NewUserGraph should not return nil")
	}
	if graph.Nodes == nil {
		t.Error("Nodes map should be initialized")
	}
	if graph.Edges == nil {
		t.Error("Edges map should be initialized")
	}
	if graph.Adjacency == nil {
		t.Error("Adjacency map should be initialized")
	}
}

func TestUserGraphAddNode(t *testing.T) {
	graph := NewUserGraph()

	node := &GraphNode{
		ID:       "node1",
		UserID:   "user1",
		SessionID: "session1",
		Features: make([]float64, GNNEmbeddingDim),
	}

	graph.AddNode(node)

	retrieved := graph.GetNode("node1")
	if retrieved == nil {
		t.Fatal("GetNode should return the added node")
	}
	if retrieved.ID != "node1" {
		t.Errorf("Expected ID 'node1', got '%s'", retrieved.ID)
	}
}

func TestUserGraphAddEdge(t *testing.T) {
	graph := NewUserGraph()

	node1 := &GraphNode{ID: "node1", UserID: "user1"}
	node2 := &GraphNode{ID: "node2", UserID: "user2"}

	graph.AddNode(node1)
	graph.AddNode(node2)

	edge := &GraphEdge{
		ID:       "edge1",
		SourceID: "node1",
		TargetID: "node2",
		Weight:   1.0,
	}

	err := graph.AddEdge(edge)
	if err != nil {
		t.Errorf("AddEdge failed: %v", err)
	}

	neighbors := graph.GetNeighbors("node1")
	if len(neighbors) != 1 {
		t.Errorf("Expected 1 neighbor, got %d", len(neighbors))
	}
	if neighbors[0] != "node2" {
		t.Errorf("Expected neighbor 'node2', got '%s'", neighbors[0])
	}
}

func TestUserGraphAddEdgeSourceNotFound(t *testing.T) {
	graph := NewUserGraph()

	node2 := &GraphNode{ID: "node2", UserID: "user2"}
	graph.AddNode(node2)

	edge := &GraphEdge{
		ID:       "edge1",
		SourceID: "node1",
		TargetID: "node2",
		Weight:   1.0,
	}

	err := graph.AddEdge(edge)
	if err == nil {
		t.Error("Expected error for non-existent source node")
	}
}

func TestUserGraphGetNodeNotFound(t *testing.T) {
	graph := NewUserGraph()

	node := graph.GetNode("nonexistent")
	if node != nil {
		t.Error("GetNode should return nil for non-existent node")
	}
}

func TestUserGraphRemoveNode(t *testing.T) {
	graph := NewUserGraph()

	node1 := &GraphNode{ID: "node1", UserID: "user1"}
	node2 := &GraphNode{ID: "node2", UserID: "user2"}

	graph.AddNode(node1)
	graph.AddNode(node2)

	edge := &GraphEdge{
		ID:       "edge1",
		SourceID: "node1",
		TargetID: "node2",
	}
	graph.AddEdge(edge)

	graph.RemoveNode("node1")

	if graph.GetNode("node1") != nil {
		t.Error("Removed node should not be found")
	}
}

func TestUserGraphCalculateCentrality(t *testing.T) {
	graph := NewUserGraph()

	for i := 0; i < 5; i++ {
		node := &GraphNode{
			ID:     fmt.Sprintf("node%d", i),
			UserID: fmt.Sprintf("user%d", i),
		}
		graph.AddNode(node)
	}

	graph.AddEdge(&GraphEdge{ID: "e1", SourceID: "node0", TargetID: "node1", Weight: 1.0})
	graph.AddEdge(&GraphEdge{ID: "e2", SourceID: "node0", TargetID: "node2", Weight: 1.0})
	graph.AddEdge(&GraphEdge{ID: "e3", SourceID: "node0", TargetID: "node3", Weight: 1.0})
	graph.AddEdge(&GraphEdge{ID: "e4", SourceID: "node0", TargetID: "node4", Weight: 1.0})

	centrality := graph.CalculateCentrality()

	if centrality["node0"] != 1.0 {
		t.Errorf("Expected centrality 1.0 for hub node, got %f", centrality["node0"])
	}
}

func TestUserGraphDetectCommunities(t *testing.T) {
	graph := NewUserGraph()

	for i := 0; i < 6; i++ {
		node := &GraphNode{
			ID:     fmt.Sprintf("node%d", i),
			UserID: fmt.Sprintf("user%d", i),
		}
		graph.AddNode(node)
	}

	graph.AddEdge(&GraphEdge{ID: "e1", SourceID: "node0", TargetID: "node1", Weight: 1.0})
	graph.AddEdge(&GraphEdge{ID: "e2", SourceID: "node1", TargetID: "node2", Weight: 1.0})
	graph.AddEdge(&GraphEdge{ID: "e3", SourceID: "node0", TargetID: "node2", Weight: 1.0})

	graph.AddEdge(&GraphEdge{ID: "e4", SourceID: "node3", TargetID: "node4", Weight: 1.0})
	graph.AddEdge(&GraphEdge{ID: "e5", SourceID: "node4", TargetID: "node5", Weight: 1.0})

	communities := graph.DetectCommunities()

	if len(communities) < 1 {
		t.Error("Should detect at least one community")
	}
}

func TestCommunityDetectAnomalousCommunity(t *testing.T) {
	community := &Community{
		ID:     "test_community",
		NodeIDs: []string{"n1", "n2", "n3"},
		AvgRisk: 0.8,
		Density: 0.1,
		Size:    3,
	}

	isAnomalous := community.detectAnomalousCommunity()
	if !isAnomalous {
		t.Error("Community with high risk should be anomalous")
	}
}

func TestNewGNN(t *testing.T) {
	config := &GNNConfig{
		EmbeddingDim: 64,
		NumLayers:     2,
		NumHeads:      2,
	}

	gnn := NewGNN(config)
	if gnn == nil {
		t.Fatal("NewGNN should not return nil")
	}
	if len(gnn.layers) != 2 {
		t.Errorf("Expected 2 layers, got %d", len(gnn.layers))
	}
}

func TestNewGNNDefaultConfig(t *testing.T) {
	gnn := NewGNN(nil)
	if gnn == nil {
		t.Fatal("NewGNN with nil config should not return nil")
	}
	if gnn.config.EmbeddingDim != GNNEmbeddingDim {
		t.Errorf("Expected default embedding dim %d, got %d", GNNEmbeddingDim, gnn.config.EmbeddingDim)
	}
}

func TestGNNInitialize(t *testing.T) {
	gnn := NewGNN(nil)
	ctx := context.Background()

	err := gnn.Initialize(ctx)
	if err != nil {
		t.Errorf("Initialize failed: %v", err)
	}
}

func TestGNNForward(t *testing.T) {
	gnn := NewGNN(nil)

	node := &GraphNode{
		ID:          "test_node",
		UserID:      "test_user",
		SessionID:   "test_session",
		Embeddings:  make([]float64, GNNEmbeddingDim),
	}
	for i := range node.Embeddings {
		node.Embeddings[i] = 0.1
	}

	gnn.graph.AddNode(node)

	embedding, err := gnn.Forward("test_node")
	if err != nil {
		t.Errorf("Forward failed: %v", err)
	}
	if embedding == nil {
		t.Error("Forward should return embedding")
	}
}

func TestGNNForwardNodeNotFound(t *testing.T) {
	gnn := NewGNN(nil)

	_, err := gnn.Forward("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent node")
	}
}

func TestGNNAddUserInteraction(t *testing.T) {
	gnn := NewGNN(nil)

	node1 := &GraphNode{ID: "n1", UserID: "u1"}
	node2 := &GraphNode{ID: "n2", UserID: "u2"}
	gnn.graph.AddNode(node1)
	gnn.graph.AddNode(node2)

	err := gnn.AddUserInteraction("n1", "n2", "interaction", 1.0, nil)
	if err != nil {
		t.Errorf("AddUserInteraction failed: %v", err)
	}
}

func TestGNNAnalyzeUserRelationships(t *testing.T) {
	gnn := NewGNN(nil)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		node := &GraphNode{
			ID:          fmt.Sprintf("n%d", i),
			UserID:      fmt.Sprintf("user%d", i),
			SessionID:   fmt.Sprintf("session%d", i),
			Embeddings:  make([]float64, GNNEmbeddingDim),
			Features:    make([]float64, GNNEmbeddingDim),
		}
		for j := range node.Embeddings {
			node.Embeddings[j] = 0.1 * float64(i+1)
		}
		gnn.graph.AddNode(node)
	}

	gnn.AddUserInteraction("n0", "n1", "follow", 0.8, nil)
	gnn.AddUserInteraction("n0", "n2", "follow", 0.6, nil)

	analysis, err := gnn.AnalyzeUserRelationships(ctx, "user0")
	if err != nil {
		t.Errorf("AnalyzeUserRelationships failed: %v", err)
	}

	if analysis == nil {
		t.Fatal("Analysis should not be nil")
	}

	if analysis.UserID != "user0" {
		t.Errorf("Expected UserID 'user0', got '%s'", analysis.UserID)
	}

	if analysis.TotalRelationships != 2 {
		t.Errorf("Expected 2 relationships, got %d", analysis.TotalRelationships)
	}
}

func TestGNNAnalyzeUserNotFound(t *testing.T) {
	gnn := NewGNN(nil)
	ctx := context.Background()

	_, err := gnn.AnalyzeUserRelationships(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent user")
	}
}

func TestGNNDetectAnomalousNodes(t *testing.T) {
	gnn := NewGNN(nil)
	ctx := context.Background()

	node := &GraphNode{
		ID:        "anomaly_node",
		UserID:    "anomaly_user",
		RiskLevel: 0.9,
		Centrality: 0.95,
	}
	gnn.graph.AddNode(node)

	anomalies := gnn.DetectAnomalousNodes(ctx)
	if len(anomalies) != 1 {
		t.Errorf("Expected 1 anomalous node, got %d", len(anomalies))
	}
}

func TestGNNRunCommunityDetection(t *testing.T) {
	gnn := NewGNN(nil)

	for i := 0; i < 5; i++ {
		node := &GraphNode{
			ID:     fmt.Sprintf("node%d", i),
			UserID: fmt.Sprintf("user%d", i),
		}
		gnn.graph.AddNode(node)
	}

	gnn.AddUserInteraction("node0", "node1", "edge", 1.0, nil)
	gnn.AddUserInteraction("node1", "node2", "edge", 1.0, nil)
	gnn.AddUserInteraction("node0", "node2", "edge", 1.0, nil)

	communities := gnn.RunCommunityDetection()

	if len(communities) < 1 {
		t.Error("Should detect at least one community")
	}
}

func TestGNNGetCommunities(t *testing.T) {
	gnn := NewGNN(nil)

	gnn.RunCommunityDetection()

	communities := gnn.GetCommunities()
	if communities == nil {
		t.Error("GetCommunities should not return nil")
	}
}

func TestGNNGetAnomalousCommunities(t *testing.T) {
	gnn := NewGNN(nil)

	gnn.RunCommunityDetection()

	anomalous := gnn.GetAnomalousCommunities()
	if anomalous == nil {
		t.Error("GetAnomalousCommunities should not return nil")
	}
}

func TestGNNService(t *testing.T) {
	service := NewGNNService(nil)
	if service == nil {
		t.Fatal("NewGNNService should not return nil")
	}

	ctx := context.Background()
	err := service.Initialize(ctx)
	if err != nil {
		t.Errorf("Initialize failed: %v", err)
	}
}

func TestGNNServiceCreateUserGraph(t *testing.T) {
	service := NewGNNService(nil)

	graph := service.CreateUserGraph("test_graph")
	if graph == nil {
		t.Fatal("CreateUserGraph should not return nil")
	}

	stats := service.GetGraphStats("test_graph")
	if stats == nil {
		t.Error("GetGraphStats should not return nil")
	}
	if stats["total_nodes"].(int) != 0 {
		t.Errorf("Expected 0 nodes, got %d", stats["total_nodes"].(int))
	}
}

func TestGNNServiceAddUserNode(t *testing.T) {
	service := NewGNNService(nil)
	ctx := context.Background()
	service.Initialize(ctx)

	service.CreateUserGraph("test_graph")

	node := &GraphNode{
		ID:       "node1",
		UserID:   "user1",
		SessionID: "session1",
	}

	err := service.AddUserNode(ctx, "test_graph", node)
	if err != nil {
		t.Errorf("AddUserNode failed: %v", err)
	}
}

func TestGNNServiceAddUserEdge(t *testing.T) {
	service := NewGNNService(nil)
	ctx := context.Background()
	service.Initialize(ctx)

	graph := service.CreateUserGraph("test_graph")

	node1 := &GraphNode{ID: "n1", UserID: "u1"}
	node2 := &GraphNode{ID: "n2", UserID: "u2"}
	graph.AddNode(node1)
	graph.AddNode(node2)

	edge := &GraphEdge{
		ID:       "e1",
		SourceID: "n1",
		TargetID: "n2",
		Weight:   1.0,
	}

	err := service.AddUserEdge(ctx, "test_graph", edge)
	if err != nil {
		t.Errorf("AddUserEdge failed: %v", err)
	}
}

func TestGNNServiceRunAnalysis(t *testing.T) {
	service := NewGNNService(nil)
	ctx := context.Background()
	service.Initialize(ctx)

	graph := service.CreateUserGraph("test_graph")

	for i := 0; i < 3; i++ {
		node := &GraphNode{
			ID:     fmt.Sprintf("node%d", i),
			UserID: fmt.Sprintf("user%d", i),
		}
		graph.AddNode(node)
	}

	result, err := service.RunAnalysis(ctx, "test_graph")
	if err != nil {
		t.Errorf("RunAnalysis failed: %v", err)
	}

	if result == nil {
		t.Fatal("RunAnalysis result should not be nil")
	}

	if result.GraphID != "test_graph" {
		t.Errorf("Expected GraphID 'test_graph', got '%s'", result.GraphID)
	}
}

func TestGNNServiceAnalyzeUser(t *testing.T) {
	service := NewGNNService(nil)
	ctx := context.Background()
	service.Initialize(ctx)

	graph := service.CreateUserGraph("test_graph")

	node := &GraphNode{
		ID:          "n1",
		UserID:      "test_user",
		SessionID:   "session1",
		Embeddings:  make([]float64, GNNEmbeddingDim),
	}
	for i := range node.Embeddings {
		node.Embeddings[i] = 0.1
	}
	graph.AddNode(node)

	analysis, err := service.AnalyzeUser(ctx, "test_graph", "test_user")
	if err != nil {
		t.Errorf("AnalyzeUser failed: %v", err)
	}

	if analysis == nil {
		t.Fatal("AnalyzeUser result should not be nil")
	}

	if analysis.UserID != "test_user" {
		t.Errorf("Expected UserID 'test_user', got '%s'", analysis.UserID)
	}
}

func TestCosineSimilarity(t *testing.T) {
	a := []float64{1.0, 0.0, 0.0}
	b := []float64{1.0, 0.0, 0.0}

	similarity := cosineSimilarity(a, b)
	if similarity < 0.999 || similarity > 1.001 {
		t.Errorf("Expected similarity ~1.0, got %f", similarity)
	}

	c := []float64{0.0, 1.0, 0.0}
	similarity2 := cosineSimilarity(a, c)
	if similarity2 > 0.1 {
		t.Errorf("Expected similarity ~0 for orthogonal vectors, got %f", similarity2)
	}
}

func TestCosineSimilarityDifferentLengths(t *testing.T) {
	a := []float64{1.0, 0.0}
	b := []float64{1.0, 0.0, 0.0}

	similarity := cosineSimilarity(a, b)
	if similarity != 0.0 {
		t.Errorf("Expected similarity 0 for different lengths, got %f", similarity)
	}
}

func TestGraphNodeFeaturesInitialization(t *testing.T) {
	node := &GraphNode{
		ID:     "test_node",
		UserID: "test_user",
	}

	if node.Features != nil {
		t.Error("Features should be nil initially")
	}
	if node.Embeddings != nil {
		t.Error("Embeddings should be nil initially")
	}
	if node.Metadata != nil {
		t.Error("Metadata should be nil initially")
	}

	node.Features = make([]float64, GNNEmbeddingDim)
	node.Embeddings = make([]float64, GNNEmbeddingDim)
	node.Metadata = make(map[string]interface{})

	if len(node.Features) != GNNEmbeddingDim {
		t.Errorf("Expected Features length %d, got %d", GNNEmbeddingDim, len(node.Features))
	}
}

func TestGraphConvolutionWithNoNeighbors(t *testing.T) {
	gnn := NewGNN(nil)

	result := gnn.GraphConvolution("node1", [][]float64{}, []float64{}, 0)

	if len(result) != GNNEmbeddingDim {
		t.Errorf("Expected result length %d, got %d", GNNEmbeddingDim, len(result))
	}
}

func TestGraphConvolutionWithSingleNeighbor(t *testing.T) {
	gnn := NewGNN(nil)

	neighborEmbedding := make([]float64, GNNEmbeddingDim)
	for i := range neighborEmbedding {
		neighborEmbedding[i] = 0.5
	}

	result := gnn.GraphConvolution("node1", [][]float64{neighborEmbedding}, []float64{1.0}, 0)

	if len(result) != GNNEmbeddingDim {
		t.Errorf("Expected result length %d, got %d", GNNEmbeddingDim, len(result))
	}
}

func TestUserRelationshipAnalysisStructure(t *testing.T) {
	analysis := &UserRelationshipAnalysis{
		UserID:                  "user1",
		NodeID:                 "node1",
		TotalRelationships:     3,
		Relationships:          []RelationshipStrength{},
		Centrality:             0.75,
		CommunityID:            "community1",
		CommunitySize:          10,
		CommunityRisk:          0.3,
		IsInAnomalousCommunity: false,
		AnomalyReasons:         []string{},
		ProcessedAt:            time.Now(),
	}

	if analysis.UserID != "user1" {
		t.Errorf("Expected UserID 'user1', got '%s'", analysis.UserID)
	}

	if analysis.TotalRelationships != 3 {
		t.Errorf("Expected 3 relationships, got %d", analysis.TotalRelationships)
	}
}

func TestGraphAnalysisResultStructure(t *testing.T) {
	result := &GraphAnalysisResult{
		GraphID:                 "graph1",
		TotalNodes:              20,
		TotalEdges:              50,
		CommunityCount:          5,
		AnomalousNodeCount:      2,
		AnomalousCommunityCount: 1,
		CentralityMetrics:      map[string]float64{"node1": 0.8},
		Communities:            []*Community{},
		AnomalousNodes:         []*GraphNode{},
		AnomalousCommunities:   []*Community{},
		ProcessedAt:             time.Now(),
	}

	if result.GraphID != "graph1" {
		t.Errorf("Expected GraphID 'graph1', got '%s'", result.GraphID)
	}

	if result.TotalNodes != 20 {
		t.Errorf("Expected 20 nodes, got %d", result.TotalNodes)
	}
}

func TestRelationshipStrengthStructure(t *testing.T) {
	strength := RelationshipStrength{
		UserID:        "user2",
		SessionID:     "session2",
		EdgeWeight:    0.9,
		Similarity:    0.85,
		CombinedScore: 0.875,
	}

	if strength.CombinedScore != (strength.EdgeWeight+strength.Similarity)/2.0 {
		t.Error("CombinedScore should be average of EdgeWeight and Similarity")
	}
}

func TestCommunityStructure(t *testing.T) {
	community := &Community{
		ID:            "comm1",
		NodeIDs:       []string{"n1", "n2", "n3"},
		Centroid:      make([]float64, GNNEmbeddingDim),
		Size:          3,
		Density:       0.6,
		AvgRisk:       0.25,
		IsAnomalous:   false,
		AnomalyReasons: []string{},
		CreatedAt:     time.Now(),
	}

	if community.Size != 3 {
		t.Errorf("Expected size 3, got %d", community.Size)
	}

	if community.Density != 0.6 {
		t.Errorf("Expected density 0.6, got %f", community.Density)
	}
}

func BenchmarkUserGraphAddNode(b *testing.B) {
	graph := NewUserGraph()

	for i := 0; i < b.N; i++ {
		node := &GraphNode{
			ID:     fmt.Sprintf("node%d", i),
			UserID: fmt.Sprintf("user%d", i),
		}
		graph.AddNode(node)
	}
}

func BenchmarkGNNForward(b *testing.B) {
	gnn := NewGNN(nil)

	node := &GraphNode{
		ID:          "test_node",
		UserID:      "test_user",
		Embeddings:  make([]float64, GNNEmbeddingDim),
	}
	for i := range node.Embeddings {
		node.Embeddings[i] = 0.1
	}
	gnn.graph.AddNode(node)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gnn.Forward("test_node")
	}
}

func BenchmarkCalculateCentrality(b *testing.B) {
	graph := NewUserGraph()

	for i := 0; i < 100; i++ {
		node := &GraphNode{
			ID:     fmt.Sprintf("node%d", i),
			UserID: fmt.Sprintf("user%d", i),
		}
		graph.AddNode(node)
	}

	for i := 0; i < 99; i++ {
		graph.AddEdge(&GraphEdge{
			ID:       fmt.Sprintf("e%d", i),
			SourceID: fmt.Sprintf("node%d", i),
			TargetID: fmt.Sprintf("node%d", i+1),
			Weight:   1.0,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		graph.CalculateCentrality()
	}
}
