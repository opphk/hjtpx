package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var (
	ruleCombinator     = service.NewRuleCombinator()
	ruleSandbox        *service.RuleSandbox
	ruleVersionManager *service.RuleVersionManager
)

func init() {
	ruleSandbox = service.NewRuleSandbox(ruleCombinator, service.NewEnhancedRuleEngine())
	ruleVersionManager = service.NewRuleVersionManager(ruleCombinator)
	ruleVersionManager.LoadAllVersionsFromDB()
}

// ========== Rule Combinator API ==========

type CombinedRuleRequest struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	RootGroup   *service.RuleGroup `json:"root_group"`
	Weight      float64            `json:"weight"`
	Severity    float64            `json:"severity"`
	Enabled     bool               `json:"enabled"`
}

func CreateCombinedRule(c *gin.Context) {
	var req CombinedRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	rule := &service.CombinedRule{
		ID:          req.ID,
		Name:        req.Name,
		Description: req.Description,
		RootGroup:   req.RootGroup,
		Weight:      req.Weight,
		Severity:    req.Severity,
		Enabled:     req.Enabled,
	}

	if err := ruleCombinator.AddCombinedRule(rule); err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"id":   rule.ID,
		"name": rule.Name,
	})
}

func GetCombinedRule(c *gin.Context) {
	ruleID := c.Param("ruleID")
	rule, err := ruleCombinator.GetCombinedRule(ruleID)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}
	response.Success(c, rule)
}

func UpdateCombinedRule(c *gin.Context) {
	ruleID := c.Param("ruleID")
	var req CombinedRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	rule := &service.CombinedRule{
		ID:          ruleID,
		Name:        req.Name,
		Description: req.Description,
		RootGroup:   req.RootGroup,
		Weight:      req.Weight,
		Severity:    req.Severity,
		Enabled:     req.Enabled,
	}

	if err := ruleCombinator.AddCombinedRule(rule); err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "规则更新成功"})
}

func DeleteCombinedRule(c *gin.Context) {
	ruleID := c.Param("ruleID")
	if err := ruleCombinator.RemoveCombinedRule(ruleID); err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, gin.H{"message": "规则删除成功"})
}

func ListCombinedRules(c *gin.Context) {
	rules := ruleCombinator.GetAllCombinedRules()
	response.Success(c, rules)
}

func EvaluateRule(c *gin.Context) {
	ruleID := c.Param("ruleID")

	var inputData map[string]interface{}
	if err := c.ShouldBindJSON(&inputData); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	result, err := ruleCombinator.EvaluateRule(ruleID, inputData)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"rule_id": ruleID,
		"result":  result,
	})
}

func ValidateRule(c *gin.Context) {
	var req CombinedRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	rule := &service.CombinedRule{
		ID:        req.ID,
		Name:      req.Name,
		RootGroup: req.RootGroup,
		Weight:    req.Weight,
		Severity:  req.Severity,
	}

	errors := ruleCombinator.ValidateRule(rule)
	response.Success(c, gin.H{
		"valid":  len(errors) == 0,
		"errors": errors,
	})
}

func ExportRule(c *gin.Context) {
	ruleID := c.Param("ruleID")
	data, err := ruleCombinator.ExportRuleToJSON(ruleID)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, gin.H{
		"data": data,
	})
}

func ImportRule(c *gin.Context) {
	var req struct {
		Data string `json:"data"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	if err := ruleCombinator.ImportRuleFromJSON(req.Data); err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "规则导入成功"})
}

// ========== Rule Sandbox API ==========

type RunTestRequest struct {
	RuleID    string                 `json:"rule_id"`
	InputData map[string]interface{} `json:"input_data"`
}

func RunRuleTest(c *gin.Context) {
	var req RunTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	result, err := ruleSandbox.RunTest(req.RuleID, req.InputData)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, result)
}

func RunAllRulesTest(c *gin.Context) {
	var req struct {
		InputData map[string]interface{} `json:"input_data"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	results, err := ruleSandbox.RunAllRulesTest(req.InputData)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, results)
}

type CreateTestCaseRequest struct {
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	InputData      map[string]interface{} `json:"input_data"`
	ExpectedResult bool                   `json:"expected_result"`
	ExpectedScore  float64                `json:"expected_score"`
	Enabled        bool                   `json:"enabled"`
}

func CreateTestCase(c *gin.Context) {
	var req CreateTestCaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	testCase := &service.TestCase{
		Name:           req.Name,
		Description:    req.Description,
		InputData:      req.InputData,
		ExpectedResult: req.ExpectedResult,
		ExpectedScore:  req.ExpectedScore,
		Enabled:        req.Enabled,
	}

	if err := ruleSandbox.AddTestCase(testCase); err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"id":   testCase.ID,
		"name": testCase.Name,
	})
}

func RunTestCase(c *gin.Context) {
	testCaseID := c.Param("testCaseID")
	result, err := ruleSandbox.RunTestCase(testCaseID)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, result)
}

func RunAllTestCases(c *gin.Context) {
	results, err := ruleSandbox.RunAllTestCases()
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, results)
}

func GetTestCases(c *gin.Context) {
	testCases := ruleSandbox.GetAllTestCases()
	response.Success(c, testCases)
}

func DeleteTestCase(c *gin.Context) {
	testCaseID := c.Param("testCaseID")
	if err := ruleSandbox.RemoveTestCase(testCaseID); err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, gin.H{"message": "测试用例删除成功"})
}

