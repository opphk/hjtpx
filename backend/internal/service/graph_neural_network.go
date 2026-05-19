package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

const (
	GNNEmbeddingDim   = 128
	GNNNumLayers      = 3
	GNNNumHeads       = 4
	CommunityMinSize  = 3
	CommunityMaxSize  = 50
)

type GNNConfig struct {
	EmbeddingDim int
	NumLayers     int
	NumHeads      int
	Dropout       float64
	LearningRate  float64
}

type UserGraph struct {
	Nodes     map[string]*GraphNode
	Edges     map[string]*GraphEdge
	Adjacency map[string][]string
	mu        sync.RWMutex
}

type GraphNode struct {
	ID           string
	UserID       string
	SessionID    string
	Features     []float64
	Embeddings   []float64
	Role         string
	RiskLevel    float64
	CommunityID  string
	Centrality   float64
	IsAnomaly    bool
	Metadata     map[string]interface{}
	LastUpdated  time.Time
}

type GraphEdge struct {
	ID        string
	SourceID  string
	TargetID  string
	EdgeType  string
	Weight    float64
	Timestamp int64
	Count     int
	Metadata  map[string]interface{}
}

type GraphNodeFeature struct {
	BehavioralFeatures []float64
	DeviceFeatures     []float64
	SessionFeatures    []float64
	RiskFeatures       []float64
}

type GNNMessage struct {
	FromNodeID  string
	ToNodeID    string
	Embeddings  []float64
	EdgeWeight  float64
	MessageType string
}

type GNNAggregation struct {
	Mean     []float64
	Max      []float64
	Min      []float64
	Std      []float64
	Attention []float64
}

type Community struct {
	ID          string
	NodeIDs     []string
	Centroid    []float64
	Size        int
	Density     float64
	AvgRisk     float64
	IsAnomalous bool
	AnomalyReasons []string
	CreatedAt   time.Time
}

type GraphConvLayer struct {
	Weights     [][]float64
	Bias        []float64
	AttentionWeights [][][]float64
	LayerNormGamma []float64
	LayerNormBeta  []float64
}

func NewUserGraph() *UserGraph {
	return &UserGraph{
		Nodes:     make(map[string]*GraphNode),
		Edges:     make(map[string]*GraphEdge),
		Adjacency: make(map[string][]string),
	}
}

func NewGNN(config *GNNConfig) *GraphNeuralNetwork {
	if config == nil {
		config = &GNNConfig{
			EmbeddingDim: GNNEmbeddingDim,
			NumLayers:    GNNNumLayers,
			NumHeads:     GNNNumHeads,
			Dropout:      0.1,
			LearningRate: 0.001,
		}
	}

	gnn := &GraphNeuralNetwork{
		config:       *config,
		layers:       make([]*GraphConvLayer, config.NumLayers),
		graph:        NewUserGraph(),
		communities:  make(map[string]*Community),
	}

	for i := 0; i < config.NumLayers; i++ {
		gnn.layers[i] = &GraphConvLayer{
			Weights:         createRandomMatrix(config.EmbeddingDim, config.EmbeddingDim, 0.02),
			Bias:            createRandomVector(config.EmbeddingDim, 0.02),
			AttentionWeights: make([][][]float64, config.NumHeads),
			LayerNormGamma:  createLayerNormParamsVec(config.EmbeddingDim),
			LayerNormBeta:    createLayerNormParamsVec(config.EmbeddingDim),
		}

		for h := 0; h < config.NumHeads; h++ {
			gnn.layers[i].AttentionWeights[h] = createRandomMatrix(config.EmbeddingDim, config.EmbeddingDim, 0.02)
		}
	}

	return gnn
}

func createLayerNormParamsVec(dim int) []float64 {
	vec := make([]float64, dim)
	for i := range vec {
		vec[i] = 1.0
	}
	return vec
}

func (g *UserGraph) AddNode(node *GraphNode) {
	g.mu.Lock()
	defer g.mu.Unlock()

	node.LastUpdated = time.Now()
	if node.Features == nil {
		node.Features = make([]float64, GNNEmbeddingDim)
	}
	if node.Embeddings == nil {
		node.Embeddings = make([]float64, GNNEmbeddingDim)
	}
	if node.Metadata == nil {
		node.Metadata = make(map[string]interface{})
	}

	g.Nodes[node.ID] = node
	if g.Adjacency[node.ID] == nil {
		g.Adjacency[node.ID] = []string{}
	}
}

