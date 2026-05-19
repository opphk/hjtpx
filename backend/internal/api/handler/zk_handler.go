package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hjtpx/hjtpx/internal/crypto"
)

type ZKHandler struct {
	zkService             *crypto.ZKSNARKService
	verifierService       *crypto.VerifierService
	privacyService        *crypto.PrivacyVerificationService
	privacyEngine         *crypto.PrivacyEngine
}

type GenerateProofRequest struct {
	Witness      map[string]interface{} `json:"witness"`
	PublicInputs map[string]interface{} `json:"public_inputs"`
	StatementType string               `json:"statement_type"`
	CurveType    string                `json:"curve_type"`
	UserID       string                `json:"user_id"`
	SessionID    string                `json:"session_id"`
}

type GenerateProofResponse struct {
	Success    bool   `json:"success"`
	ProofID    string `json:"proof_id"`
	ProofData  string `json:"proof_data"`
	PublicHash string `json:"public_hash"`
	CreatedAt  int64  `json:"created_at"`
	ExpiresAt  int64  `json:"expires_at"`
	Error      string `json:"error,omitempty"`
}

type VerifyProofRequest struct {
	Proof       string                 `json:"proof"`
	PublicInputs map[string]interface{} `json:"public_inputs"`
	Statement   string                 `json:"statement"`
	UserID      string                 `json:"user_id"`
	SessionID   string                 `json:"session_id"`
}

