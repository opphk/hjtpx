package service

import (
	"fmt"
	"math"
	"sort"
	"time"
)

type RandomnessQualityChecker struct {
	sampleSize     int
	confidenceLevel float64
	testResults    map[string]*TestResult
}

type TestResult struct {
	TestName       string
	Statistic       float64
	PValue         float64
	Passed         bool
	CriticalValue  float64
	SampleSize     int
	Timestamp      int64
}

type QualityReport struct {
	OverallScore    float64                  `json:"overall_score"`
	Grade          string                    `json:"grade"`
	TestsPassed    int                       `json:"tests_passed"`
	TestsFailed    int                       `json:"tests_failed"`
	TotalTests     int                       `json:"total_tests"`
	EntropyBits    float64                  `json:"entropy_bits"`
	ChiSquareScore float64                   `json:"chi_square_score"`
	Results        map[string]*TestResult    `json:"results"`
	Recommendations []string                 `json:"recommendations"`
	Timestamp      int64                     `json:"timestamp"`
}

func NewRandomnessQualityChecker() *RandomnessQualityChecker {
	return &RandomnessQualityChecker{
		sampleSize:      10000,
		confidenceLevel: 0.95,
		testResults:     make(map[string]*TestResult),
	}
}

func (c *RandomnessQualityChecker) CheckQuality(data []byte) (*QualityReport, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("no data provided")
	}

	c.testResults = make(map[string]*TestResult)

	c.runFrequencyTest(data)
	c.runBlockFrequencyTest(data, 4)
	c.runRunsTest(data)
	c.runLongestRunTest(data)
	c.runSerialTest(data)
	c.runEntropyTest(data)
	c.runAutocorrelationTest(data)

	report := c.generateReport(data)

	return report, nil
}

