package risk

import (
	"math"
	"math/rand"
	"time"
)

type HMMState string

const (
	HMMStateHumanNormal  HMMState = "human_normal"
	HMMStateHumanRush    HMMState = "human_rush"
	HMMStateMachine      HMMState = "machine"
	HMMStateUncertain    HMMState = "uncertain"
)

type HMMObservation string

const (
	ObsFastSlide     HMMObservation = "fast_slide"
	ObsSlowSlide     HMMObservation = "slow_slide"
	ObsNormalSlide   HMMObservation = "normal_slide"
	ObsSmoothTrack   HMMObservation = "smooth_track"
	ObsJitteryTrack  HMMObservation = "jittery_track"
	ObsFastClick     HMMObservation = "fast_click"
	ObsNormalClick   HMMObservation = "normal_click"
	ObsUniformClick  HMMObservation = "uniform_click"
	ObsVariableTime  HMMObservation = "variable_time"
	ObsConsistentTime HMMObservation = "consistent_time"
)

type HMMModel struct {
	States            []HMMState
	Observations      []HMMObservation
	InitialProb       map[HMMState]float64
	TransitionMatrix  map[HMMState]map[HMMState]float64
	EmissionMatrix    map[HMMState]map[HMMObservation]float64
}

type BehaviorSequence struct {
	Observations []HMMObservation
	StatePath    []HMMState
	Probability  float64
	DecodedState HMMState
}

func NewHMMModel() *HMMModel {
	model := &HMMModel{
		States:       []HMMState{HMMStateHumanNormal, HMMStateHumanRush, HMMStateMachine, HMMStateUncertain},
		Observations: []HMMObservation{
			ObsFastSlide, ObsSlowSlide, ObsNormalSlide,
			ObsSmoothTrack, ObsJitteryTrack,
			ObsFastClick, ObsNormalClick, ObsUniformClick,
			ObsVariableTime, ObsConsistentTime,
		},
		InitialProb:      make(map[HMMState]float64),
		TransitionMatrix: make(map[HMMState]map[HMMState]float64),
		EmissionMatrix:   make(map[HMMState]map[HMMObservation]float64),
	}

	model.InitialProb = map[HMMState]float64{
		HMMStateHumanNormal: 0.5,
		HMMStateHumanRush:   0.2,
		HMMStateMachine:     0.2,
		HMMStateUncertain:  0.1,
	}

	model.TransitionMatrix = map[HMMState]map[HMMState]float64{
		HMMStateHumanNormal: {
			HMMStateHumanNormal: 0.7,
			HMMStateHumanRush:   0.1,
			HMMStateMachine:     0.05,
			HMMStateUncertain:   0.15,
		},
		HMMStateHumanRush: {
			HMMStateHumanNormal: 0.3,
			HMMStateHumanRush:   0.4,
			HMMStateMachine:     0.1,
			HMMStateUncertain:   0.2,
		},
		HMMStateMachine: {
			HMMStateHumanNormal: 0.1,
			HMMStateHumanRush:   0.05,
			HMMStateMachine:     0.8,
			HMMStateUncertain:   0.05,
		},
		HMMStateUncertain: {
			HMMStateHumanNormal: 0.25,
			HMMStateHumanRush:   0.25,
			HMMStateMachine:     0.25,
			HMMStateUncertain:   0.25,
		},
	}

	model.EmissionMatrix = map[HMMState]map[HMMObservation]float64{
		HMMStateHumanNormal: {
			ObsFastSlide:      0.05,
			ObsSlowSlide:      0.1,
			ObsNormalSlide:    0.4,
			ObsSmoothTrack:    0.2,
			ObsJitteryTrack:  0.3,
			ObsFastClick:      0.1,
			ObsNormalClick:    0.4,
			ObsUniformClick:   0.1,
			ObsVariableTime:   0.35,
			ObsConsistentTime: 0.15,
		},
		HMMStateHumanRush: {
			ObsFastSlide:      0.25,
			ObsSlowSlide:      0.05,
			ObsNormalSlide:    0.2,
			ObsSmoothTrack:    0.2,
			ObsJitteryTrack:  0.2,
			ObsFastClick:      0.25,
			ObsNormalClick:    0.2,
			ObsUniformClick:   0.15,
			ObsVariableTime:   0.15,
			ObsConsistentTime: 0.2,
		},
		HMMStateMachine: {
			ObsFastSlide:      0.35,
			ObsSlowSlide:      0.05,
			ObsNormalSlide:    0.1,
			ObsSmoothTrack:    0.5,
			ObsJitteryTrack:  0.05,
			ObsFastClick:      0.3,
			ObsNormalClick:    0.1,
			ObsUniformClick:   0.35,
			ObsVariableTime:   0.05,
			ObsConsistentTime: 0.4,
		},
		HMMStateUncertain: {
			ObsFastSlide:      0.2,
			ObsSlowSlide:      0.2,
			ObsNormalSlide:    0.2,
			ObsSmoothTrack:    0.2,
			ObsJitteryTrack:  0.2,
			ObsFastClick:      0.2,
			ObsNormalClick:    0.2,
			ObsUniformClick:   0.2,
			ObsVariableTime:   0.2,
			ObsConsistentTime: 0.2,
		},
	}

	return model
}

