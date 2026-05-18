package service

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
	"gorm.io/gorm"
)

// VersionChangeType 版本变更类型
type VersionChangeType string

const (
	ChangeTypeCreate VersionChangeType = "create"
	ChangeTypeUpdate VersionChangeType = "update"
	ChangeTypeDelete VersionChangeType = "delete"
	ChangeTypeRollback VersionChangeType = "rollback"
	ChangeTypeOptimize VersionChangeType = "optimize"
)

// RuleVersion 规则版本
type RuleVersion struct {
	Version     string            `json:"version"`
	ChangeType  VersionChangeType `json:"change_type"`
	Description string            `json:"description"`
	Operator    string            `json:"operator"`
	CreatedAt   time.Time         `json:"created_at"`
	Rules       []*CombinedRule   `json:"rules"`
	IsCurrent   bool              `json:"is_current"`
	RuleCount   int               `json:"rule_count"`
}

// VersionDiff 版本差异
type VersionDiff struct {
	VersionFrom string              `json:"version_from"`
	VersionTo   string              `json:"version_to"`
	AddedRules  []*CombinedRule     `json:"added_rules"`
	RemovedRules []string           `json:"removed_rules"`
	UpdatedRules []RuleUpdateDetail `json:"updated_rules"`
}

// RuleUpdateDetail 规则更新详情
type RuleUpdateDetail struct {
	RuleID      string                 `json:"rule_id"`
	RuleName    string                 `json:"rule_name"`
	OldVersion  *CombinedRule         `json:"old_version"`
	NewVersion  *CombinedRule         `json:"new_version"`
	Changes     map[string]interface{} `json:"changes"`
}

// RuleVersionManager 规则版本管理器
type RuleVersionManager struct {
	versions     map[string]*RuleVersion
	combinator   *RuleCombinator
	db           *gorm.DB
	mu           sync.RWMutex
	maxVersions  int
}

func NewRuleVersionManager(combinator *RuleCombinator) *RuleVersionManager {
	return &RuleVersionManager{
		versions:    make(map[string]*RuleVersion),
		combinator:  combinator,
		db:          database.DB,
		maxVersions: 50,
	}
}

// CreateVersion 创建版本快照
func (rvm *RuleVersionManager) CreateVersion(changeType VersionChangeType, description string, operator string) (*RuleVersion, error) {
	rvm.mu.Lock()
	defer rvm.mu.Unlock()

	version := rvm.generateVersionNumber()
	rules := rvm.combinator.GetAllCombinedRules()

	// 深拷贝规则
	rulesCopy := make([]*CombinedRule, 0, len(rules))
	for _, rule := range rules {
		rulesCopy = append(rulesCopy, copyRule(rule))
	}

	newVersion := &RuleVersion{
		Version:     version,
		ChangeType:  changeType,
		Description: description,
		Operator:    operator,
		CreatedAt:   time.Now(),
		Rules:       rulesCopy,
		IsCurrent:   true,
		RuleCount:   len(rulesCopy),
	}

	// 更新当前版本标记
	for _, v := range rvm.versions {
		v.IsCurrent = false
	}

	rvm.versions[version] = newVersion

	// 清理旧版本
	if len(rvm.versions) > rvm.maxVersions {
		rvm.cleanupOldVersions()
	}

	// 保存到数据库
	if err := rvm.saveVersionToDB(newVersion); err != nil {
		return nil, err
	}

	return newVersion, nil
}

// GetVersion 获取版本
func (rvm *RuleVersionManager) GetVersion(version string) (*RuleVersion, error) {
	rvm.mu.RLock()
	defer rvm.mu.RUnlock()

	v, exists := rvm.versions[version]
	if !exists {
		// 尝试从数据库加载
		v, err := rvm.loadVersionFromDB(version)
		if err != nil {
			return nil, fmt.Errorf("版本不存在: %s", version)
		}
		rvm.versions[version] = v
		return v, nil
	}
	return v, nil
}

