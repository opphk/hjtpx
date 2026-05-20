package privacy

import (
	"encoding/json"
	"sync"
	"time"
)

type FederatedLearningConfig struct {
	NumClients        int
	NumRounds         int
	LocalEpochs       int
	BatchSize         int
	LearningRate      float64
	ModelType         string
	PrivacyBudget     float64
	AggregationMethod string
	UseDP             bool
	DPConfig          DPConfig
}

type DPConfig struct {
	Epsilon     float64
	Delta       float64
	ClipNorm    float64
	NoiseType   NoiseType
	MaxGradNorm float64
}

type ModelParameters struct {
	Weights map[string][]float64
	Biases map[string][]float64
	Version int
	Round   int
}

type FederatedLearning struct {
	config      FederatedLearningConfig
	server      *FederatedServer
	clients     map[string]*FederatedClient
	model       *ModelParameters
	round       int
	isRunning   bool
	mu          sync.RWMutex
	eventLog    []FederatedEvent
}

type FederatedEvent struct {
	Round    int
	ClientID string
	Type     string
	Time     time.Time
	Data     map[string]interface{}
}

func NewFederatedLearning(config FederatedLearningConfig) *FederatedLearning {
	fl := &FederatedLearning{
		config:  config,
		clients: make(map[string]*FederatedClient),
		model: &ModelParameters{
			Weights: make(map[string][]float64),
			Biases:  make(map[string][]float64),
			Version: 0,
			Round:   0,
		},
		eventLog: make([]FederatedEvent, 0),
	}

	fl.server = NewFederatedServer(FederatedServerConfig{
		AggregationMethod: config.AggregationMethod,
		UseDP:            config.UseDP,
		DPConfig:         config.DPConfig,
	})

	return fl
}

func (fl *FederatedLearning) RegisterClient(clientID string, dataSize int) *FederatedClient {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	client := NewFederatedClient(FederatedClientConfig{
		ClientID:      clientID,
		DataSize:      dataSize,
		LocalEpochs:   fl.config.LocalEpochs,
		BatchSize:     fl.config.BatchSize,
		LearningRate:  fl.config.LearningRate,
		UseDP:         fl.config.UseDP,
		DPConfig:      fl.config.DPConfig,
		ModelType:     fl.config.ModelType,
	})

	fl.clients[clientID] = client
	fl.logEvent(fl.round, clientID, "registered", nil)
	return client
}

func (fl *FederatedLearning) UnregisterClient(clientID string) bool {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	if _, exists := fl.clients[clientID]; exists {
		delete(fl.clients, clientID)
		fl.logEvent(fl.round, clientID, "unregistered", nil)
		return true
	}
	return false
}

func (fl *FederatedLearning) StartTraining() error {
	fl.mu.Lock()
	if fl.isRunning {
		fl.mu.Unlock()
		return ErrTrainingAlreadyRunning
	}
	fl.isRunning = true
	fl.mu.Unlock()

	go fl.runTrainingLoop()
	return nil
}

func (fl *FederatedLearning) runTrainingLoop() {
	for fl.round < fl.config.NumRounds {
		if !fl.isRunning {
			break
		}

		fl.mu.RLock()
		clientIDs := fl.getActiveClientIDs()
		fl.mu.RUnlock()

		if len(clientIDs) == 0 {
			time.Sleep(time.Second)
			continue
		}

		selectedClients := fl.selectClients(clientIDs)

		fl.mu.RLock()
		modelCopy := fl.copyModel()
		fl.mu.RUnlock()

		clientUpdates := fl.distributeAndCollectUpdates(selectedClients, modelCopy)

		fl.mu.Lock()
		fl.server.AggregateUpdates(clientUpdates)
		fl.model = fl.server.GetGlobalModel()
		fl.round++
		fl.mu.Unlock()

		time.Sleep(100 * time.Millisecond)
	}

	fl.mu.Lock()
	fl.isRunning = false
	fl.mu.Unlock()
}

func (fl *FederatedLearning) selectClients(clientIDs []string) []string {
	numToSelect := int(float64(len(clientIDs)) * 0.5)
	if numToSelect < 1 {
		numToSelect = 1
	}

	selected := make([]string, numToSelect)
	for i := 0; i < numToSelect; i++ {
		selected[i] = clientIDs[i%len(clientIDs)]
	}
	return selected
}

func (fl *FederatedLearning) distributeAndCollectUpdates(clients []string, model *ModelParameters) []*ClientUpdate {
	updates := make([]*ClientUpdate, 0, len(clients))

	for _, clientID := range clients {
		fl.mu.RLock()
		client := fl.clients[clientID]
		fl.mu.RUnlock()

		if client == nil {
			continue
		}

		update := client.Train(model, fl.round)
		updates = append(updates, update)

		fl.logEvent(fl.round, clientID, "update_submitted", map[string]interface{}{
			"gradient_norm": update.GradientNorm,
			"data_size":     update.DataSize,
		})
	}

	return updates
}

