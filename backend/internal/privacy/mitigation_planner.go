package privacy

import (
	"sync"
	"time"
)

type MitigationPlanner struct {
	mitigations    map[string]*Mitigation
	riskTracker    *RiskCalculator
	implementationPlan *ImplementationPlan
	mu             sync.RWMutex
}

type Mitigation struct {
	ID             string
	Name           string
	Description    string
	TargetRiskID   string
	Effectiveness  float64
	Cost           float64
	ImplementationTime time.Duration
	Status         MitigationStatus
	AssignedTo     string
	StartDate      time.Time
	CompletionDate time.Time
	Milestones     []Milestone
	Dependencies   []string
}

type MitigationStatus int

const (
	MitigationStatusPending MitigationStatus = iota
	MitigationStatusPlanning
	MitigationStatusInProgress
	MitigationStatusCompleted
	MitigationStatusOnHold
	MitigationStatusCancelled
)

type Milestone struct {
	ID          string
	Name        string
	Description string
	DueDate     time.Time
	Status      MilestoneStatus
	CompletedDate time.Time
}

type MilestoneStatus int

const (
	MilestonePending MilestoneStatus = iota
	MilestoneInProgress
	MilestoneCompleted
	MilestoneDelayed
	MilestoneCancelled
)

type ImplementationPlan struct {
	ID            string
	Name          string
	Description   string
	StartDate     time.Time
	EndDate       time.Time
	Mitigations   []*Mitigation
	Resources     map[string]Resource
	Budget        float64
	SpentBudget   float64
	Status        PlanStatus
}

type PlanStatus int

const (
	PlanStatusDraft PlanStatus = iota
	PlanStatusActive
	PlanStatusCompleted
	PlanStatusCancelled
)

type Resource struct {
	Name        string
	Type        ResourceType
	Allocation  float64
	CostPerHour float64
}

type ResourceType int

const (
	ResourceHuman ResourceType = iota
	ResourceTechnology
	ResourceFinancial
	ResourceExternal
)

func NewMitigationPlanner(riskTracker *RiskCalculator) *MitigationPlanner {
	return &MitigationPlanner{
		mitigations:    make(map[string]*Mitigation),
		riskTracker:    riskTracker,
		implementationPlan: NewImplementationPlan("默认实施计划"),
	}
}

func (mp *MitigationPlanner) CreateMitigation(name, description, targetRiskID string, effectiveness, cost float64, implTime time.Duration) *Mitigation {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	mitigation := &Mitigation{
		ID:                 generateMitigationID(),
		Name:               name,
		Description:        description,
		TargetRiskID:       targetRiskID,
		Effectiveness:      effectiveness,
		Cost:               cost,
		ImplementationTime: implTime,
		Status:             MitigationStatusPending,
		Milestones:         make([]Milestone, 0),
		Dependencies:      make([]string, 0),
	}

	mp.mitigations[mitigation.ID] = mitigation
	return mitigation
}

func (mp *MitigationPlanner) AddMilestone(mitigationID string, name, description string, dueDate time.Time) *Milestone {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	mitigation := mp.mitigations[mitigationID]
	if mitigation == nil {
		return nil
	}

	milestone := Milestone{
		ID:          generateMiligationID(),
		Name:        name,
		Description: description,
		DueDate:     dueDate,
		Status:      MilestonePending,
	}

	mitigation.Milestones = append(mitigation.Milestones, milestone)
	return &mitigation.Milestones[len(mitigation.Milestones)-1]
}

func (mp *MitigationPlanner) AddDependency(mitigationID, dependsOnID string) {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	if mitigation, exists := mp.mitigations[mitigationID]; exists {
		mitigation.Dependencies = append(mitigation.Dependencies, dependsOnID)
	}
}

func (mp *MitigationPlanner) StartMitigation(mitigationID, assignedTo string) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	mitigation := mp.mitigations[mitigationID]
	if mitigation == nil {
		return ErrMitigationNotFound
	}

	if !mp.checkDependencies(mitigation) {
		return ErrDependenciesNotMet
	}

	mitigation.Status = MitigationStatusInProgress
	mitigation.AssignedTo = assignedTo
	mitigation.StartDate = time.Now()

	return nil
}

func (mp *MitigationPlanner) checkDependencies(mitigation *Mitigation) bool {
	for _, depID := range mitigation.Dependencies {
		dep := mp.mitigations[depID]
		if dep == nil || dep.Status != MitigationStatusCompleted {
			return false
		}
	}
	return true
}

