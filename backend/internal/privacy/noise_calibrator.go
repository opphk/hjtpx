package privacy

import (
	"math"
	"sync"
)

type NoiseCalibrator struct {
	mechanismType MechanismType
	targetEpsilon float64
	targetDelta   float64
	delta         float64
	confidence    float64
	mu            sync.RWMutex
}

type MechanismType int

const (
	GaussianCalibration MechanismType = iota
	LaplaceCalibration
	ExponentialCalibration
	SnappingCalibration
)

type CalibrationConfig struct {
	MechanismType   MechanismType
	TargetEpsilon   float64
	TargetDelta     float64
	Confidence      float64
	Sensitivity     float64
}

func NewNoiseCalibrator(config CalibrationConfig) *NoiseCalibrator {
	return &NoiseCalibrator{
		mechanismType: config.MechanismType,
		targetEpsilon: config.TargetEpsilon,
		targetDelta:   config.TargetDelta,
		delta:         config.TargetDelta,
		confidence:    config.Confidence,
	}
}

func (nc *NoiseCalibrator) CalibrateSigma(sensitivity float64) float64 {
	nc.mu.RLock()
	defer nc.mu.RUnlock()

	switch nc.mechanismType {
	case GaussianCalibration:
		return nc.calibrateGaussianSigma(sensitivity)
	case LaplaceCalibration:
		return nc.calibrateLaplaceScale(sensitivity)
	default:
		return 0.0
	}
}

func (nc *NoiseCalibrator) calibrateGaussianSigma(sensitivity float64) float64 {
	c := math.Sqrt(2 * math.Log(1.25/nc.targetDelta))
	return c * sensitivity / nc.targetEpsilon
}

func (nc *NoiseCalibrator) calibrateLaplaceScale(sensitivity float64) float64 {
	return sensitivity / nc.targetEpsilon
}

func (nc *NoiseCalibrator) SetTargetEpsilon(epsilon float64) {
	nc.mu.Lock()
	defer nc.mu.Unlock()
	nc.targetEpsilon = epsilon
}

func (nc *NoiseCalibrator) SetTargetDelta(delta float64) {
	nc.mu.Lock()
	defer nc.mu.Unlock()
	nc.targetDelta = delta
}

func (nc *NoiseCalibrator) GetPrivacyParameters() (epsilon, delta float64) {
	nc.mu.RLock()
	defer nc.mu.RUnlock()
	return nc.targetEpsilon, nc.targetDelta
}

type PrivacyCalibrator struct {
	delta         float64
	sensitivity   float64
	mechanism     string
	mu            sync.RWMutex
}

func NewPrivacyCalibrator(delta, sensitivity float64) *PrivacyCalibrator {
	return &PrivacyCalibrator{
		delta:       delta,
		sensitivity: sensitivity,
		mechanism:   "gaussian",
	}
}

func (pc *PrivacyCalibrator) CalibrateEpsilon(epsilon float64) float64 {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	if pc.mechanism == "gaussian" {
		return pc.calibrateGaussian(epsilon)
	}
	return epsilon
}

func (pc *PrivacyCalibrator) calibrateGaussian(epsilon float64) float64 {
	return epsilon * math.Sqrt(2*math.Log(1.25/pc.delta))
}

func (pc *PrivacyCalibrator) SetMechanism(mechanism string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.mechanism = mechanism
}

func (pc *PrivacyCalibrator) GetMechanism() string {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	return pc.mechanism
}

type RDPCalibrator struct {
	rho           float64
	alpha         float64
	sensitivity   float64
	mu            sync.RWMutex
}

func NewRDPCalibrator(rho, alpha, sensitivity float64) *RDPCalibrator {
	return &RDPCalibrator{
		rho:         rho,
		alpha:       alpha,
		sensitivity: sensitivity,
	}
}

func (rc *RDPCalibrator) ComputeEpsilonFromRDP(delta float64) float64 {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	epsilon := rc.rho * (rc.alpha - 1)
	deltaApprox := math.Pow(rc.alpha, -rc.rho)
	if deltaApprox > delta {
		return epsilon
	}
	return rc.rho * rc.alpha
}

func (rc *RDPCalibrator) ComputeDeltaFromRDP(epsilon float64) float64 {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	if epsilon <= rc.rho*rc.alpha {
		return 0.0
	}
	alphaOptimal := epsilon / rc.rho
	return math.Pow(alphaOptimal, -rc.rho)
}

func (rc *RDPCalibrator) SetRho(rho float64) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.rho = rho
}

func (rc *RDPCalibrator) GetRho() float64 {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.rho
}

type VuardCalibrator struct {
	epsilon       float64
	delta         float64
	sensitivity   float64
	mu            sync.RWMutex
}

func NewVuardCalibrator(epsilon, delta, sensitivity float64) *VuardCalibrator {
	return &VuardCalibrator{
		epsilon:     epsilon,
		delta:       delta,
		sensitivity: sensitivity,
	}
}