func (e *RiskEngine) CreateBehaviorSequence(behavior *BehaviorData) *BehaviorSequence {
	seq := &BehaviorSequence{
		Observations: make([]HMMObservation, 0),
	}

	if len(behavior.MouseTracks) >= 2 {
		seq.Observations = append(seq.Observations, e.classifySlideSpeed(behavior))
		seq.Observations = append(seq.Observations, e.classifyTrackSmoothness(behavior.MouseTracks))
	}

	if len(behavior.ClickTimes) >= 2 {
		seq.Observations = append(seq.Observations, e.classifyClickPattern(behavior.ClickTimes))
	}

	return seq
}

func (e *RiskEngine) classifySlideSpeed(behavior *BehaviorData) HMMObservation {
	duration := float64(behavior.SlideEnd - behavior.SlideStart)
	if duration <= 0 {
		return ObsNormalSlide
	}

	if duration < 1000 {
		return ObsFastSlide
	} else if duration > 30000 {
		return ObsSlowSlide
	}
	return ObsNormalSlide
}

func (e *RiskEngine) classifyTrackSmoothness(tracks []MouseTrack) HMMObservation {
	if len(tracks) < 3 {
		return ObsNormalSlide
	}

	smoothness := e.calculateSmoothness(tracks)
	jitter := e.calculateJitter(tracks)

	if smoothness > 0.9 && jitter < 0.3 {
		return ObsSmoothTrack
	} else if jitter > 1.0 {
		return ObsJitteryTrack
	}
	return ObsNormalSlide
}

func (e *RiskEngine) classifyClickPattern(clicks []int64) HMMObservation {
	if len(clicks) < 2 {
		return ObsNormalClick
	}

	var intervals []float64
	for i := 1; i < len(clicks); i++ {
		intervals = append(intervals, float64(clicks[i]-clicks[i-1]))
	}

	minInterval := intervals[0]
	for _, interval := range intervals {
		if interval < minInterval {
			minInterval = interval
		}
	}
	if minInterval < 50 {
		return ObsFastClick
	}

	variance := e.calculateRhythmVariance(clicks)
	if variance < 0.05 {
		return ObsUniformClick
	}

	allSame := true
	for i := 1; i < len(intervals); i++ {
		if math.Abs(intervals[i]-intervals[i-1]) > 5 {
			allSame = false
			break
		}
	}
	if allSame && len(intervals) >= 3 {
		return ObsUniformClick
	}

	return ObsNormalClick
}

func (model *HMMModel) ForwardAlgorithm(observations []HMMObservation) (float64, []map[HMMState]float64) {
	if len(observations) == 0 {
		return 0.0, nil
	}

	alpha := make([]map[HMMState]float64, len(observations))
	alpha[0] = make(map[HMMState]float64)

	for _, state := range model.States {
		alpha[0][state] = model.InitialProb[state] * model.EmissionMatrix[state][observations[0]]
	}

	for t := 1; t < len(observations); t++ {
		alpha[t] = make(map[HMMState]float64)

		for _, currState := range model.States {
			var sum float64
			for _, prevState := range model.States {
				sum += alpha[t-1][prevState] * model.TransitionMatrix[prevState][currState]
			}
			alpha[t][currState] = sum * model.EmissionMatrix[currState][observations[t]]
		}
	}

	var totalProb float64
	for _, state := range model.States {
		totalProb += alpha[len(alpha)-1][state]
	}

	return totalProb, alpha
}