func (mp *MitigationPlanner) CompleteMilestone(mitigationID, milestoneID string) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	mitigation := mp.mitigations[mitigationID]
	if mitigation == nil {
		return ErrMitigationNotFound
	}

	for i := range mitigation.Milestones {
		if mitigation.Milestones[i].ID == milestoneID {
			mitigation.Milestones[i].Status = MilestoneCompleted
			mitigation.Milestones[i].CompletedDate = time.Now()
			return nil
		}
	}

	return ErrMilestoneNotFound
}

func (mp *MitigationPlanner) CompleteMitigation(mitigationID string) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	mitigation := mp.mitigations[mitigationID]
	if mitigation == nil {
		return ErrMitigationNotFound
	}

	mitigation.Status = MitigationStatusCompleted
	mitigation.CompletionDate = time.Now()

	mp.implementationPlan.SpentBudget += mitigation.Cost

	mp.riskTracker.SetMitigation(mitigation.TargetRiskID, mitigation.Effectiveness)

	return nil
}

func (mp *MitigationPlanner) GetMitigation(mitigationID string) *Mitigation {
	mp.mu.RLock()
	defer mp.mu.RUnlock()
	return mp.mitigations[mitigationID]
}

func (mp *MitigationPlanner) GetAllMitigations() []*Mitigation {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	mitigations := make([]*Mitigation, 0, len(mp.mitigations))
	for _, m := range mp.mitigations {
		mitigations = append(mitigations, m)
	}
	return mitigations
}

func (mp *MitigationPlanner) GetMitigationsByStatus(status MitigationStatus) []*Mitigation {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	mitigations := make([]*Mitigation, 0)
	for _, m := range mp.mitigations {
		if m.Status == status {
			mitigations = append(mitigations, m)
		}
	}
	return mitigations
}

func (mp *MitigationPlanner) GetMitigationsByRisk(riskID string) []*Mitigation {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	mitigations := make([]*Mitigation, 0)
	for _, m := range mp.mitigations {
		if m.TargetRiskID == riskID {
			mitigations = append(mitigations, m)
		}
	}
	return mitigations
}

func (mp *MitigationPlanner) CancelMitigation(mitigationID string) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	mitigation := mp.mitigations[mitigationID]
	if mitigation == nil {
		return ErrMitigationNotFound
	}

	mitigation.Status = MitigationStatusCancelled
	return nil
}

func (mp *MitigationPlanner) PutOnHold(mitigationID string) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	mitigation := mp.mitigations[mitigationID]
	if mitigation == nil {
		return ErrMitigationNotFound
	}

	mitigation.Status = MitigationStatusOnHold
	return nil
}

func (mp *MitigationPlanner) ResumeMitigation(mitigationID string) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	mitigation := mp.mitigations[mitigationID]
	if mitigation == nil {
		return ErrMitigationNotFound
	}

	if !mp.checkDependencies(mitigation) {
		return ErrDependenciesNotMet
	}

	mitigation.Status = MitigationStatusInProgress
	return nil
}

func (mp *MitigationPlanner) GetImplementationPlan() *ImplementationPlan {
	mp.mu.RLock()
	defer mp.mu.RUnlock()
	return mp.implementationPlan
}

func (mp *MitigationPlanner) UpdatePlanBudget(spent float64) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.implementationPlan.SpentBudget = spent
}

func (mp *MitigationPlanner) GetProgress() PlanProgress {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	progress := PlanProgress{}

	for _, m := range mp.mitigations {
		progress.Total++

		switch m.Status {
		case MitigationStatusCompleted:
			progress.Completed++
		case MitigationStatusInProgress:
			progress.InProgress++
		case MitigationStatusPending:
			progress.Pending++
		}

		for _, ms := range m.Milestones {
			progress.TotalMilestones++
			if ms.Status == MilestoneCompleted {
				progress.CompletedMilestones++
			}
		}
	}

	if progress.Total > 0 {
		progress.PercentageComplete = float64(progress.Completed) / float64(progress.Total) * 100
	}

	if progress.TotalMilestones > 0 {
		progress.MilestonePercentageComplete = float64(progress.CompletedMilestones) / float64(progress.TotalMilestones) * 100
	}

	if mp.implementationPlan.Budget > 0 {
		progress.BudgetUsed = mp.implementationPlan.SpentBudget / mp.implementationPlan.Budget * 100
	}

	return progress
}

type PlanProgress struct {
	Total                     int
	Completed                 int
	InProgress                int
	Pending                   int
	PercentageComplete        float64
	TotalMilestones           int
	CompletedMilestones       int
	MilestonePercentageComplete float64
	BudgetUsed                float64
}

func NewImplementationPlan(name string) *ImplementationPlan {
	return &ImplementationPlan{
		ID:          generatePlanID(),
		Name:        name,
		StartDate:   time.Now(),
		Mitigations: make([]*Mitigation, 0),
		Resources:   make(map[string]Resource),
		Status:      PlanStatusDraft,
	}
}