func (g *UserGraph) AddEdge(edge *GraphEdge) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.Nodes[edge.SourceID]; !exists {
		return fmt.Errorf("source node %s does not exist", edge.SourceID)
	}
	if _, exists := g.Nodes[edge.TargetID]; !exists {
		return fmt.Errorf("target node %s does not exist", edge.TargetID)
	}

	g.Edges[edge.ID] = edge

	found := false
	for _, neighbor := range g.Adjacency[edge.SourceID] {
		if neighbor == edge.TargetID {
			found = true
			break
		}
	}
	if !found {
		g.Adjacency[edge.SourceID] = append(g.Adjacency[edge.SourceID], edge.TargetID)
	}

	found = false
	for _, neighbor := range g.Adjacency[edge.TargetID] {
		if neighbor == edge.SourceID {
			found = true
			break
		}
	}
	if !found {
		g.Adjacency[edge.TargetID] = append(g.Adjacency[edge.TargetID], edge.SourceID)
	}

	return nil
}

func (g *UserGraph) GetNeighbors(nodeID string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	neighbors := g.Adjacency[nodeID]
	result := make([]string, len(neighbors))
	copy(result, neighbors)
	return result
}

func (g *UserGraph) GetNode(nodeID string) *GraphNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	node, exists := g.Nodes[nodeID]
	if !exists {
		return nil
	}

	nodeCopy := &GraphNode{
		ID:          node.ID,
		UserID:      node.UserID,
		SessionID:   node.SessionID,
		Features:    make([]float64, len(node.Features)),
		Embeddings:  make([]float64, len(node.Embeddings)),
		Role:        node.Role,
		RiskLevel:   node.RiskLevel,
		CommunityID: node.CommunityID,
		Centrality:  node.Centrality,
		IsAnomaly:   node.IsAnomaly,
		Metadata:    make(map[string]interface{}),
		LastUpdated: node.LastUpdated,
	}
	copy(nodeCopy.Features, node.Features)
	copy(nodeCopy.Embeddings, node.Embeddings)
	for k, v := range node.Metadata {
		nodeCopy.Metadata[k] = v
	}

	return nodeCopy
}

func (g *UserGraph) RemoveNode(nodeID string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	delete(g.Nodes, nodeID)

	for _, neighbors := range g.Adjacency {
		newNeighbors := make([]string, 0)
		for _, n := range neighbors {
			if n != nodeID {
				newNeighbors = append(newNeighbors, n)
			}
		}
		neighbors = newNeighbors
	}
	delete(g.Adjacency, nodeID)

	var edgesToRemove []string
	for id, edge := range g.Edges {
		if edge.SourceID == nodeID || edge.TargetID == nodeID {
			edgesToRemove = append(edgesToRemove, id)
		}
	}
	for _, id := range edgesToRemove {
		delete(g.Edges, id)
	}
}

func (g *UserGraph) CalculateCentrality() map[string]float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()

	centrality := make(map[string]float64)
	degree := make(map[string]int)

	for nodeID := range g.Nodes {
		degree[nodeID] = len(g.Adjacency[nodeID])
	}

	maxDegree := 0
	for _, d := range degree {
		if d > maxDegree {
			maxDegree = d
		}
	}

	if maxDegree > 0 {
		for nodeID, d := range degree {
			centrality[nodeID] = float64(d) / float64(maxDegree)
		}
	}

	return centrality
}

func (g *UserGraph) DetectCommunities() []*Community {
	g.mu.Lock()
	defer g.mu.Unlock()

	visited := make(map[string]bool)
	var communities []*Community
	communityID := 0

	for nodeID := range g.Nodes {
		if visited[nodeID] {
			continue
		}

		var communityNodes []string
		g.bfsCollect(nodeID, visited, &communityNodes, 3)

		if len(communityNodes) >= CommunityMinSize {
			community := g.createCommunity(communityNodes, communityID)
			communities = append(communities, community)
			communityID++
		}
	}

	for _, community := range communities {
		for _, nodeID := range community.NodeIDs {
			if node, exists := g.Nodes[nodeID]; exists {
				node.CommunityID = community.ID
			}
		}
	}

	return communities
}