func (model *HMMModel) ViterbiAlgorithm(observations []HMMObservation) ([]HMMState, float64) {
	if len(observations) == 0 {
		return nil, 0.0
	}

	n := len(observations)
	m := len(model.States)

	delta := make([][]float64, n)
	psi := make([][]int, n)

	for i := 0; i < n; i++ {
		delta[i] = make([]float64, m)
		psi[i] = make([]int, m)
	}

	for j, state := range model.States {
		delta[0][j] = model.InitialProb[state] * model.EmissionMatrix[state][observations[0]]
		psi[0][j] = 0
	}

	for t := 1; t < n; t++ {
		for j, currState := range model.States {
			var maxProb float64
			var maxIdx int

			for i, prevState := range model.States {
				prob := delta[t-1][i] * model.TransitionMatrix[prevState][currState]
				if prob > maxProb {
					maxProb = prob
					maxIdx = i
				}
			}

			delta[t][j] = maxProb * model.EmissionMatrix[currState][observations[t]]
			psi[t][j] = maxIdx
		}
	}

	var maxProb float64
	var maxIdx int
	for j, prob := range delta[n-1] {
		if prob > maxProb {
			maxProb = prob
			maxIdx = j
		}
	}

	path := make([]HMMState, n)
	path[n-1] = model.States[maxIdx]

	for t := n - 2; t >= 0; t-- {
		maxIdx = psi[t+1][maxIdx]
		path[t] = model.States[maxIdx]
	}

	return path, maxProb
}

func (model *HMMModel) CalculateStateProbability(alpha map[HMMState]float64) map[HMMState]float64 {
	var total float64
	for _, prob := range alpha {
		total += prob
	}

	if total == 0 {
		return make(map[HMMState]float64)
	}

	normalized := make(map[HMMState]float64)
	for state, prob := range alpha {
		normalized[state] = prob / total
	}

	return normalized
}

func (e *RiskEngine) AnalyzeBehaviorSequence(behavior *BehaviorData) (int, []RiskFactor) {
	var factors []RiskFactor
	score := 0

	if len(behavior.MouseTracks) < 2 && len(behavior.ClickTimes) < 2 {
		return score, factors
	}

	model := NewHMMModel()

	seq := e.CreateBehaviorSequence(behavior)

	if len(seq.Observations) == 0 {
		return score, factors
	}

	_, alpha := model.ForwardAlgorithm(seq.Observations)

	stateProbs := model.CalculateStateProbability(alpha[len(alpha)-1])

	machineProb := stateProbs[HMMStateMachine]
	humanProb := stateProbs[HMMStateHumanNormal] + stateProbs[HMMStateHumanRush]
	uncertainProb := stateProbs[HMMStateUncertain]

	if machineProb > 0.7 {
		score += 35
		factors = append(factors, RiskFactor{
			Name:   "hmm_machine_pattern",
			Weight: 35,
			Reason: "行为序列高度符合机器模式(P>0.7)",
		})
	} else if machineProb > 0.5 {
		score += 25
		factors = append(factors, RiskFactor{
			Name:   "hmm_likely_machine",
			Weight: 25,
			Reason: "行为序列可能为机器生成(P>0.5)",
		})
	} else if machineProb > 0.3 {
		score += 15
		factors = append(factors, RiskFactor{
			Name:   "hmm_suspicious_pattern",
			Weight: 15,
			Reason: "行为序列存在可疑模式",
		})
	}

	if humanProb > 0.8 && machineProb < 0.1 {
		score -= 10
	}

	if uncertainProb > 0.5 {
		score += 10
		factors = append(factors, RiskFactor{
			Name:   "hmm_uncertain_state",
			Weight: 10,
			Reason: "行为状态不确定，可能存在伪装",
		})
	}

	path, _ := model.ViterbiAlgorithm(seq.Observations)
	if len(path) >= 2 {
		machineCount := 0
		for _, state := range path {
			if state == HMMStateMachine {
				machineCount++
			}
		}

		if float64(machineCount)/float64(len(path)) > 0.5 {
			score += 15
			factors = append(factors, RiskFactor{
				Name:   "hmm_frequent_machine_state",
				Weight: 15,
				Reason: "状态序列中频繁出现机器状态",
			})
		}
	}

	if machineProb > 0.3 && humanProb < 0.3 {
		score += 20
		factors = append(factors, RiskFactor{
			Name:   "hmm_pattern_mismatch",
			Weight: 20,
			Reason: "行为特征与人类行为模式严重不匹配",
		})
	}

	return score, factors
}