func GetTestHistory(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 20
	}

	history := ruleSandbox.GetTestHistory(limit)
	response.Success(c, history)
}

func ClearTestHistory(c *gin.Context) {
	ruleSandbox.ClearTestHistory()
	response.Success(c, gin.H{"message": "测试历史已清空"})
}

func GetSampleData(c *gin.Context) {
	dataType := c.DefaultQuery("type", "human")
	data := ruleSandbox.GenerateSampleData(dataType)
	response.Success(c, data)
}

func ExportTestCases(c *gin.Context) {
	data, err := ruleSandbox.ExportTestCases()
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, gin.H{"data": data})
}

func ImportTestCases(c *gin.Context) {
	var req struct {
		Data string `json:"data"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	if err := ruleSandbox.ImportTestCases(req.Data); err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "测试用例导入成功"})
}

// ========== Rule Version Manager API ==========

type CreateVersionRequest struct {
	ChangeType  string `json:"change_type"`
	Description string `json:"description"`
	Operator    string `json:"operator"`
}

func CreateVersion(c *gin.Context) {
	var req CreateVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	version, err := ruleVersionManager.CreateVersion(
		service.VersionChangeType(req.ChangeType),
		req.Description,
		req.Operator,
	)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, version)
}

func GetVersion(c *gin.Context) {
	version := c.Param("version")
	v, err := ruleVersionManager.GetVersion(version)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, v)
}

func ListVersions(c *gin.Context) {
	versions := ruleVersionManager.GetAllVersions()
	response.Success(c, versions)
}

func RollbackVersion(c *gin.Context) {
	version := c.Param("version")

	var req struct {
		Operator string `json:"operator"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	if err := ruleVersionManager.RollbackToVersion(version, req.Operator); err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "已成功回滚到版本 " + version})
}

func CompareVersions(c *gin.Context) {
	version := c.Param("version")

	var req struct {
		CompareWith string `json:"compare_with"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	if req.CompareWith == "" {
		current, err := ruleVersionManager.GetCurrentVersion()
		if err != nil {
			response.InternalServerError(c, err.Error())
			return
		}
		req.CompareWith = current.Version
	}

	diff, err := ruleVersionManager.CompareVersions(req.CompareWith, version)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, diff)
}

func ExportVersion(c *gin.Context) {
	version := c.Param("version")
	data, err := ruleVersionManager.ExportVersion(version)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, gin.H{"data": data})
}

func ImportVersion(c *gin.Context) {
	var req struct {
		Data     string `json:"data"`
		Operator string `json:"operator"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	version, err := ruleVersionManager.ImportVersion(req.Data, req.Operator)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, version)
}

func GetCurrentVersion(c *gin.Context) {
	version, err := ruleVersionManager.GetCurrentVersion()
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, version)
}

func DeleteVersion(c *gin.Context) {
	version := c.Param("version")
	if err := ruleVersionManager.DeleteVersion(version); err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, gin.H{"message": "版本删除成功"})
}

// RegisterRiskRulesRoutes registers all risk rules related routes
func RegisterRiskRulesRoutes(r *gin.RouterGroup) {
	// Rule Combinator Routes
	r.POST("/rules/combined", CreateCombinedRule)
	r.GET("/rules/combined", ListCombinedRules)
	r.GET("/rules/combined/:ruleID", GetCombinedRule)
	r.PUT("/rules/combined/:ruleID", UpdateCombinedRule)
	r.DELETE("/rules/combined/:ruleID", DeleteCombinedRule)
	r.POST("/rules/combined/:ruleID/evaluate", EvaluateRule)
	r.POST("/rules/combined/:ruleID/validate", ValidateRule)
	r.GET("/rules/combined/:ruleID/export", ExportRule)
	r.POST("/rules/combined/import", ImportRule)

	// Rule Sandbox Routes
	r.POST("/rules/sandbox/test", RunRuleTest)
	r.POST("/rules/sandbox/test-all", RunAllRulesTest)
	r.GET("/rules/sandbox/history", GetTestHistory)
	r.DELETE("/rules/sandbox/history", ClearTestHistory)
	r.GET("/rules/sandbox/sample-data", GetSampleData)

	r.POST("/rules/sandbox/test-cases", CreateTestCase)
	r.GET("/rules/sandbox/test-cases", GetTestCases)
	r.POST("/rules/sandbox/test-cases/:testCaseID/run", RunTestCase)
	r.POST("/rules/sandbox/test-cases/run-all", RunAllTestCases)
	r.DELETE("/rules/sandbox/test-cases/:testCaseID", DeleteTestCase)
	r.GET("/rules/sandbox/test-cases/export", ExportTestCases)
	r.POST("/rules/sandbox/test-cases/import", ImportTestCases)

	// Rule Version Routes
	r.POST("/rules/versions", CreateVersion)
	r.GET("/rules/versions", ListVersions)
	r.GET("/rules/versions/current", GetCurrentVersion)
	r.GET("/rules/versions/:version", GetVersion)
	r.DELETE("/rules/versions/:version", DeleteVersion)
	r.POST("/rules/versions/:version/rollback", RollbackVersion)
	r.POST("/rules/versions/:version/compare", CompareVersions)
	r.GET("/rules/versions/:version/export", ExportVersion)
	r.POST("/rules/versions/import", ImportVersion)
}