func (g *UserGraph) bfsCollect(startID string, visited map[string]bool, nodes *[]string, maxDepth int) {
	if visited[startID] || maxDepth < 0 {
		return
	}

	visited[startID] = true
	*nodes = append(*nodes, startID)

	for _, neighbor := range g.Adjacency[startID] {
		if !visited[neighbor] {
			g.bfsCollect(neighbor, visited, nodes, maxDepth-1)
		}
	}
}

func (g *UserGraph) createCommunity(nodeIDs []string, communityIndex int) *Community {
	community := &Community{
		ID:            fmt.Sprintf("community_%d", communityIndex),
		NodeIDs:       make([]string, len(nodeIDs)),
		Centroid:      make([]float64, GNNEmbeddingDim),
		AvgRisk:       0.0,
		AnomalyReasons: []string{},
		CreatedAt:     time.Now(),
	}
	copy(community.NodeIDs, nodeIDs)
	community.Size = len(nodeIDs)

	riskSum := 0.0
	for _, nodeID := range nodeIDs {
		if node, exists := g.Nodes[nodeID]; exists {
			for i, f := range node.Features {
				community.Centroid[i] += f / float64(len(nodeIDs))
			}
			riskSum += node.RiskLevel
		}
	}
	if len(nodeIDs) > 0 {
		community.AvgRisk = riskSum / float64(len(nodeIDs))
	}

	community.Density = g.calculateCommunityDensity(nodeIDs)
	community.IsAnomalous = community.detectAnomalousCommunity()

	return community
}

func (g *UserGraph) calculateCommunityDensity(nodeIDs []string) float64 {
	if len(nodeIDs) < 2 {
		return 0.0
	}

	nodeSet := make(map[string]bool)
	for _, id := range nodeIDs {
		nodeSet[id] = true
	}

	edgeCount := 0
	for _, id := range nodeIDs {
		for _, neighbor := range g.Adjacency[id] {
			if nodeSet[neighbor] && id < neighbor {
				edgeCount++
			}
		}
	}

	maxEdges := float64(len(nodeIDs) * (len(nodeIDs) - 1) / 2)
	if maxEdges > 0 {
		return float64(edgeCount) / maxEdges
	}
	return 0.0
}

func (c *Community) detectAnomalousCommunity() bool {
	if c.AvgRisk > 0.7 {
		c.AnomalyReasons = append(c.AnomalyReasons, "高平均风险")
	}

	if c.Density < 0.2 {
		c.AnomalyReasons = append(c.AnomalyReasons, "低图密度")
	}

	if c.Size > CommunityMaxSize {
		c.AnomalyReasons = append(c.AnomalyReasons, "社区规模过大")
	}

	return len(c.AnomalyReasons) > 0
}

type GraphNeuralNetwork struct {
	config       GNNConfig
	layers       []*GraphConvLayer
	graph        *UserGraph
	communities  map[string]*Community
	anomalyCache map[string]bool
	mu           sync.RWMutex
}

func (gnn *GraphNeuralNetwork) Initialize(ctx context.Context) error {
	gnn.mu.Lock()
	defer gnn.mu.Unlock()

	gnn.anomalyCache = make(map[string]bool)
	return nil
}

func (gnn *GraphNeuralNetwork) GraphConvolution(nodeID string, neighborEmbeddings [][]float64, edgeWeights []float64, layerIdx int) []float64 {
	if len(neighborEmbeddings) == 0 {
		return make([]float64, gnn.config.EmbeddingDim)
	}

	layer := gnn.layers[layerIdx]
	aggregated := make([]float64, gnn.config.EmbeddingDim)

	if len(neighborEmbeddings) == 1 {
		for j := range aggregated {
			aggregated[j] = neighborEmbeddings[0][j]
		}
	} else {
		attentions := make([][]float64, gnn.config.NumHeads)
		for h := range attentions {
			attentions[h] = make([]float64, len(neighborEmbeddings))
		}

		for i, emb := range neighborEmbeddings {
			for h := 0; h < gnn.config.NumHeads; h++ {
				attentions[h][i] = math.Tanh(dotProduct(emb, layer.AttentionWeights[h][0]))
			}
		}

		for h := range attentions {
			attentions[h] = softmax(attentions[h])
		}

		for j := range aggregated {
			for i, emb := range neighborEmbeddings {
				weight := edgeWeights[i]
				if weight <= 0 {
					weight = 1.0
				}
				attnSum := 0.0
				for h := 0; h < gnn.config.NumHeads; h++ {
					attnSum += attentions[h][i]
				}
				attnAvg := attnSum / float64(gnn.config.NumHeads)
				aggregated[j] += emb[j] * weight * attnAvg
			}
		}

		norm := float64(len(neighborEmbeddings))
		if norm > 0 {
			for j := range aggregated {
				aggregated[j] /= norm
			}
		}
	}

	output := make([]float64, gnn.config.EmbeddingDim)
	for i := range output {
		for j := range aggregated {
			output[i] += aggregated[j] * layer.Weights[i][j]
		}
		output[i] += layer.Bias[i]
	}

	output = relu(output)

	return output
}