func (fl *FederatedLearning) getActiveClientIDs() []string {
	ids := make([]string, 0, len(fl.clients))
	for id := range fl.clients {
		ids = append(ids, id)
	}
	return ids
}

func (fl *FederatedLearning) copyModel() *ModelParameters {
	weightsCopy := make(map[string][]float64)
	biasesCopy := make(map[string][]float64)

	for k, v := range fl.model.Weights {
		copied := make([]float64, len(v))
		copy(copied, v)
		weightsCopy[k] = copied
	}

	for k, v := range fl.model.Biases {
		copied := make([]float64, len(v))
		copy(copied, v)
		biasesCopy[k] = copied
	}

	return &ModelParameters{
		Weights: weightsCopy,
		Biases:  biasesCopy,
		Version: fl.model.Version,
		Round:   fl.round,
	}
}

func (fl *FederatedLearning) StopTraining() {
	fl.mu.Lock()
	defer fl.mu.Unlock()
	fl.isRunning = false
}

func (fl *FederatedLearning) GetGlobalModel() *ModelParameters {
	fl.mu.RLock()
	defer fl.mu.RUnlock()
	return fl.copyModel()
}

func (fl *FederatedLearning) GetCurrentRound() int {
	fl.mu.RLock()
	defer fl.mu.RUnlock()
	return fl.round
}

func (fl *FederatedLearning) GetNumClients() int {
	fl.mu.RLock()
	defer fl.mu.RUnlock()
	return len(fl.clients)
}

func (fl *FederatedLearning) GetEventLog() []FederatedEvent {
	fl.mu.RLock()
	defer fl.mu.RUnlock()
	return append([]FederatedEvent{}, fl.eventLog...)
}

func (fl *FederatedLearning) logEvent(round int, clientID, eventType string, data map[string]interface{}) {
	fl.eventLog = append(fl.eventLog, FederatedEvent{
		Round:    round,
		ClientID: clientID,
		Type:     eventType,
		Time:     time.Now(),
		Data:     data,
	})
}

func (fl *FederatedLearning) ExportModel() ([]byte, error) {
	fl.mu.RLock()
	defer fl.mu.RUnlock()
	return json.Marshal(fl.model)
}

func (fl *FederatedLearning) ImportModel(data []byte) error {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	var model ModelParameters
	if err := json.Unmarshal(data, &model); err != nil {
		return err
	}

	fl.model = &model
	return nil
}

func (fl *FederatedLearning) GetServer() *FederatedServer {
	return fl.server
}

type ClientUpdate struct {
	ClientID      string
	Weights       map[string][]float64
	Biases        map[string][]float64
	DataSize      int
	GradientNorm  float64
	Round         int
	Timestamp     time.Time
}

type FederatedStatistics struct {
	TotalClients       int
	ActiveClients      int
	CompletedRounds    int
	AverageUpdateSize  float64
	AverageGradNorm    float64
	PrivacyBudgetUsed  float64
}

func (fl *FederatedLearning) GetStatistics() FederatedStatistics {
	fl.mu.RLock()
	defer fl.mu.RUnlock()

	avgGradNorm := 0.0
	count := 0
	for _, event := range fl.eventLog {
		if event.Type == "update_submitted" {
			if norm, ok := event.Data["gradient_norm"].(float64); ok {
				avgGradNorm += norm
				count++
			}
		}
	}

	if count > 0 {
		avgGradNorm /= float64(count)
	}

	privacyUsed := 0.0
	if fl.config.UseDP {
		privacyUsed = float64(fl.round) * fl.config.DPConfig.Epsilon
	}

	return FederatedStatistics{
		TotalClients:      len(fl.clients),
		ActiveClients:     len(fl.clients),
		CompletedRounds:   fl.round,
		AverageUpdateSize: 0,
		AverageGradNorm:   avgGradNorm,
		PrivacyBudgetUsed: privacyUsed,
	}
}

var ErrTrainingAlreadyRunning = &FederatedError{message: "training is already running"}
var ErrNoClients = &FederatedError{message: "no clients registered"}

type FederatedError struct {
	message string
}

func (e *FederatedError) Error() string {
	return e.message
}

type FederatedCheckpoint struct {
	Round           int
	Model           *ModelParameters
	Statistics      FederatedStatistics
	Timestamp       time.Time
}

func (fl *FederatedLearning) CreateCheckpoint() *FederatedCheckpoint {
	fl.mu.RLock()
	defer fl.mu.RUnlock()

	return &FederatedCheckpoint{
		Round:      fl.round,
		Model:      fl.copyModel(),
		Statistics: fl.GetStatistics(),
		Timestamp:  time.Now(),
	}
}

func (fl *FederatedLearning) RestoreFromCheckpoint(checkpoint *FederatedCheckpoint) {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	fl.round = checkpoint.Round
	fl.model = checkpoint.Model
}