// GetAllVersions 获取所有版本
func (rvm *RuleVersionManager) GetAllVersions() []*RuleVersion {
	rvm.mu.RLock()
	defer rvm.mu.RUnlock()

	versions := make([]*RuleVersion, 0, len(rvm.versions))
	for _, v := range rvm.versions {
		versions = append(versions, v)
	}

	// 按时间排序（最新的在前）
	for i := len(versions) - 1; i > 0; i-- {
		for j := 0; j < i; j++ {
			if versions[j].CreatedAt.Before(versions[j+1].CreatedAt) {
				versions[j], versions[j+1] = versions[j+1], versions[j]
			}
		}
	}

	return versions
}

// RollbackToVersion 回滚到指定版本
func (rvm *RuleVersionManager) RollbackToVersion(version string, operator string) error {
	rvm.mu.Lock()
	defer rvm.mu.Unlock()

	targetVersion, exists := rvm.versions[version]
	if !exists {
		// 尝试从数据库加载
		var err error
		targetVersion, err = rvm.loadVersionFromDB(version)
		if err != nil {
			return fmt.Errorf("版本不存在: %s", version)
		}
	}

	// 备份当前状态
	currentVersion := rvm.generateVersionNumber()
	currentRules := rvm.combinator.GetAllCombinedRules()
	rulesCopy := make([]*CombinedRule, 0, len(currentRules))
	for _, rule := range currentRules {
		rulesCopy = append(rulesCopy, copyRule(rule))
	}

	backupVersion := &RuleVersion{
		Version:     currentVersion,
		ChangeType:  ChangeTypeRollback,
		Description: fmt.Sprintf("回滚前备份，将回滚到版本 %s", version),
		Operator:    operator,
		CreatedAt:   time.Now(),
		Rules:       rulesCopy,
		IsCurrent:   false,
		RuleCount:   len(rulesCopy),
	}
	rvm.versions[currentVersion] = backupVersion

	// 更新当前版本标记
	for _, v := range rvm.versions {
		v.IsCurrent = false
	}

	// 恢复目标版本的规则
	for _, rule := range targetVersion.Rules {
		err := rvm.combinator.AddCombinedRule(copyRule(rule))
		if err != nil {
			return err
		}
	}

	// 更新目标版本为当前版本
	targetVersion.IsCurrent = true

	// 创建回滚记录
	rollbackVersion := &RuleVersion{
		Version:     rvm.generateVersionNumber(),
		ChangeType:  ChangeTypeRollback,
		Description: fmt.Sprintf("回滚到版本 %s", version),
		Operator:    operator,
		CreatedAt:   time.Now(),
		Rules:       copyRules(targetVersion.Rules),
		IsCurrent:   true,
		RuleCount:   len(targetVersion.Rules),
	}

	// 更新当前版本标记
	for _, v := range rvm.versions {
		v.IsCurrent = false
	}
	rvm.versions[rollbackVersion.Version] = rollbackVersion

	// 保存到数据库
	if err := rvm.saveVersionToDB(rollbackVersion); err != nil {
		return err
	}

	return nil
}

// CompareVersions 比较两个版本
func (rvm *RuleVersionManager) CompareVersions(fromVersion, toVersion string) (*VersionDiff, error) {
	from, err := rvm.GetVersion(fromVersion)
	if err != nil {
		return nil, err
	}

	to, err := rvm.GetVersion(toVersion)
	if err != nil {
		return nil, err
	}

	diff := &VersionDiff{
		VersionFrom:  fromVersion,
		VersionTo:    toVersion,
		AddedRules:   make([]*CombinedRule, 0),
		RemovedRules: make([]string, 0),
		UpdatedRules: make([]RuleUpdateDetail, 0),
	}

	// 构建规则映射
	fromRules := make(map[string]*CombinedRule)
	for _, rule := range from.Rules {
		fromRules[rule.ID] = rule
	}

	toRules := make(map[string]*CombinedRule)
	for _, rule := range to.Rules {
		toRules[rule.ID] = rule
	}

	// 查找新增的规则
	for id, rule := range toRules {
		if _, exists := fromRules[id]; !exists {
			diff.AddedRules = append(diff.AddedRules, rule)
		}
	}

	// 查找删除的规则
	for id := range fromRules {
		if _, exists := toRules[id]; !exists {
			diff.RemovedRules = append(diff.RemovedRules, id)
		}
	}

	// 查找更新的规则
	for id, fromRule := range fromRules {
		if toRule, exists := toRules[id]; exists {
			changes := compareRules(fromRule, toRule)
			if len(changes) > 0 {
				diff.UpdatedRules = append(diff.UpdatedRules, RuleUpdateDetail{
					RuleID:     id,
					RuleName:   fromRule.Name,
					OldVersion: fromRule,
					NewVersion: toRule,
					Changes:    changes,
				})
			}
		}
	}

	return diff, nil
}