func (gnn *GraphNeuralNetwork) propagate() {
	gnn.mu.Lock()
	defer gnn.mu.Unlock()

	for iter := 0; iter < gnn.config.NumLayers; iter++ {
		newEmbeddings := make(map[string][]float64)

		for nodeID := range gnn.graph.Nodes {
			neighbors := gnn.graph.GetNeighbors(nodeID)
			if len(neighbors) == 0 {
				newEmbeddings[nodeID] = gnn.graph.Nodes[nodeID].Embeddings
				continue
			}

			neighborEmbeddings := make([][]float64, 0, len(neighbors))
			edgeWeights := make([]float64, 0, len(neighbors))

			for _, neighborID := range neighbors {
				neighbor := gnn.graph.GetNode(neighborID)
				if neighbor != nil {
					neighborEmbeddings = append(neighborEmbeddings, neighbor.Embeddings)
					edgeWeight := 1.0
					for _, edge := range gnn.graph.Edges {
						if (edge.SourceID == nodeID && edge.TargetID == neighborID) ||
						   (edge.SourceID == neighborID && edge.TargetID == nodeID) {
							edgeWeight = edge.Weight
							break
						}
					}
					edgeWeights = append(edgeWeights, edgeWeight)
				}
			}

			newEmbeddings[nodeID] = gnn.GraphConvolution(nodeID, neighborEmbeddings, edgeWeights, iter)
		}

		for nodeID, embedding := range newEmbeddings {
			if node, exists := gnn.graph.Nodes[nodeID]; exists {
				copy(node.Embeddings, embedding)
			}
		}
	}
}

func (gnn *GraphNeuralNetwork) Forward(nodeID string) ([]float64, error) {
	gnn.mu.RLock()
	defer gnn.mu.RUnlock()

	node, exists := gnn.graph.Nodes[nodeID]
	if !exists {
		return nil, fmt.Errorf("node %s not found", nodeID)
	}

	embedding := make([]float64, len(node.Embeddings))
	copy(embedding, node.Embeddings)

	return embedding, nil
}

func (gnn *GraphNeuralNetwork) AddUserInteraction(sourceID, targetID, edgeType string, weight float64, metadata map[string]interface{}) error {
	edgeID := fmt.Sprintf("%s_%s_%s", sourceID, targetID, edgeType)

	edge := &GraphEdge{
		ID:        edgeID,
		SourceID:  sourceID,
		TargetID:  targetID,
		EdgeType:  edgeType,
		Weight:    weight,
		Timestamp: time.Now().Unix(),
		Count:     1,
		Metadata:  metadata,
	}

	return gnn.graph.AddEdge(edge)
}