func (e *RiskEngine) UpdateHMMModel(model *HMMModel, observation HMMObservation, actualState HMMState, learningRate float64) {
	if model.EmissionMatrix[actualState] == nil {
		model.EmissionMatrix[actualState] = make(map[HMMObservation]float64)
	}

	if _, exists := model.EmissionMatrix[actualState][observation]; !exists {
		model.EmissionMatrix[actualState][observation] = 1.0 / float64(len(model.Observations))
	}

	currentProb := model.EmissionMatrix[actualState][observation]
	model.EmissionMatrix[actualState][observation] = currentProb*(1-learningRate) + learningRate

	if model.EmissionMatrix[actualState][observation] > 0.99 {
		model.EmissionMatrix[actualState][observation] = 0.99
	}

	sum := 0.0
	for _, obs := range model.Observations {
		if obs != observation {
			sum += model.EmissionMatrix[actualState][obs]
		}
	}
	if sum > 0 {
		adjustment := (1.0 - model.EmissionMatrix[actualState][observation]) / sum
		for _, obs := range model.Observations {
			if obs != observation {
				model.EmissionMatrix[actualState][obs] *= (1 - adjustment)
			}
		}
	}
}

func (e *RiskEngine) AnalyzeSequenceEntropy(observations []HMMObservation) float64 {
	if len(observations) == 0 {
		return 0.0
	}

	counts := make(map[HMMObservation]int)
	for _, obs := range observations {
		counts[obs]++
	}

	var entropy float64
	n := float64(len(observations))

	for _, count := range counts {
		if count > 0 {
			p := float64(count) / n
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

func (e *RiskEngine) DetectBehaviorAnomalies(behavior *BehaviorData) (int, []RiskFactor) {
	var factors []RiskFactor
	score := 0

	if len(behavior.MouseTracks) < 2 {
		return score, factors
	}

	var velocities []float64
	for i := 1; i < len(behavior.MouseTracks); i++ {
		dx := behavior.MouseTracks[i].X - behavior.MouseTracks[i-1].X
		dy := behavior.MouseTracks[i].Y - behavior.MouseTracks[i-1].Y
		dt := float64(behavior.MouseTracks[i].Timestamp - behavior.MouseTracks[i-1].Timestamp)
		if dt > 0 {
			velocities = append(velocities, math.Sqrt(dx*dx+dy*dy)/dt*1000)
		}
	}

	if len(velocities) >= 3 {
		var mean, variance float64
		for _, v := range velocities {
			mean += v
		}
		mean /= float64(len(velocities))

		for _, v := range velocities {
			diff := v - mean
			variance += diff * diff
		}
		variance /= float64(len(velocities))
		stdDev := math.Sqrt(variance)

		outliers := 0
		for _, v := range velocities {
			zScore := math.Abs((v - mean) / (stdDev + 0.001))
			if zScore > 3 {
				outliers++
			}
		}

		if float64(outliers)/float64(len(velocities)) > 0.3 {
			score += 15
			factors = append(factors, RiskFactor{
				Name:   "velocity_outliers",
				Weight: 15,
				Reason: "速度数据存在大量异常值",
			})
		}
	}

	if len(behavior.ClickTimes) >= 3 {
		entropy := e.AnalyzeSequenceEntropy([]HMMObservation{e.classifyClickPattern(behavior.ClickTimes)})

		var intervals []float64
		for i := 1; i < len(behavior.ClickTimes); i++ {
			intervals = append(intervals, float64(behavior.ClickTimes[i]-behavior.ClickTimes[i-1]))
		}

		var mean, variance float64
		for _, interval := range intervals {
			mean += interval
		}
		mean /= float64(len(intervals))

		for _, interval := range intervals {
			diff := interval - mean
			variance += diff * diff
		}
		variance /= float64(len(intervals))
		stdDev := math.Sqrt(variance)

		coefficientVariation := stdDev / (mean + 0.001)

		if coefficientVariation < 0.01 && entropy < 0.5 {
			score += 20
			factors = append(factors, RiskFactor{
				Name:   "mechanical_click_sequence",
				Weight: 20,
				Reason: "点击序列高度规律，疑似自动化脚本",
			})
		}
	}

	return score, factors
}

func (e *RiskEngine) AnalyzeTemporalPattern(behavior *BehaviorData, historicalBehaviors []*BehaviorData) (int, []RiskFactor) {
	var factors []RiskFactor
	score := 0

	if len(historicalBehaviors) < 5 {
		return score, factors
	}

	var historicalDurations []float64
	for _, hb := range historicalBehaviors {
		duration := float64(hb.SlideEnd - hb.SlideStart)
		if duration > 0 {
			historicalDurations = append(historicalDurations, duration)
		}
	}

	if len(historicalDurations) >= 5 {
		var mean, variance float64
		for _, d := range historicalDurations {
			mean += d
		}
		mean /= float64(len(historicalDurations))

		for _, d := range historicalDurations {
			diff := d - mean
			variance += diff * diff
		}
		variance /= float64(len(historicalDurations))
		stdDev := math.Sqrt(variance)

		currentDuration := float64(behavior.SlideEnd - behavior.SlideStart)
		if currentDuration > 0 {
			zScore := math.Abs((currentDuration - mean) / (stdDev + 0.001))

			if zScore > 3 {
				score += 20
				factors = append(factors, RiskFactor{
					Name:   "temporal_deviation",
					Weight: 20,
					Reason: "行为时间与历史记录显著偏离",
				})
			}
		}
	}

	speedConsistency := 0
	for _, hb := range historicalBehaviors {
		if len(hb.MouseTracks) >= 2 {
			var avgVelocity float64
			for i := 1; i < len(hb.MouseTracks); i++ {
				dx := hb.MouseTracks[i].X - hb.MouseTracks[i-1].X
				dy := hb.MouseTracks[i].Y - hb.MouseTracks[i-1].Y
				dt := float64(hb.MouseTracks[i].Timestamp - hb.MouseTracks[i-1].Timestamp)
				if dt > 0 {
					avgVelocity += math.Sqrt(dx*dx+dy*dy) / dt * 1000
				}
			}
			avgVelocity /= float64(len(hb.MouseTracks)-1)

			if avgVelocity > 50 && avgVelocity < 500 {
				speedConsistency++
			}
		}
	}

	if float64(speedConsistency)/float64(len(historicalBehaviors)) > 0.8 && len(behavior.MouseTracks) >= 2 {
		var currentAvgVelocity float64
		for i := 1; i < len(behavior.MouseTracks); i++ {
			dx := behavior.MouseTracks[i].X - behavior.MouseTracks[i-1].X
			dy := behavior.MouseTracks[i].Y - behavior.MouseTracks[i-1].Y
			dt := float64(behavior.MouseTracks[i].Timestamp - behavior.MouseTracks[i-1].Timestamp)
			if dt > 0 {
				currentAvgVelocity += math.Sqrt(dx*dx+dy*dy) / dt * 1000
			}
		}
		if len(behavior.MouseTracks) > 1 {
			currentAvgVelocity /= float64(len(behavior.MouseTracks)-1)
		}

		if currentAvgVelocity < 10 || currentAvgVelocity > 1000 {
			score += 15
			factors = append(factors, RiskFactor{
				Name:   "unusual_speed_pattern",
				Weight: 15,
				Reason: "滑动速度异常偏离正常范围",
			})
		}
	}

	return score, factors
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