type VerifyProofResponse struct {
	Success     bool                   `json:"success"`
	Valid       bool                   `json:"valid"`
	VerifiedAt  int64                  `json:"verified_at"`
	Duration    int64                  `json:"duration_ms"`
	Score       float64                `json:"score,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

type PrivacyVerifyRequest struct {
	DataSubject   map[string]interface{} `json:"data_subject"`
	DataToVerify  map[string]interface{} `json:"data_to_verify"`
	StatementType string                  `json:"statement_type"`
	PrivacyLevel  string                  `json:"privacy_level"`
	UserID        string                  `json:"user_id"`
}

type PrivacyVerifyResponse struct {
	Success          bool                   `json:"success"`
	Valid            bool                   `json:"valid"`
	ProofValid       bool                   `json:"proof_valid"`
	ConsentValid     bool                   `json:"consent_valid"`
	PolicyValid      bool                   `json:"policy_valid"`
	PrivacyPreserved bool                   `json:"privacy_preserved"`
	Error            string                 `json:"error,omitempty"`
	Warnings         []string               `json:"warnings,omitempty"`
	VerifiedAt       int64                  `json:"verified_at"`
	Duration         int64                  `json:"duration_ms"`
}

type MaskDataRequest struct {
	Data     interface{} `json:"data"`
	DataType string      `json:"data_type"`
	Level    string      `json:"level"`
	UserID   string      `json:"user_id"`
}

type MaskDataResponse struct {
	Success      bool                   `json:"success"`
	OriginalData interface{}            `json:"original_data,omitempty"`
	MaskedData   interface{}            `json:"masked_data"`
	Tokens       []string               `json:"tokens,omitempty"`
	MaskingInfo  *MaskingInfoResponse   `json:"masking_info"`
	Error        string                 `json:"error,omitempty"`
}

type MaskingInfoResponse struct {
	Strategy  string `json:"strategy"`
	DataType  string `json:"data_type"`
	RuleID    string `json:"rule_id"`
	Level     string `json:"level"`
	Timestamp int64  `json:"timestamp"`
}

type BudgetCheckRequest struct {
	UserID    string `json:"user_id"`
	Operation string `json:"operation"`
	DataSize  int64  `json:"data_size"`
}

type BudgetCheckResponse struct {
	Success   bool           `json:"success"`
	Allowed   bool           `json:"allowed"`
	Remaining int64          `json:"remaining"`
	Used      int64          `json:"used"`
	Alert     *BudgetAlert   `json:"alert,omitempty"`
	ResetAt   int64          `json:"reset_at"`
	Error     string         `json:"error,omitempty"`
}

type BudgetAlert struct {
	AlertID      string `json:"alert_id"`
	AlertType    string `json:"alert_type"`
	Message      string `json:"message"`
	CurrentUsage float64 `json:"current_usage"`
	Threshold    float64 `json:"threshold"`
}

type StatsResponse struct {
	Success bool                    `json:"success"`
	ZKStats *crypto.VerificationStats `json:"zk_stats,omitempty"`
	PrivacyStats *crypto.PrivacyStats `json:"privacy_stats,omitempty"`
	EngineStats *crypto.EngineStats   `json:"engine_stats,omitempty"`
}

func NewZKHandler() *ZKHandler {
	zkService := crypto.NewZKSNARKService(crypto.CurveP256)
	circuit := zkService.CreateKnowledgeProofCircuit("predicate")
	zkService.Setup(circuit)

	verifierService := crypto.NewVerifierService(crypto.CurveP256, nil)
	privacyService := crypto.NewPrivacyVerificationService(nil)
	privacyEngine := crypto.NewPrivacyEngine(nil)

	return &ZKHandler{
		zkService:        zkService,
		verifierService:  verifierService,
		privacyService:   privacyService,
		privacyEngine:    privacyEngine,
	}
}

func GenerateProof(c *gin.Context) {
	var req GenerateProofRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, GenerateProofResponse{
			Success: false,
			Error:   "invalid request: " + err.Error(),
		})
		return
	}

	zkHandler := getZKHandler(c)

	curveType := crypto.CurveP256
	if req.CurveType != "" {
		switch req.CurveType {
		case "P256":
			curveType = crypto.CurveP256
		case "P384":
			curveType = crypto.CurveP384
		case "P521":
			curveType = crypto.CurveP521
		}
	}

	zkService := crypto.NewZKSNARKService(curveType)
	circuit := zkService.CreateKnowledgeProofCircuit("predicate")
	if err := zkService.Setup(circuit); err != nil {
		c.JSON(http.StatusInternalServerError, GenerateProofResponse{
			Success: false,
			Error:   "setup failed: " + err.Error(),
		})
		return
	}

	statementType := crypto.StatementKnowledge
	if req.StatementType != "" {
		switch req.StatementType {
		case "range_proof":
			statementType = crypto.StatementRangeProof
		case "set_membership":
			statementType = crypto.StatementSetMembership
		case "knowledge_proof":
			statementType = crypto.StatementKnowledge
		case "equality_proof":
			statementType = crypto.StatementEquality
		}
	}

	generator, _ := crypto.NewZKProofGenerator(curveType)
	witness := &crypto.Witness{
		SecretValues: []string{},
		WitnessType:  string(statementType),
	}

	for _, v := range req.Witness {
		witness.SecretValues = append(witness.SecretValues, formatValue(v))
	}

	publicInput := &crypto.PublicInput{
		Values:    req.PublicInputs,
		CurveType: curveType,
	}

	proof, err := generator.CreateProof(witness, publicInput, statementType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, GenerateProofResponse{
			Success: false,
			Error:   "proof generation failed: " + err.Error(),
		})
		return
	}

	proofData, err := proof.ToJSON()
	if err != nil {
		c.JSON(http.StatusInternalServerError, GenerateProofResponse{
			Success: false,
			Error:   "proof serialization failed: " + err.Error(),
		})
		return
	}

	snarkRequest := &crypto.SNARKProofRequest{
		Witness:      req.Witness,
		PublicInputs: req.PublicInputs,
		Protocol:     "G16",
	}

	snarkResponse, err := zkService.GenerateProof(snarkRequest)
	if err != nil {
		proofData = proofData + "|snark_error:" + err.Error()
	} else if snarkResponse != nil {
		proofData = proofData + "|snark:" + snarkResponse.PublicHash
	}

	_ = zkHandler

	c.JSON(http.StatusOK, GenerateProofResponse{
		Success:    true,
		ProofID:    uuid.New().String(),
		ProofData:  proofData,
		PublicHash: formatPublicHash(req.PublicInputs),
		CreatedAt:  time.Now().Unix(),
		ExpiresAt:  time.Now().Add(30 * time.Minute).Unix(),
	})
}

func VerifyProof(c *gin.Context) {
	var req VerifyProofRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, VerifyProofResponse{
			Success: false,
			Error:   "invalid request: " + err.Error(),
		})
		return
	}

	curveType := crypto.CurveP256
	statementType := crypto.StatementKnowledge

	verifier, err := crypto.NewZKProofVerifier(curveType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, VerifyProofResponse{
			Success: false,
			Error:   "verifier creation failed: " + err.Error(),
		})
		return
	}

	proof, err := crypto.ParseProofFromJSON(req.Proof)
	if err != nil {
		proof = &crypto.ZKProof{
			ProofData:    []byte(req.Proof),
			Statement:    statementType,
			CurveType:    curveType,
			CreatedAt:    time.Now().Unix(),
			ExpiresAt:    time.Now().Add(30 * time.Minute).Unix(),
		}
	}

	verificationReq := &crypto.VerificationRequest{
		Proof:        proof,
		PublicInputs: req.PublicInputs,
		SessionID:    req.SessionID,
	}

	result, err := verifier.Verify(verificationReq)
	if err != nil {
		c.JSON(http.StatusOK, VerifyProofResponse{
			Success:    false,
			Valid:      false,
			VerifiedAt: time.Now().Unix(),
			Error:      err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, VerifyProofResponse{
		Success:    true,
		Valid:      result.Valid,
		VerifiedAt: result.VerifiedAt.Unix(),
		Duration:   result.Duration,
		Score:      result.Score,
		Metadata: map[string]interface{}{
			"mode": result.Mode,
		},
	})
}

func PrivacyVerify(c *gin.Context) {
	var req PrivacyVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, PrivacyVerifyResponse{
			Success: false,
			Error:   "invalid request: " + err.Error(),
		})
		return
	}

	privacyLevel := crypto.PrivacyLevelMedium
	switch req.PrivacyLevel {
	case "none":
		privacyLevel = crypto.PrivacyLevelNone
	case "basic":
		privacyLevel = crypto.PrivacyLevelBasic
	case "medium":
		privacyLevel = crypto.PrivacyLevelMedium
	case "high":
		privacyLevel = crypto.PrivacyLevelHigh
	case "maximum":
		privacyLevel = crypto.PrivacyLevelMaximum
	}

	statementType := crypto.StatementKnowledge
	switch req.StatementType {
	case "range_proof":
		statementType = crypto.StatementRangeProof
	case "set_membership":
		statementType = crypto.StatementSetMembership
	case "knowledge_proof":
		statementType = crypto.StatementKnowledge
	case "equality_proof":
		statementType = crypto.StatementEquality
	}

	dataSubject := &crypto.DataSubject{
		SubjectID:    getStringValue(req.DataSubject, "subject_id", req.UserID),
		PrivacyLevel: privacyLevel,
		Consents: map[crypto.ConsentType]bool{
			crypto.ConsentDataCollection: true,
			crypto.ConsentDataProcessing: true,
		},
		CreatedAt: time.Now(),
	}

	privacyReq := &crypto.PrivacyVerificationRequest{
		RequestID:     uuid.New().String(),
		DataSubject:   dataSubject,
		DataToVerify:  req.DataToVerify,
		StatementType: statementType,
		PrivacyLevel:  privacyLevel,
		Timestamp:     time.Now().Unix(),
		SessionID:     req.UserID,
	}

	privacyService := crypto.NewPrivacyVerificationService(nil)
	result, err := privacyService.Verify(privacyReq)
	if err != nil {
		c.JSON(http.StatusOK, PrivacyVerifyResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, PrivacyVerifyResponse{
		Success:          true,
		Valid:            result.Valid,
		ProofValid:       result.ProofValid,
		ConsentValid:     result.ConsentValid,
		PolicyValid:      result.PolicyValid,
		PrivacyPreserved: result.PrivacyPreserved,
		Error:            result.Error,
		Warnings:         result.Warnings,
		VerifiedAt:       result.VerifiedAt.Unix(),
		Duration:         result.Duration,
	})
}

func MaskData(c *gin.Context) {
	var req MaskDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, MaskDataResponse{
			Success: false,
			Error:   "invalid request: " + err.Error(),
		})
		return
	}

	privacyLevel := crypto.PrivacyLevelMedium
	switch req.Level {
	case "none":
		privacyLevel = crypto.PrivacyLevelNone
	case "basic":
		privacyLevel = crypto.PrivacyLevelBasic
	case "medium":
		privacyLevel = crypto.PrivacyLevelMedium
	case "high":
		privacyLevel = crypto.PrivacyLevelHigh
	case "maximum":
		privacyLevel = crypto.PrivacyLevelMaximum
	}

	dataType, _ := crypto.ParseDataType(req.DataType)
	if dataType == "" {
		dataType = crypto.DataTypeCustom
	}

	privacyEngine := crypto.NewPrivacyEngine(nil)
	maskingReq := &crypto.DataMaskingRequest{
		Data:     req.Data,
		DataType: dataType,
		Level:    privacyLevel,
		UserID:   req.UserID,
	}

	response, err := privacyEngine.MaskData(maskingReq)
	if err != nil {
		c.JSON(http.StatusOK, MaskDataResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, MaskDataResponse{
		Success:      true,
		MaskedData:   response.MaskedData,
		Tokens:       response.Tokens,
		MaskingInfo: &MaskingInfoResponse{
			Strategy:  string(response.MaskingInfo.Strategy),
			DataType:  string(response.MaskingInfo.DataType),
			RuleID:    response.MaskingInfo.RuleID,
			Level:     response.MaskingInfo.Level.String(),
			Timestamp: response.MaskingInfo.Timestamp.Unix(),
		},
	})
}

func CheckBudget(c *gin.Context) {
	var req BudgetCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, BudgetCheckResponse{
			Success: false,
			Error:   "invalid request: " + err.Error(),
		})
		return
	}

	privacyEngine := crypto.NewPrivacyEngine(nil)
	budgetReq := &crypto.BudgetCheckRequest{
		UserID:    req.UserID,
		Operation: req.Operation,
		DataSize:  req.DataSize,
	}

	response, err := privacyEngine.CheckBudget(budgetReq)
	if err != nil {
		c.JSON(http.StatusOK, BudgetCheckResponse{
			Success: false,
			Allowed: false,
			Error:   err.Error(),
		})
		return
	}

	alert := (*BudgetAlert)(nil)
	if response.Alert != nil {
		alert = &BudgetAlert{
			AlertID:      response.Alert.AlertID,
			AlertType:    response.Alert.AlertType,
			Message:      response.Alert.Message,
			CurrentUsage: response.Alert.CurrentUsage,
			Threshold:    response.Alert.Threshold,
		}
	}

	c.JSON(http.StatusOK, BudgetCheckResponse{
		Success:   true,
		Allowed:   response.Allowed,
		Remaining: response.Remaining,
		Used:      response.Used,
		Alert:     alert,
		ResetAt:   response.ResetAt.Unix(),
	})
}

func GetStats(c *gin.Context) {
	verifierService := crypto.NewVerifierService(crypto.CurveP256, nil)
	privacyService := crypto.NewPrivacyVerificationService(nil)
	privacyEngine := crypto.NewPrivacyEngine(nil)

	c.JSON(http.StatusOK, StatsResponse{
		Success:        true,
		ZKStats:       verifierService.GetStats(),
		PrivacyStats:   privacyService.GetStats(),
		EngineStats:    privacyEngine.GetStats(),
	})
}

func BatchVerifyProofs(c *gin.Context) {
	var requests []VerifyProofRequest
	if err := c.ShouldBindJSON(&requests); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid request: " + err.Error(),
		})
		return
	}

	verifier, _ := crypto.NewZKProofVerifier(crypto.CurveP256)

	verificationRequests := make([]crypto.VerificationRequest, 0, len(requests))
	for _, req := range requests {
		proof, _ := crypto.ParseProofFromJSON(req.Proof)
		if proof == nil {
			proof = &crypto.ZKProof{
				ProofData:    []byte(req.Proof),
				Statement:    crypto.StatementKnowledge,
				CurveType:    crypto.CurveP256,
				CreatedAt:    time.Now().Unix(),
				ExpiresAt:    time.Now().Add(30 * time.Minute).Unix(),
			}
		}

		verificationRequests = append(verificationRequests, crypto.VerificationRequest{
			Proof:        proof,
			PublicInputs: req.PublicInputs,
			SessionID:    req.SessionID,
		})
	}

	batchReq := &crypto.BatchVerificationRequest{
		Proofs: verificationRequests,
		Mode:   crypto.ModeStandard,
	}

	batchResult, err := verifier.BatchVerify(batchReq)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"total_valid":   batchResult.TotalValid,
		"total_invalid": batchResult.TotalInvalid,
		"total_duration": batchResult.TotalDuration,
		"results":       batchResult.Results,
	})
}

func CreateConsent(c *gin.Context) {
	var consent struct {
		UserID      string `json:"user_id"`
		ConsentType string `json:"consent_type"`
		Granted     bool   `json:"granted"`
		IPAddress   string `json:"ip_address"`
	}

	if err := c.ShouldBindJSON(&consent); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid request: " + err.Error(),
		})
		return
	}

	consentRecord := &crypto.ConsentRecord{
		UserID:      consent.UserID,
		ConsentType: crypto.ConsentType(consent.ConsentType),
		Granted:     consent.Granted,
		GrantedAt:   time.Now(),
		Version:     "1.0",
		IPAddress:   consent.IPAddress,
	}

	privacyService := crypto.NewPrivacyVerificationService(nil)
	if err := privacyService.RecordConsent(consentRecord); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "consent recorded",
	})
}

func RevokeConsent(c *gin.Context) {
	var request struct {
		UserID       string `json:"user_id"`
		ConsentType string `json:"consent_type"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid request: " + err.Error(),
		})
		return
	}

	privacyService := crypto.NewPrivacyVerificationService(nil)
	if err := privacyService.RevokeConsent(request.UserID, crypto.ConsentType(request.ConsentType)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "consent revoked",
	})
}