func (gnn *GraphNeuralNetwork) AnalyzeUserRelationships(ctx context.Context, userID string) (*UserRelationshipAnalysis, error) {
	gnn.mu.RLock()
	defer gnn.mu.RUnlock()

	var userNode *GraphNode
	for _, node := range gnn.graph.Nodes {
		if node.UserID == userID {
			userNode = node
			break
		}
	}

	if userNode == nil {
		return nil, fmt.Errorf("user %s not found in graph", userID)
	}

	neighbors := gnn.graph.GetNeighbors(userNode.ID)
	relationshipStrengths := make([]RelationshipStrength, 0, len(neighbors))

	for _, neighborID := range neighbors {
		neighbor := gnn.graph.GetNode(neighborID)
		if neighbor == nil {
			continue
		}

		edgeWeight := 1.0
		for _, edge := range gnn.graph.Edges {
			if (edge.SourceID == userNode.ID && edge.TargetID == neighborID) ||
			   (edge.SourceID == neighborID && edge.TargetID == userNode.ID) {
				edgeWeight = edge.Weight
				break
			}
		}

		similarity := cosineSimilarity(userNode.Embeddings, neighbor.Embeddings)

		strength := RelationshipStrength{
			UserID:       neighbor.UserID,
			SessionID:    neighbor.SessionID,
			EdgeWeight:  edgeWeight,
			Similarity:  similarity,
			CombinedScore: (edgeWeight + similarity) / 2.0,
		}
		relationshipStrengths = append(relationshipStrengths, strength)
	}

	sort.Slice(relationshipStrengths, func(i, j int) bool {
		return relationshipStrengths[i].CombinedScore > relationshipStrengths[j].CombinedScore
	})

	communityID := userNode.CommunityID
	var community *Community
	if communityID != "" {
		community = gnn.communities[communityID]
	}

	analysis := &UserRelationshipAnalysis{
		UserID:              userID,
		NodeID:              userNode.ID,
		TotalRelationships:  len(neighbors),
		Relationships:       relationshipStrengths,
		Centrality:          userNode.Centrality,
		CommunityID:         communityID,
		CommunitySize:       0,
		CommunityRisk:       0.0,
		IsInAnomalousCommunity: false,
		AnomalyReasons:      []string{},
		ProcessedAt:         time.Now(),
	}

	if community != nil {
		analysis.CommunitySize = community.Size
		analysis.CommunityRisk = community.AvgRisk
		analysis.IsInAnomalousCommunity = community.IsAnomalous
		analysis.AnomalyReasons = community.AnomalyReasons
	}

	return analysis, nil
}

func (gnn *GraphNeuralNetwork) DetectAnomalousNodes(ctx context.Context) []*GraphNode {
	gnn.mu.RLock()
	defer gnn.mu.RUnlock()

	var anomalies []*GraphNode

	for _, node := range gnn.graph.Nodes {
		if gnn.isNodeAnomalous(node) {
			nodeCopy := &GraphNode{
				ID:           node.ID,
				UserID:       node.UserID,
				SessionID:    node.SessionID,
				Features:     node.Features,
				Embeddings:   node.Embeddings,
				Role:         node.Role,
				RiskLevel:    node.RiskLevel,
				CommunityID:  node.CommunityID,
				Centrality:   node.Centrality,
				IsAnomaly:    true,
				Metadata:     node.Metadata,
				LastUpdated:  node.LastUpdated,
			}
			anomalies = append(anomalies, nodeCopy)
		}
	}

	return anomalies
}

func (gnn *GraphNeuralNetwork) isNodeAnomalous(node *GraphNode) bool {
	if node.RiskLevel > 0.8 {
		return true
	}

	if node.Centrality > 0.9 {
		return true
	}

	if node.CommunityID != "" {
		if community, exists := gnn.communities[node.CommunityID]; exists {
			if community.IsAnomalous && community.AvgRisk > 0.6 {
				return true
			}
		}
	}

	return false
}

func (gnn *GraphNeuralNetwork) RunCommunityDetection() []*Community {
	communities := gnn.graph.DetectCommunities()

	gnn.mu.Lock()
	defer gnn.mu.Unlock()

	gnn.communities = make(map[string]*Community)
	for _, community := range communities {
		gnn.communities[community.ID] = community
	}

	return communities
}

func (gnn *GraphNeuralNetwork) GetCommunities() map[string]*Community {
	gnn.mu.RLock()
	defer gnn.mu.RUnlock()

	result := make(map[string]*Community)
	for k, v := range gnn.communities {
		result[k] = v
	}

	return result
}

func (gnn *GraphNeuralNetwork) GetAnomalousCommunities() []*Community {
	gnn.mu.RLock()
	defer gnn.mu.RUnlock()

	var anomalous []*Community
	for _, community := range gnn.communities {
		if community.IsAnomalous {
			anomalous = append(anomalous, community)
		}
	}

	return anomalous
}