func (c *RandomnessQualityChecker) runFrequencyTest(data []byte) {
	n := len(data)
	ones := 0
	for _, b := range data {
		for i := 0; i < 8; i++ {
			if (b>>i)&1 == 1 {
				ones++
			}
	}
	onesCount := float64(ones)
	nBits := float64(n * 8)
	proportion := onesCount / nBits

	statistic := math.Pow(proportion-0.5, 2) * 2 * nBits
	pValue := 1.0 - chiSquarePValue(statistic, 1)

	c.testResults["frequency"] = &TestResult{
		TestName:      "Monobit Frequency Test",
		Statistic:     statistic,
		PValue:        pValue,
		Passed:        pValue >= (1 - c.confidenceLevel),
		CriticalValue: chiSquareCriticalValue(1, 1-c.confidenceLevel),
		SampleSize:    nBits,
		Timestamp:     time.Now().Unix(),
	}
}

func (c *RandomnessQualityChecker) runBlockFrequencyTest(data []byte, blockSize int) {
	if blockSize <= 0 {
		blockSize = 4
	}

	n := len(data) * 8
	numBlocks := n / blockSize
	if numBlocks == 0 {
		numBlocks = 1
	}

	proportions := make([]float64, numBlocks)
	for i := 0; i < numBlocks; i++ {
		ones := 0
		for j := 0; j < blockSize && (i*blockSize+j) < n; j++ {
			byteIndex := (i*blockSize + j) / 8
			bitIndex := (i*blockSize + j) % 8
			if byteIndex < len(data) && (data[byteIndex]>>bitIndex)&1 == 1 {
				ones++
			}
		}
		proportions[i] = float64(ones) / float64(blockSize)
	}

	var chiSquare float64
	for _, p := range proportions {
		chiSquare += math.Pow(p-0.5, 2) * 2 * float64(blockSize)
	}

	pValue := 1.0 - chiSquarePValue(chiSquare, float64(numBlocks))

	c.testResults["block_frequency"] = &TestResult{
		TestName:      fmt.Sprintf("Block Frequency Test (Block Size=%d)", blockSize),
		Statistic:     chiSquare,
		PValue:        pValue,
		Passed:        pValue >= (1 - c.confidenceLevel),
		CriticalValue: chiSquareCriticalValue(float64(numBlocks), 1-c.confidenceLevel),
		SampleSize:    n,
		Timestamp:     time.Now().Unix(),
	}
}

func (c *RandomnessQualityChecker) runRunsTest(data []byte) {
	bits := c.bytesToBits(data)
	n := len(bits)

	ones := 0
	for _, b := range bits {
		if b == 1 {
			ones++
		}
	}
	proportion := float64(ones) / float64(n)

	var runs int
	for i := 1; i < n; i++ {
		if bits[i] != bits[i-1] {
			runs++
		}
	}
	runs++

	expectedRuns := 2*float64(n)*proportion*(1-proportion) + 1
	var variance float64
	if proportion > 0 && proportion < 1 {
		variance = 2*(2*float64(n)-1)*proportion*(1-proportion) / float64(n-1)
	}

	var statistic float64
	if variance > 0 {
		statistic = math.Pow(float64(runs)-expectedRuns, 2) / variance
	} else {
		statistic = 0
	}

	pValue := 1.0 - chiSquarePValue(statistic, 1)

	c.testResults["runs"] = &TestResult{
		TestName:      "Runs Test",
		Statistic:     float64(runs),
		PValue:        pValue,
		Passed:        pValue >= (1 - c.confidenceLevel),
		CriticalValue: expectedRuns,
		SampleSize:    n,
		Timestamp:     time.Now().Unix(),
	}
}

func (c *RandomnessQualityChecker) runLongestRunTest(data []byte) {
	bits := c.bytesToBits(data)
	n := len(bits)

	longestRun := 0
	currentRun := 1
	for i := 1; i < n; i++ {
		if bits[i] == bits[i-1] {
			currentRun++
			if currentRun > longestRun {
				longestRun = currentRun
			}
		} else {
			currentRun = 1
		}
	}

	expectedLongestRun := 0
	if n >= 10000 {
		expectedLongestRun = 34
	} else if n >= 5000 {
		expectedLongestRun = 31
	} else if n >= 1000 {
		expectedLongestRun = 26
	} else {
		expectedLongestRun = int(math.Log2(float64(n))) + 3
	}

	variance := 0.0
	if n >= 10000 {
		variance = 15.6
	} else if n >= 5000 {
		variance = 13.8
	} else if n >= 1000 {
		variance = 11.5
	} else {
		variance = 5.0
	}

	var statistic float64
	if variance > 0 {
		statistic = math.Pow(float64(longestRun)-float64(expectedLongestRun), 2) / variance
	} else {
		statistic = 0
	}

	pValue := 1.0 - chiSquarePValue(statistic, 1)

	c.testResults["longest_run"] = &TestResult{
		TestName:      "Longest Run of Ones Test",
		Statistic:     float64(longestRun),
		PValue:        pValue,
		Passed:        pValue >= (1 - c.confidenceLevel),
		CriticalValue: float64(expectedLongestRun),
		SampleSize:    n,
		Timestamp:     time.Now().Unix(),
	}
}

func (c *RandomnessQualityChecker) runSerialTest(data []byte) {
	bits := c.bytesToBits(data)
	n := len(bits)

	frequencies := make(map[int]int)
	for i := 0; i < n-1; i++ {
		pair := bits[i]*2 + bits[i+1]
		frequencies[pair]++
	}

	chiSquare := 0.0
	expectedFreq := float64(n - 1)

	for i := 0; i < 4; i++ {
		observed := float64(frequencies[i])
		chiSquare += math.Pow(observed-expectedFreq, 2) / expectedFreq
	}

	pValue := 1.0 - chiSquarePValue(chiSquare, 2)

	c.testResults["serial"] = &TestResult{
		TestName:      "Serial Test",
		Statistic:     chiSquare,
		PValue:        pValue,
		Passed:        pValue >= (1 - c.confidenceLevel),
		CriticalValue: chiSquareCriticalValue(2, 1-c.confidenceLevel),
		SampleSize:    n,
		Timestamp:     time.Now().Unix(),
	}
}

func (c *RandomnessQualityChecker) runEntropyTest(data []byte) {
	frequencies := make(map[byte]int)
	for _, b := range data {
		frequencies[b]++
	}

	n := len(data)
	entropy := 0.0
	for _, freq := range frequencies {
		if freq > 0 {
			p := float64(freq) / float64(n)
			entropy -= p * math.Log2(p)
		}
	}

	maxEntropy := 8.0
	entropyRatio := entropy / maxEntropy

	chiSquareScore := entropyRatio * 100

	c.testResults["entropy"] = &TestResult{
		TestName:      "Entropy Test",
		Statistic:     entropy,
		PValue:        entropyRatio,
		Passed:        entropyRatio >= 0.95,
		CriticalValue: 7.6,
		SampleSize:    n,
		Timestamp:     time.Now().Unix(),
	}
}

func (c *RandomnessQualityChecker) runAutocorrelationTest(data []byte) {
	bits := c.bytesToBits(data)
	n := len(bits)

	if n < 100 {
		c.testResults["autocorrelation"] = &TestResult{
			TestName:      "Autocorrelation Test",
			Statistic:     0,
			PValue:        1.0,
			Passed:        true,
			CriticalValue: 0,
			SampleSize:    n,
			Timestamp:     time.Now().Unix(),
		}
		return
	}

	shift := 8
	ones1 := 0
	ones2 := 0
	matches := 0

	for i := 0; i < n-shift; i++ {
		if bits[i] == 1 {
			ones1++
		}
		if bits[i+shift] == 1 {
			ones2++
		}
		if bits[i] == bits[i+shift] {
			matches++
		}
	}

	observedMatches := float64(matches)
	expectedMatches := float64(n-shift) * 0.5

	var statistic float64
	variance := float64(n-shift) * 0.25
	if variance > 0 {
		statistic = math.Pow(observedMatches-expectedMatches, 2) / variance
	} else {
		statistic = 0
	}

	pValue := 1.0 - chiSquarePValue(statistic, 1)

	c.testResults["autocorrelation"] = &TestResult{
		TestName:      "Autocorrelation Test",
		Statistic:     statistic,
		PValue:        pValue,
		Passed:        pValue >= (1 - c.confidenceLevel),
		CriticalValue: chiSquareCriticalValue(1, 1-c.confidenceLevel),
		SampleSize:    n,
		Timestamp:     time.Now().Unix(),
	}
}

func (c *RandomnessQualityChecker) bytesToBits(data []byte) []int {
	bits := make([]int, len(data)*8)
	for i, b := range data {
		for j := 0; j < 8; j++ {
			bits[i*8+j] = int((b >> j) & 1)
		}
	}
	return bits
}

func (c *RandomnessQualityChecker) generateReport(data []byte) *QualityReport {
	testsPassed := 0
	testsFailed := 0
	totalScore := 0.0
	entropyBits := 0.0

	for _, result := range c.testResults {
		if result.Passed {
			testsPassed++
		} else {
			testsFailed++
		}
		totalScore += result.PValue
	}

	if len(c.testResults) > 0 {
		totalScore /= float64(len(c.testResults))
	}

	if entropyResult, ok := c.testResults["entropy"]; ok {
		entropyBits = entropyResult.Statistic
	}

	chiSquareScore := 0.0
	if csResult, ok := c.testResults["chi_square"]; ok {
		chiSquareScore = csResult.Statistic
	}

	grade := c.calculateGrade(totalScore)

	recommendations := c.generateRecommendations(testsPassed, testsFailed, entropyBits)

	return &QualityReport{
		OverallScore:     totalScore * 100,
		Grade:           grade,
		TestsPassed:      testsPassed,
		TestsFailed:      testsFailed,
		TotalTests:      testsPassed + testsFailed,
		EntropyBits:     entropyBits,
		ChiSquareScore:  chiSquareScore,
		Results:         c.testResults,
		Recommendations: recommendations,
		Timestamp:       time.Now().Unix(),
	}
}

func (c *RandomnessQualityChecker) calculateGrade(score float64) string {
	switch {
	case score >= 95:
		return "A+ (Excellent)"
	case score >= 90:
		return "A (Very Good)"
	case score >= 85:
		return "B+ (Good)"
	case score >= 80:
		return "B (Satisfactory)"
	case score >= 70:
		return "C (Acceptable)"
	case score >= 60:
		return "D (Poor)"
	default:
		return "F (Failed)"
	}
}

func (c *RandomnessQualityChecker) generateRecommendations(passed, failed, entropy float64) []string {
	var recommendations []string

	if failed > passed/2 {
		recommendations = append(recommendations, "严重警告：随机性测试失败率较高，建议更换随机数生成源")
	}

	if entropy < 7.0 {
		recommendations = append(recommendations, "警告：熵值低于安全阈值，建议增加随机性来源")
	}

	if passed == 0 {
		recommendations = append(recommendations, "紧急：所有测试均未通过，随机数生成可能存在严重问题")
	}

	if failed > 0 {
		recommendations = append(recommendations, "建议：对失败的测试进行深入分析并优化生成算法")
	}

	if entropy >= 7.5 && passed >= len(c.testResults)/2 {
		recommendations = append(recommendations, "随机性质量良好，可继续使用当前随机数生成器")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "随机性质量优秀，符合密码学安全标准")
	}

	return recommendations
}