// ExportVersion 导出版本配置
func (rvm *RuleVersionManager) ExportVersion(version string) (string, error) {
	v, err := rvm.GetVersion(version)
	if err != nil {
		return "", err
	}

	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ImportVersion 导入版本配置
func (rvm *RuleVersionManager) ImportVersion(jsonData string, operator string) (*RuleVersion, error) {
	var version RuleVersion
	if err := json.Unmarshal([]byte(jsonData), &version); err != nil {
		return nil, err
	}

	// 生成新的版本号
	version.Version = rvm.generateVersionNumber()
	version.CreatedAt = time.Now()
	version.Operator = operator
	version.ChangeType = ChangeTypeUpdate
	version.IsCurrent = true

	// 更新当前版本标记
	rvm.mu.Lock()
	for _, v := range rvm.versions {
		v.IsCurrent = false
	}
	rvm.versions[version.Version] = &version
	rvm.mu.Unlock()

	// 应用导入的规则
	for _, rule := range version.Rules {
		err := rvm.combinator.AddCombinedRule(copyRule(rule))
		if err != nil {
			return nil, err
		}
	}

	// 保存到数据库
	if err := rvm.saveVersionToDB(&version); err != nil {
		return nil, err
	}

	return &version, nil
}

// GetCurrentVersion 获取当前版本
func (rvm *RuleVersionManager) GetCurrentVersion() (*RuleVersion, error) {
	rvm.mu.RLock()
	defer rvm.mu.RUnlock()

	for _, v := range rvm.versions {
		if v.IsCurrent {
			return v, nil
		}
	}

	// 如果没有当前版本，创建一个初始版本
	return nil, fmt.Errorf("没有找到当前版本")
}

// DeleteVersion 删除版本（非当前版本）
func (rvm *RuleVersionManager) DeleteVersion(version string) error {
	rvm.mu.Lock()
	defer rvm.mu.Unlock()

	v, exists := rvm.versions[version]
	if !exists {
		return fmt.Errorf("版本不存在: %s", version)
	}

	if v.IsCurrent {
		return fmt.Errorf("不能删除当前版本")
	}

	delete(rvm.versions, version)

	// 从数据库删除
	if err := rvm.deleteVersionFromDB(version); err != nil {
		return err
	}

	return nil
}

// generateVersionNumber 生成版本号
func (rvm *RuleVersionManager) generateVersionNumber() string {
	return fmt.Sprintf("v%d.%d.%d.%d",
		time.Now().Year(),
		int(time.Now().Month()),
		time.Now().Day(),
		time.Now().Hour()*10000+time.Now().Minute()*100+time.Now().Second(),
	)
}

// cleanupOldVersions 清理旧版本
func (rvm *RuleVersionManager) cleanupOldVersions() {
	versions := make([]*RuleVersion, 0, len(rvm.versions))
	for _, v := range rvm.versions {
		versions = append(versions, v)
	}

	// 按时间排序（最老的在前）
	for i := len(versions) - 1; i > 0; i-- {
		for j := 0; j < i; j++ {
			if versions[j].CreatedAt.After(versions[j+1].CreatedAt) {
				versions[j], versions[j+1] = versions[j+1], versions[j]
			}
		}
	}

	// 删除最老的版本，保留maxVersions个
	for i := 0; i < len(versions)-rvm.maxVersions; i++ {
		if !versions[i].IsCurrent {
			delete(rvm.versions, versions[i].Version)
		}
	}
}

// copyRule 深拷贝规则
func copyRule(rule *CombinedRule) *CombinedRule {
	data, _ := json.Marshal(rule)
	var copied CombinedRule
	json.Unmarshal(data, &copied)
	return &copied
}

// copyRules 深拷贝规则列表
func copyRules(rules []*CombinedRule) []*CombinedRule {
	result := make([]*CombinedRule, 0, len(rules))
	for _, rule := range rules {
		result = append(result, copyRule(rule))
	}
	return result
}

// compareRules 比较两个规则的差异
func compareRules(oldRule, newRule *CombinedRule) map[string]interface{} {
	changes := make(map[string]interface{})

	if oldRule.Name != newRule.Name {
		changes["name"] = map[string]string{"old": oldRule.Name, "new": newRule.Name}
	}
	if oldRule.Description != newRule.Description {
		changes["description"] = map[string]string{"old": oldRule.Description, "new": newRule.Description}
	}
	if oldRule.Weight != newRule.Weight {
		changes["weight"] = map[string]float64{"old": oldRule.Weight, "new": newRule.Weight}
	}
	if oldRule.Severity != newRule.Severity {
		changes["severity"] = map[string]float64{"old": oldRule.Severity, "new": newRule.Severity}
	}
	if oldRule.Enabled != newRule.Enabled {
		changes["enabled"] = map[string]bool{"old": oldRule.Enabled, "new": newRule.Enabled}
	}

	return changes
}

// saveVersionToDB 保存版本到数据库
func (rvm *RuleVersionManager) saveVersionToDB(version *RuleVersion) error {
	rulesJSON, err := json.Marshal(version.Rules)
	if err != nil {
		return err
	}

	dbVersion := models.RiskRuleVersion{
		Version:     version.Version,
		ChangeType:  string(version.ChangeType),
		Description: version.Description,
		Operator:    version.Operator,
		RulesJSON:   string(rulesJSON),
		RuleCount:   version.RuleCount,
		IsCurrent:   version.IsCurrent,
	}

	return rvm.db.Create(&dbVersion).Error
}

// loadVersionFromDB 从数据库加载版本
func (rvm *RuleVersionManager) loadVersionFromDB(version string) (*RuleVersion, error) {
	var dbVersion models.RiskRuleVersion
	err := rvm.db.Where("version = ?", version).First(&dbVersion).Error
	if err != nil {
		return nil, err
	}

	var rules []*CombinedRule
	if err := json.Unmarshal([]byte(dbVersion.RulesJSON), &rules); err != nil {
		return nil, err
	}

	return &RuleVersion{
		Version:     dbVersion.Version,
		ChangeType:  VersionChangeType(dbVersion.ChangeType),
		Description: dbVersion.Description,
		Operator:    dbVersion.Operator,
		CreatedAt:   dbVersion.CreatedAt,
		Rules:       rules,
		IsCurrent:   dbVersion.IsCurrent,
		RuleCount:   dbVersion.RuleCount,
	}, nil
}

// deleteVersionFromDB 从数据库删除版本
func (rvm *RuleVersionManager) deleteVersionFromDB(version string) error {
	return rvm.db.Where("version = ?", version).Delete(&models.RiskRuleVersion{}).Error
}

// LoadAllVersionsFromDB 从数据库加载所有版本
func (rvm *RuleVersionManager) LoadAllVersionsFromDB() error {
	if rvm.db == nil {
		return nil // 数据库未初始化时跳过
	}
	var dbVersions []models.RiskRuleVersion
	err := rvm.db.Order("created_at DESC").Find(&dbVersions).Error
	if err != nil {
		return err
	}

	rvm.mu.Lock()
	defer rvm.mu.Unlock()

	for _, dbVersion := range dbVersions {
		var rules []*CombinedRule
		if err := json.Unmarshal([]byte(dbVersion.RulesJSON), &rules); err != nil {
			continue
		}

		rvm.versions[dbVersion.Version] = &RuleVersion{
			Version:     dbVersion.Version,
			ChangeType:  VersionChangeType(dbVersion.ChangeType),
			Description: dbVersion.Description,
			Operator:    dbVersion.Operator,
			CreatedAt:   dbVersion.CreatedAt,
			Rules:       rules,
			IsCurrent:   dbVersion.IsCurrent,
			RuleCount:   dbVersion.RuleCount,
		}
	}

	return nil
}