func (vc *VuardCalibrator) Calibrate() (sigma float64) {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	sigma = vc.sensitivity * math.Sqrt(2*math.Log(1.25/vc.delta)) / vc.epsilon
	return sigma
}

func (vc *VuardCalibrator) VerifyPrivacy(noiseSamples []float64, numQueries int) bool {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	threshold := math.Log(1 / vc.delta)
	positiveCount := 0

	for _, sample := range noiseSamples {
		if math.Abs(sample) < threshold {
			positiveCount++
		}
	}

	falsePositiveRate := float64(positiveCount) / float64(len(noiseSamples))
	return falsePositiveRate < vc.delta
}

type SensitivityCalibrator struct {
	lowerBound    float64
	upperBound    float64
	mu            sync.RWMutex
}

func NewSensitivityCalibrator(lowerBound, upperBound float64) *SensitivityCalibrator {
	return &SensitivityCalibrator{
		lowerBound: lowerBound,
		upperBound: upperBound,
	}
}

func (sc *SensitivityCalibrator) ComputeGlobalSensitivity(queryType string) float64 {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	switch queryType {
	case "sum", "mean":
		return sc.upperBound - sc.lowerBound
	case "count":
		return 1.0
	case "variance":
		return (sc.upperBound - sc.lowerBound) * (sc.upperBound - sc.lowerBound)
	default:
		return 1.0
	}
}

func (sc *SensitivityCalibrator) ComputeSmoothSensitivity(function func(float64) float64, epsilon float64) float64 {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	maxSensitivity := 0.0
	step := (sc.upperBound - sc.lowerBound) / 100.0

	for x := sc.lowerBound; x <= sc.upperBound; x += step {
		fx := function(x)
		for d := sc.lowerBound; d <= sc.upperBound; d += step {
			fxd := function(x + d)
			sensitivity := math.Abs(fx - fxd) * math.Exp(-epsilon*d)
			if sensitivity > maxSensitivity {
				maxSensitivity = sensitivity
			}
		}
	}

	return maxSensitivity
}

func (sc *SensitivityCalibrator) SetBounds(lower, upper float64) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.lowerBound = lower
	sc.upperBound = upper
}

func (sc *SensitivityCalibrator) GetBounds() (lower, upper float64) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.lowerBound, sc.upperBound
}

type AdaptiveCalibrator struct {
	baseEpsilon   float64
	baseDelta     float64
	adaptationFunc func(int, float64) float64
	currentRound  int
	mu            sync.RWMutex
}

func NewAdaptiveCalibrator(baseEpsilon, baseDelta float64) *AdaptiveCalibrator {
	return &AdaptiveCalibrator{
		baseEpsilon: baseEpsilon,
		baseDelta:   baseDelta,
		adaptationFunc: func(round int, epsilon float64) float64 {
			return epsilon * math.Sqrt(float64(round+1))
		},
		currentRound: 0,
	}
}

func (ac *AdaptiveCalibrator) GetCurrentEpsilon() float64 {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return ac.adaptationFunc(ac.currentRound, ac.baseEpsilon)
}

func (ac *AdaptiveCalibrator) IncrementRound() {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.currentRound++
}

func (ac *AdaptiveCalibrator) GetCurrentRound() int {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return ac.currentRound
}

func (ac *AdaptiveCalibrator) SetAdaptationFunc(f func(int, float64) float64) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.adaptationFunc = f
}

type CompositionCalibrator struct {
	totalEpsilon float64
	totalDelta   float64
	spentEpsilon float64
	spentDelta   float64
	mu           sync.Mutex
}

func NewCompositionCalibrator(totalEpsilon, totalDelta float64) *CompositionCalibrator {
	return &CompositionCalibrator{
		totalEpsilon: totalEpsilon,
		totalDelta:   totalDelta,
	}
}

func (cc *CompositionCalibrator) Allocate(epsilon, delta float64) bool {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if cc.spentEpsilon+epsilon > cc.totalEpsilon {
		return false
	}
	if cc.spentDelta+delta > cc.totalDelta {
		return false
	}

	cc.spentEpsilon += epsilon
	cc.spentDelta += delta
	return true
}

func (cc *CompositionCalibrator) GetRemaining() (epsilon, delta float64) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	return cc.totalEpsilon - cc.spentEpsilon, cc.totalDelta - cc.spentDelta
}

func (cc *CompositionCalibrator) ComputeAdvancedComposition(numQueries int, epsilon, delta float64) (float64, float64) {
	chiSquared := 2 * math.Log(2/delta)
	composedEpsilon := epsilon*math.Sqrt(2*chiSquared*float64(numQueries)) + chiSquared*float64(numQueries)*(epsilon*epsilon)
	composedDelta := delta * float64(numQueries)
	return composedEpsilon, composedDelta
}