func chiSquarePValue(x float64, df float64) float64 {
	if x <= 0 || df <= 0 {
		return 1.0
	}

	gammaVal := math.Gamma(df / 2)
	lower := math.Pow(x/2, df/2) * math.Exp(-x/2) / gammaVal

	upper := 0.0
	term := 1.0
	for i := 1.0; i < 1000; i++ {
		term *= x / (2 * i)
		upper += term
		if term < 1e-10 {
			break
		}
	}

	pValue := lower * upper

	if pValue > 1 {
		pValue = 1
	}
	if pValue < 0 {
		pValue = 0
	}

	return 1 - pValue
}

func chiSquareCriticalValue(df float64, confidence float64) float64 {
	alpha := 1 - confidence

	switch {
	case df == 1:
		switch {
		case alpha >= 0.95: return 0.000039
		case alpha >= 0.90: return 0.0158
		case alpha >= 0.10: return 2.706
		case alpha >= 0.05: return 3.841
		default: return 5.024
		}
	case df == 2:
		switch {
		case alpha >= 0.95: return 0.0201
		case alpha >= 0.90: return 0.103
		case alpha >= 0.10: return 4.605
		case alpha >= 0.05: return 5.991
		default: return 7.378
		}
	default:
		baseValue := df * (1 - 2/(9*df))
		z := inverseNormalCDF(1 - alpha/2)
		modifier := z * math.Sqrt(2/(9*df))
		return baseValue + z*math.Sqrt(2/(9*df))
	}
}