func AddMaskingRule(c *gin.Context) {
	var rule struct {
		RuleID      string `json:"rule_id"`
		DataType    string `json:"data_type"`
		Strategy    string `json:"strategy"`
		VisibleHead int    `json:"visible_head"`
		VisibleTail int    `json:"visible_tail"`
		MaskChar    string `json:"mask_char"`
		Priority    int    `json:"priority"`
	}

	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid request: " + err.Error(),
		})
		return
	}

	dataType, _ := crypto.ParseDataType(rule.DataType)
	strategy, _ := crypto.ParseMaskingStrategy(rule.Strategy)

	maskChar := rule.MaskChar
	if maskChar == "" {
		maskChar = "*"
	}

	maskingRule := &crypto.MaskingRule{
		RuleID:      rule.RuleID,
		DataType:    dataType,
		Strategy:    strategy,
		VisibleHead: rule.VisibleHead,
		VisibleTail: rule.VisibleTail,
		MaskChar:    maskChar,
		Priority:    rule.Priority,
	}

	privacyEngine := crypto.NewPrivacyEngine(nil)
	if err := privacyEngine.AddMaskingRule(maskingRule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "masking rule added",
		"rule_id": rule.RuleID,
	})
}

func ListMaskingRules(c *gin.Context) {
	privacyEngine := crypto.NewPrivacyEngine(nil)
	rules := privacyEngine.ListMaskingRules()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"rules":   rules,
	})
}

func getZKHandler(c *gin.Context) *ZKHandler {
	if handler, exists := c.Get("zk_handler"); exists {
		return handler.(*ZKHandler)
	}
	return NewZKHandler()
}

func formatValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return formatFloat(val)
	case int:
		return formatInt(val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func formatInt(i int) string {
	return strconv.Itoa(i)
}

func formatPublicHash(inputs map[string]interface{}) string {
	if inputs == nil || len(inputs) == 0 {
		return ""
	}
	result := ""
	for k, v := range inputs {
		result += k + "=" + formatValue(v) + ";"
	}
	return result
}

func getStringValue(m map[string]interface{}, key, fallback string) string {
	if m == nil {
		return fallback
	}
	if val, ok := m[key].(string); ok {
		return val
	}
	return fallback
}
