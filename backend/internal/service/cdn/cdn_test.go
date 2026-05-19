package cdn

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCDNService_InitializeDefaultRegions(t *testing.T) {
	service := NewCDNService(nil)
	err := service.InitializeDefaultRegions()
	assert.NoError(t, err)
	assert.Equal(t, 10, len(service.ListRegions()))
}

func TestCDNService_GetRegion(t *testing.T) {
	service := NewCDNService(nil)
	service.InitializeDefaultRegions()

	region, err := service.GetRegion("ap-east-1")
	assert.NoError(t, err)
	assert.Equal(t, "亚太东部", region.Name)

	_, err = service.GetRegion("invalid-region")
	assert.Error(t, err)
	assert.Equal(t, ErrRegionNotFound, err)
}

func TestCDNService_AddRemoveRegion(t *testing.T) {
	service := NewCDNService(nil)

	region := &Region{
		ID:        "test-region",
		Name:      "测试区域",
		Code:      "TS",
		Continent: "Test",
		Enabled:   true,
	}

	err := service.AddRegion(region)
	assert.NoError(t, err)

	regions := service.ListRegions()
	assert.Equal(t, 1, len(regions))

	err = service.DeleteRegion("test-region")
	assert.NoError(t, err)

	regions = service.ListRegions()
	assert.Equal(t, 0, len(regions))
}

func TestCDNService_RegisterNode(t *testing.T) {
	service := NewCDNService(nil)
	service.InitializeDefaultRegions()

	node := NewEdgeNode("node-1", "测试节点", "ap-east-1", "192.168.1.1", "node1.example.com", 8080)
	err := service.RegisterNode(node)
	assert.NoError(t, err)

	nodes := service.ListNodes("")
	assert.Equal(t, 1, len(nodes))

	nodeFromService, err := service.GetNode("node-1")
	assert.NoError(t, err)
	assert.Equal(t, "node-1", nodeFromService.ID)
}

func TestCDNService_GetRegionStats(t *testing.T) {
	service := NewCDNService(nil)
	service.InitializeDefaultRegions()

	node := NewEdgeNode("node-1", "测试节点", "ap-east-1", "192.168.1.1", "node1.example.com", 8080)
	service.RegisterNode(node)

	stats, err := service.GetRegionStats("ap-east-1")
	assert.NoError(t, err)
	assert.Equal(t, "ap-east-1", stats.RegionID)
	assert.Equal(t, 1, stats.NodeCount)
}

func TestEdgeNode_HealthCheck(t *testing.T) {
	node := NewEdgeNode("test-node", "Test Node", "ap-east-1", "127.0.0.1", "localhost", 9999)
	result := node.HealthCheck()
	assert.False(t, result)
	assert.False(t, node.IsHealthy)
	assert.Greater(t, node.LatencyMs, float64(0))
}

func TestEdgeNode_ExecuteFunction(t *testing.T) {
	node := NewEdgeNode("test-node", "Test Node", "ap-east-1", "127.0.0.1", "localhost", 8080)

	function := EdgeFunction{
		Name:      "test-func",
		Code:      "return 'hello'",
		Runtime:   "javascript",
		Enabled:   true,
	}
	node.DeployFunction(function)

	result, err := node.ExecuteFunction(context.Background(), "test-func", map[string]interface{}{"key": "value"})
	assert.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "test-func", result.FunctionName)
}

func TestSmartRouter_GetClientLocation(t *testing.T) {
	router := NewSmartRouter(nil)

	location, err := router.GetClientLocation("127.0.0.1")
	assert.NoError(t, err)
	assert.Equal(t, "Local", location.Country)

	location, err = router.GetClientLocation("114.114.114.114")
	assert.NoError(t, err)
	assert.Equal(t, "ap-east-1", location.Region)
}

func TestSmartRouter_GeoIPLookup(t *testing.T) {
	router := NewSmartRouter(nil)

	region := router.geoIPLookup("114.114.114.114")
	assert.Equal(t, "ap-east-1", region)

	region = router.geoIPLookup("3.0.0.1")
	assert.Equal(t, "ap-south-1", region)

	region = router.geoIPLookup("8.8.8.8")
	assert.Equal(t, "us-west-1", region)

	region = router.geoIPLookup("91.189.92.10")
	assert.Equal(t, "eu-west-1", region)

	region = router.geoIPLookup("141.101.198.205")
	assert.Equal(t, "ap-northeast-1", region)

	region = router.geoIPLookup("3.1.0.1")
	assert.Equal(t, "us-east-1", region)
}