func (ip *ImplementationPlan) AddMitigation(m *Mitigation) {
	ip.Mitigations = append(ip.Mitigations, m)
	ip.Budget += m.Cost
}

func (ip *ImplementationPlan) SetStatus(status PlanStatus) {
	ip.Status = status
}

func (ip *ImplementationPlan) AddResource(resource Resource) {
	ip.Resources[resource.Name] = resource
}

func (mp *MitigationPlanner) GenerateReport() MitigationReport {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	report := MitigationReport{
		GeneratedAt: time.Now(),
	}

	for _, m := range mp.mitigations {
		report.TotalMitigations++

		switch m.Status {
		case MitigationStatusCompleted:
			report.Completed++
		case MitigationStatusInProgress:
			report.InProgress++
		case MitigationStatusPending:
			report.Pending++
		case MitigationStatusOnHold:
			report.OnHold++
		}

		report.TotalCost += m.Cost
		if m.Status == MitigationStatusCompleted {
			report.SpentCost += m.Cost
		}

		report.TotalEffectiveness += m.Effectiveness

		for _, ms := range m.Milestones {
			report.TotalMilestones++
			if ms.Status == MilestoneCompleted {
				report.CompletedMilestones++
			}
		}
	}

	if report.TotalMitigations > 0 {
		report.CompletionRate = float64(report.Completed) / float64(report.TotalMitigations) * 100
	}

	if report.TotalMilestones > 0 {
		report.MilestoneCompletionRate = float64(report.CompletedMilestones) / float64(report.TotalMilestones) * 100
	}

	if report.TotalMitigations > 0 {
		report.AverageEffectiveness = report.TotalEffectiveness / float64(report.TotalMitigations)
	}

	return report
}

type MitigationReport struct {
	GeneratedAt              time.Time
	TotalMitigations         int
	Completed                int
	InProgress               int
	Pending                  int
	OnHold                   int
	CompletionRate           float64
	TotalMilestones          int
	CompletedMilestones      int
	MilestoneCompletionRate   float64
	TotalCost                float64
	SpentCost                float64
	TotalEffectiveness       float64
	AverageEffectiveness     float64
}

var ErrMitigationNotFound = &MitigationError{message: "mitigation not found"}
var ErrMilestoneNotFound = &MitigationError{message: "milestone not found"}
var ErrDependenciesNotMet = &MitigationError{message: "dependencies not met"}

type MitigationError struct {
	message string
}

func (e *MitigationError) Error() string {
	return e.message
}

func generateMitigationID() string {
	return "MIG-" + time.Now().Format("20060102150405.000000000")
}

func generateMiligationID() string {
	return "MLS-" + time.Now().Format("20060102150405.000")
}

func generatePlanID() string {
	return "PLAN-" + time.Now().Format("20060102150405")
}

type MitigationOptimizer struct {
	planner *MitigationPlanner
	mu      sync.RWMutex
}

func NewMitigationOptimizer(planner *MitigationPlanner) *MitigationOptimizer {
	return &MitigationOptimizer{
		planner: planner,
	}
}

func (mo *MitigationOptimizer) OptimizeForBudget(budget float64) []*Mitigation {
	mo.mu.RLock()
	allMitigations := mo.planner.GetAllMitigations()
	mo.mu.RUnlock()

	selected := make([]*Mitigation, 0)
	totalCost := 0.0

	effectiveness := make([]struct {
		mitigation *Mitigation
		ratio      float64
	}, len(allMitigations))

	for i, m := range allMitigations {
		if m.Cost > 0 {
			effectiveness[i].ratio = m.Effectiveness / m.Cost
		}
		effectiveness[i].mitigation = m
	}

	for i := 0; i < len(effectiveness)-1; i++ {
		for j := i + 1; j < len(effectiveness); j++ {
			if effectiveness[i].ratio < effectiveness[j].ratio {
				effectiveness[i], effectiveness[j] = effectiveness[j], effectiveness[i]
			}
		}
	}

	for _, e := range effectiveness {
		if totalCost+e.mitigation.Cost <= budget {
			selected = append(selected, e.mitigation)
			totalCost += e.mitigation.Cost
		}
	}

	return selected
}

func (mo *MitigationOptimizer) OptimizeForTime(deadline time.Time) []*Mitigation {
	mo.mu.RLock()
	allMitigations := mo.planner.GetAllMitigations()
	mo.mu.RUnlock()

	selected := make([]*Mitigation, 0)

	for _, m := range allMitigations {
		estimatedCompletion := m.StartDate.Add(m.ImplementationTime)
		if estimatedCompletion.Before(deadline) {
			selected = append(selected, m)
		}
	}

	return selected
}