func inverseNormalCDF(p float64) float64 {
	if p <= 0 {
		return -10
	}
	if p >= 1 {
		return 10
	}

	a := []float64{-3.969683028665376e+01, 2.209460984245205e+02, -2.759285104469687e+02, 1.383577518672690e+02, -3.066479806614716e+01, 2.506628277459239e+00}
	b := []float64{-5.447609879822406e+01, 1.615858368580409e+02, -1.556989798598866e+02, 6.680131188771972e+01, -1.328068155288572e+01}
	c := []float64{-7.784894002430293e-03, -3.223964580411365e-01, -2.400758277161838e+00, -2.549732539343734e+00, 4.374664141464968e+00, 2.938163982698783e+00}
	d := []float64{7.784695709041462e-03, 3.224671290740398e-01, 2.445134137142996e+00, 3.754408661907416e+00}

	pLow := 0.02425
	pHigh := 1 - pLow

	var q, r float64
	if p < pLow {
		q = math.Sqrt(-2 * math.Log(p))
	} else if p <= pHigh {
		q = p - 0.5
		r = q * q
	} else {
		q = math.Sqrt(-2 * math.Log(1 - p))
	}

	var x float64
	if p < pLow {
		x = (((((c[0]*q+c[1])*q+c[2])*q+c[3])*q+c[4])*q + c[5]) / ((((d[0]*q+d[1])*q+d[2])*q+d[3])*q+1)
	} else if p <= pHigh {
		x = (((((a[0]*r+a[1])*r+a[2])*r+a[3])*r+a[4])*r+a[5]) * q / ((((b[0]*r+b[1])*r+b[2])*r+b[3])*r+1)
	} else {
		x = (((((c[0]*q+c[1])*q+c[2])*q+c[3])*q+c[4])*q + c[5]) / ((((d[0]*q+d[1])*q+d[2])*q+d[3])*q+1)
		x = -x
	}

	return x
}

func (c *RandomnessQualityChecker) GetTestResults() map[string]*TestResult {
	return c.testResults
}

func (c *RandomnessQualityChecker) SetSampleSize(size int) {
	if size > 0 {
		c.sampleSize = size
	}
}

func (c *RandomnessQualityChecker) SetConfidenceLevel(level float64) {
	if level > 0 && level < 1 {
		c.confidenceLevel = level
	}
}