func TestSmartRouter_CalculateDistanceScore(t *testing.T) {
	router := NewSmartRouter(nil)

	location := &GeoLocation{
		IP:        "114.114.114.114",
		Country:   "China",
		Region:    "ap-east-1",
		Latitude:  39.9042,
		Longitude: 116.4074,
	}

	score := router.calculateDistanceScore("ap-east-1", location)
	assert.Greater(t, score, float64(0.5))

	score = router.calculateDistanceScore("us-east-1", location)
	assert.Less(t, score, float64(0.6))
}

func TestStaticAssetAccelerator_GetContentType(t *testing.T) {
	accelerator := NewStaticAssetAccelerator(nil)

	assert.Equal(t, "text/css", accelerator.getContentType("style.css"))
	assert.Equal(t, "application/javascript", accelerator.getContentType("app.js"))
	assert.Equal(t, "image/jpeg", accelerator.getContentType("photo.jpg"))
	assert.Equal(t, "image/png", accelerator.getContentType("icon.png"))
	assert.Equal(t, "text/html", accelerator.getContentType("index.html"))
	assert.Equal(t, "application/json", accelerator.getContentType("data.json"))
	assert.Equal(t, "application/octet-stream", accelerator.getContentType("unknown.bin"))
}

func TestStaticAssetAccelerator_MinifyCSS(t *testing.T) {
	accelerator := NewStaticAssetAccelerator(nil)

	input := `/* comment */
body {
  color: red;
  font-size: 12px;
}`

	result, size := accelerator.minifyCSS([]byte(input))
	assert.Contains(t, string(result), "body")
	assert.Contains(t, string(result), "color: red")
	assert.NotContains(t, string(result), "/* comment */")
	assert.Less(t, size, int64(len(input)))
}

func TestGenerateETag(t *testing.T) {
	content := []byte("hello world")
	etag := generateETag(content)

	assert.Len(t, etag, 64)

	content2 := []byte("hello world")
	etag2 := generateETag(content2)
	assert.Equal(t, etag, etag2)

	content3 := []byte("hello world!")
	etag3 := generateETag(content3)
	assert.NotEqual(t, etag, etag3)
}

func TestEdgeNodeManager_GetLeastLoadedNode(t *testing.T) {
	manager := NewEdgeNodeManager()
	defer manager.Stop()

	node1 := NewEdgeNode("node1", "Node 1", "ap-east-1", "192.168.1.1", "node1.local", 8080)
	node2 := NewEdgeNode("node2", "Node 2", "ap-east-1", "192.168.1.2", "node2.local", 8080)

	node1.UpdateLoad(100)
	node2.UpdateLoad(50)

	manager.RegisterNode(node1)
	manager.RegisterNode(node2)

	node, err := manager.GetLeastLoadedNode("ap-east-1")
	assert.NoError(t, err)
	assert.Equal(t, "node2", node.ID)
}

func TestEdgeNodeManager_GetFastestNode(t *testing.T) {
	manager := NewEdgeNodeManager()
	defer manager.Stop()

	node1 := NewEdgeNode("node1", "Node 1", "ap-east-1", "192.168.1.1", "node1.local", 8080)
	node2 := NewEdgeNode("node2", "Node 2", "ap-east-1", "192.168.1.2", "node2.local", 8080)

	node1.LatencyMs = 50
	node2.LatencyMs = 25

	manager.RegisterNode(node1)
	manager.RegisterNode(node2)

	node, err := manager.GetFastestNode("ap-east-1")
	assert.NoError(t, err)
	assert.Equal(t, "node2", node.ID)
}

func TestSmartRouter_HaversineDistance(t *testing.T) {
	router := NewSmartRouter(nil)

	distance := router.haversineDistance(39.9042, 116.4074, 39.9042, 116.4074)
	assert.Equal(t, float64(0), distance)

	distance = router.haversineDistance(0, 0, 0, 180)
	assert.InDelta(t, 20015, distance, 100)
}