type UserRelationshipAnalysis struct {
	UserID                  string
	NodeID                  string
	TotalRelationships      int
	Relationships           []RelationshipStrength
	Centrality              float64
	CommunityID             string
	CommunitySize           int
	CommunityRisk           float64
	IsInAnomalousCommunity  bool
	AnomalyReasons          []string
	ProcessedAt             time.Time
}

type RelationshipStrength struct {
	UserID        string
	SessionID     string
	EdgeWeight    float64
	Similarity    float64
	CombinedScore float64
}

type GNNService struct {
	gnn     *GraphNeuralNetwork
	graphs  map[string]*UserGraph
	mu      sync.RWMutex
}

func NewGNNService(config *GNNConfig) *GNNService {
	return &GNNService{
		gnn:    NewGNN(config),
		graphs: make(map[string]*UserGraph),
	}
}

func (s *GNNService) Initialize(ctx context.Context) error {
	return s.gnn.Initialize(ctx)
}

func (s *GNNService) CreateUserGraph(graphID string) *UserGraph {
	s.mu.Lock()
	defer s.mu.Unlock()

	graph := NewUserGraph()
	s.graphs[graphID] = graph
	return graph
}

func (s *GNNService) AddUserNode(ctx context.Context, graphID string, node *GraphNode) error {
	s.mu.RLock()
	graph, exists := s.graphs[graphID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("graph %s not found", graphID)
	}

	graph.AddNode(node)
	return nil
}

func (s *GNNService) AddUserEdge(ctx context.Context, graphID string, edge *GraphEdge) error {
	s.mu.RLock()
	graph, exists := s.graphs[graphID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("graph %s not found", graphID)
	}

	return graph.AddEdge(edge)
}

func (s *GNNService) RunAnalysis(ctx context.Context, graphID string) (*GraphAnalysisResult, error) {
	s.mu.RLock()
	graph, exists := s.graphs[graphID]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("graph %s not found", graphID)
	}

	centrality := graph.CalculateCentrality()
	for nodeID, cent := range centrality {
		if node, ok := graph.Nodes[nodeID]; ok {
			node.Centrality = cent
		}
	}

	communities := s.gnn.RunCommunityDetection()

	s.gnn.propagate()

	anomalousNodes := s.gnn.DetectAnomalousNodes(ctx)

	anomalousCommunities := s.gnn.GetAnomalousCommunities()

	result := &GraphAnalysisResult{
		GraphID:              graphID,
		TotalNodes:           len(graph.Nodes),
		TotalEdges:           len(graph.Edges),
		CommunityCount:       len(communities),
		AnomalousNodeCount:   len(anomalousNodes),
		AnomalousCommunityCount: len(anomalousCommunities),
		CentralityMetrics:    centrality,
		Communities:          communities,
		AnomalousNodes:       anomalousNodes,
		AnomalousCommunities: anomalousCommunities,
		ProcessedAt:         time.Now(),
	}

	return result, nil
}

func (s *GNNService) AnalyzeUser(ctx context.Context, graphID, userID string) (*UserRelationshipAnalysis, error) {
	return s.gnn.AnalyzeUserRelationships(ctx, userID)
}

func (s *GNNService) GetGraphStats(graphID string) map[string]interface{} {
	s.mu.RLock()
	graph, exists := s.graphs[graphID]
	s.mu.RUnlock()

	if !exists {
		return nil
	}

	stats := map[string]interface{}{
		"total_nodes": len(graph.Nodes),
		"total_edges": len(graph.Edges),
		"communities": len(s.gnn.communities),
	}

	return stats
}

type GraphAnalysisResult struct {
	GraphID                   string
	TotalNodes                int
	TotalEdges                int
	CommunityCount            int
	AnomalousNodeCount        int
	AnomalousCommunityCount   int
	CentralityMetrics         map[string]float64
	Communities               []*Community
	AnomalousNodes            []*GraphNode
	AnomalousCommunities      []*Community
	ProcessedAt               time.Time
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	dotProd := 0.0
	normA := 0.0
	normB := 0.0

	for i := 0; i < len(a); i++ {
		dotProd += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	normASqrt := math.Sqrt(normA)
	normBSqrt := math.Sqrt(normB)

	if normASqrt == 0 || normBSqrt == 0 {
		return 0.0
	}

	return dotProd / (normASqrt * normBSqrt)
}
