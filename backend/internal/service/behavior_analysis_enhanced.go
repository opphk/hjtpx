package service

import (
	"encoding/json"
	"fmt"
	"math"
	"math/cmplx"
	"math/rand"
	"sort"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
)

// ============================================
// 第一部分：高级特征提取（10+ 新特征）
// ============================================

// EnhancedFeatures 包含所有新增的高级特征（768维特征向量扩展）
type EnhancedFeatures struct {
	// ============ 曲率特征（3个）============
	CurvatureVariance          float64   // 曲率变化方差
	CurvaturePeaks             int       // 曲率峰值数量
	CurvatureEntropy           float64   // 曲率熵

	// ============ 点击间隔特征（5个）============
	ClickIntervalSkewness      float64   // 点击间隔偏度
	ClickIntervalKurtosis      float64   // 点击间隔峰度
	ClickIntervalQuantiles     []float64 // 点击间隔分位数

	// ============ 方向变化特征（2个）============
	DirectionChangeFrequency   float64   // 方向变化频率
	DirectionChangeRegularity  float64   // 方向变化规律性

	// ============ 频域分析特征（2个）============
	FourierDominantFrequency   float64   // 傅里叶主频率
	FourierEnergyRatio         float64   // 傅里叶能量比

	// ============ 分形特征（1个）============
	FractalDimension           float64   // 分形维数

	// ============ 悬停特征（2个）============
	HoverCount                 int       // 悬停次数
	HoverDurationVariance      float64   // 悬停时长方差

	// ============ 滚动特征（2个）============
	ScrollSpeedVariance        float64   // 滚动速度方差
	ScrollDirectionChanges     int       // 滚动方向变化次数

	// ============ 键盘节奏特征（2个）============
	KeyPressIntervalRegularity float64   // 按键间隔规律性
	KeyHoldDurationVariance   float64   // 按键保持时长方差

	// ============ 停顿特征（2个）============
	PauseIntervalDistribution  []float64 // 停顿间隔分布
	PauseFrequency            float64   // 停顿频率

	// ============ 加加速度特征（2个）============
	JerkAverage                float64   // 平均加加速度（加速度的变化率）
	JerkVariance               float64   // 加加速度方差

	// ============ 速度特征扩展（15个）- 任务3.1&3.2 ============
	SpeedMedian                float64   // 速度中位数
	SpeedSkewness             float64   // 速度偏度
	SpeedKurtosis             float64   // 速度峰度
	SpeedRange                float64   // 速度范围
	SpeedIQR                  float64   // 速度四分位距
	SpeedCoefficientVariation float64   // 速度变异系数
	SpeedPercentile25         float64   // 速度25分位数
	SpeedPercentile75         float64   // 速度75分位数
	SpeedPercentile90         float64   // 速度90分位数
	UniformMotionRatio        float64   // 匀速运动比例
	AccelerationVariance      float64   // 加速度方差
	AccelerationSkewness      float64   // 加速度偏度
	AccelerationKurtosis      float64   // 加速度峰度
	HumanMachineSpeedDiff     float64   // 人机速度差异指标
	SpeedCurveComplexity      float64   // 速度曲线复杂度

	// ============ 抖动特征扩展（10个）- 任务3.3 ============
	JitterFrequency            float64   // 抖动频率
	JitterAmplitudeMean        float64   // 抖动幅度均值
	JitterAmplitudeMax         float64   // 抖动幅度最大值
	JitterAmplitudeVariance    float64   // 抖动幅度方差
	MicroJitterCount           int       // 微抖动次数
	MicroJitterRatio           float64   // 微抖动比例
	JitterRegularity           float64   // 抖动规律性
	JitterEntropy              float64   // 抖动熵
	JitterClusterCount         int       // 抖动聚类数
	JitterWaveformType         string    // 抖动波形类型

	// ============ 加速度特征扩展（8个）- 任务3.1 ============
	AccelerationMean          float64   // 加速度均值
	AccelerationRange        float64   // 加速度范围
	DecelerationRatio         float64   // 减速比例
	AccelerationPeakCount    int       // 加速度峰值数量
	AccelerationZeroCrossing int       // 加速度过零点数量
	AccelerationEnergy       float64   // 加速度能量
	TangentialAcceleration    float64   // 切向加速度
	NormalAcceleration        float64   // 法向加速度

	// ============ 时序特征扩展（12个）============
	InterPointTimeMean         float64   // 点间时间均值
	InterPointTimeVariance    float64   // 点间时间方差
	InterPointTimeSkewness    float64   // 点间时间偏度
	InterPointTimeCV          float64   // 点间时间变异系数
	TimeRegularityIndex       float64   // 时间规律性指数
	TimeBurstinessIndex       float64   // 时间突发性指数
	LongPauseCount            int       // 长停顿次数
	LongPauseRatio            float64   // 长停顿比例
	ShortPauseCount           int       // 短停顿次数
	ShortPauseRatio           float64   // 短停顿比例
	TimeSequenceEntropy       float64   // 时间序列熵
	TimeSequencePattern       string    // 时间序列模式

	// ============ 空间特征扩展（10个）============
	TotalDisplacement          float64   // 总位移
	NetDisplacement            float64   // 净位移
	DisplacementRatio          float64   // 位移比（净位移/总位移）
	AreaCovered               float64   // 覆盖面积
	AreaPerDistance           float64   // 单位距离覆盖面积
	SpatialConcentration      float64   // 空间集中度
	SpatialSpread             float64   // 空间分散度
	TrajectoryRoughness       float64   // 轨迹粗糙度
	TrajectorySelfSimilarity   float64   // 轨迹自相似性
	PathSinuosity             float64   // 路径蜿蜒度

	// ============ 统计特征扩展（8个）============
	PointCount                 int       // 轨迹点数量
	MeanPointInterval         float64   // 平均点间隔
	PointIntervalConsistency  float64   // 点间隔一致性
	TrajectoryDuration         float64   // 轨迹持续时间
	VelocityAutoCorrelation   float64   // 速度自相关系数
	VelocityLaggedCorrelation  float64   // 速度滞后相关性
	AccelerationAutoCorr      float64   // 加速度自相关系数
	MovementRhythmScore       float64   // 运动节奏评分

	// ============ 人机差异特征（5个）- 任务3.2 ============
	HumanLikenessScore        float64   // 人类相似度评分
	MechanicalPatternScore     float64   // 机械模式评分
	NaturalFluctuationScore   float64   // 自然波动评分
	PrecisionScore            float64   // 精准度评分
	ConsistencyScore          float64   // 一致性评分
}

// EnhancedBehaviorAnalysisService 增强版行为分析服务
type EnhancedBehaviorAnalysisService struct {
	*BehaviorAnalysisService
	ensembleClassifier *AdvancedEnsembleClassifier
	isolationForest    *IsolationForest
	adaptiveThreshold  *AdaptiveThreshold
	trainingData       [][]float64
	trainingLabels     []float64
}

// NewEnhancedBehaviorAnalysisService 创建增强版行为分析服务
func NewEnhancedBehaviorAnalysisService() *EnhancedBehaviorAnalysisService {
	return &EnhancedBehaviorAnalysisService{
		BehaviorAnalysisService: NewBehaviorAnalysisService(),
		ensembleClassifier:      NewAdvancedEnsembleClassifier(),
		isolationForest:         NewIsolationForest(100, 256),
		adaptiveThreshold:       NewAdaptiveThreshold(65.0, 40.0, 85.0, 0.01),
		trainingData:            make([][]float64, 0),
		trainingLabels:          make([]float64, 0),
	}
}

// ExtractEnhancedFeatures 提取所有高级特征
func (ebas *EnhancedBehaviorAnalysisService) ExtractEnhancedFeatures(points []BehaviorDataPoint, clicks []BehaviorDataPoint, keyStrokes []KeyboardDataPoint) *EnhancedFeatures {
	features := &EnhancedFeatures{}

	if len(points) < 3 {
		return features
	}

	// 1. 提取曲率特征
	features.CurvatureVariance, features.CurvaturePeaks, features.CurvatureEntropy = ebas.extractCurvatureFeatures(points)

	// 2. 提取点击间隔分布特征
	if len(clicks) >= 3 {
		features.ClickIntervalSkewness, features.ClickIntervalKurtosis, features.ClickIntervalQuantiles = ebas.extractClickIntervalFeatures(clicks)
	}

	// 3. 提取方向变化特征
	features.DirectionChangeFrequency, features.DirectionChangeRegularity = ebas.extractDirectionChangeFeatures(points)

	// 4. 提取傅里叶分析特征
	features.FourierDominantFrequency, features.FourierEnergyRatio = ebas.extractFourierFeatures(points)

	// 5. 提取分形特征
	features.FractalDimension = ebas.calculateFractalDimension(points)

	// 6. 提取悬停特征
	features.HoverCount, features.HoverDurationVariance = ebas.extractHoverFeatures(points)

	// 7. 提取键盘节奏特征
	if len(keyStrokes) >= 3 {
		features.KeyPressIntervalRegularity, features.KeyHoldDurationVariance = ebas.extractKeyboardRhythmFeatures(keyStrokes)
	}

	// 8. 提取停顿模式特征
	features.PauseIntervalDistribution, features.PauseFrequency = ebas.extractPausePatternFeatures(points)

	// 9. 提取加速度变化率特征
	features.JerkAverage, features.JerkVariance = ebas.extractJerkFeatures(points)

	// 10. 提取速度特征扩展（任务3.1&3.2）
	ebas.extractEnhancedSpeedFeatures(points, features)

	// 11. 提取抖动特征扩展（任务3.3）
	ebas.extractEnhancedJitterFeatures(points, features)

	// 12. 提取加速度特征扩展（任务3.1）
	ebas.extractEnhancedAccelerationFeatures(points, features)

	// 13. 提取时序特征扩展
	ebas.extractTemporalFeatures(points, features)

	// 14. 提取空间特征扩展
	ebas.extractSpatialFeatures(points, features)

	// 15. 提取统计特征扩展
	ebas.extractStatisticalFeatures(points, features)

	// 16. 识别人机差异特征（任务3.2）
	ebas.extractHumanMachineDifferenceFeatures(points, features)

	return features
}

// extractCurvatureFeatures 提取曲率特征
func (ebas *EnhancedBehaviorAnalysisService) extractCurvatureFeatures(points []BehaviorDataPoint) (variance float64, peaks int, entropy float64) {
	if len(points) < 3 {
		return 0, 0, 0
	}

	curvatures := make([]float64, 0)
	for i := 1; i < len(points)-1; i++ {
		curv := ebas.computeCurvature(points[i-1], points[i], points[i+1])
		curvatures = append(curvatures, math.Abs(curv))
	}

	if len(curvatures) == 0 {
		return 0, 0, 0
	}

	// 计算曲率方差
	mean := ebas.meanFloat(curvatures)
	variance = 0.0
	for _, c := range curvatures {
		variance += math.Pow(c-mean, 2)
	}
	variance /= float64(len(curvatures))

	// 计算曲率峰值
	peaks = 0
	for i := 1; i < len(curvatures)-1; i++ {
		if curvatures[i] > curvatures[i-1] && curvatures[i] > curvatures[i+1] && curvatures[i] > mean+math.Sqrt(variance) {
			peaks++
		}
	}

	// 计算曲率熵
	bins := 10
	histogram := make([]int, bins)
	minC, maxC := ebas.minFloat(curvatures), ebas.maxFloat(curvatures)
	if maxC > minC {
		for _, c := range curvatures {
			bin := int((c - minC) / (maxC - minC) * float64(bins-1))
			if bin >= bins {
				bin = bins - 1
			}
			if bin < 0 {
				bin = 0
			}
			histogram[bin]++
		}
		entropy = 0.0
		total := len(curvatures)
		for _, count := range histogram {
			if count > 0 {
				p := float64(count) / float64(total)
				entropy -= p * math.Log2(p)
			}
		}
	}

	return variance, peaks, entropy
}

// extractClickIntervalFeatures 提取点击间隔分布特征
func (ebas *EnhancedBehaviorAnalysisService) extractClickIntervalFeatures(clicks []BehaviorDataPoint) (skewness float64, kurtosis float64, quantiles []float64) {
	if len(clicks) < 3 {
		return 0, 0, nil
	}

	intervals := make([]float64, 0)
	for i := 1; i < len(clicks); i++ {
		interval := float64(clicks[i].Timestamp - clicks[i-1].Timestamp)
		intervals = append(intervals, interval)
	}

	if len(intervals) < 2 {
		return 0, 0, nil
	}

	mean := ebas.meanFloat(intervals)
	std := math.Sqrt(ebas.varianceFloat(intervals))

	// 计算偏度
	if std > 0 {
		sum := 0.0
		for _, x := range intervals {
			sum += math.Pow((x-mean)/std, 3)
		}
		skewness = sum / float64(len(intervals))

		// 计算峰度
		sum = 0.0
		for _, x := range intervals {
			sum += math.Pow((x-mean)/std, 4)
		}
		kurtosis = sum/float64(len(intervals)) - 3
	}

	// 计算分位数
	sorted := make([]float64, len(intervals))
	copy(sorted, intervals)
	sort.Float64s(sorted)
	quantiles = make([]float64, 5)
	if len(sorted) > 0 {
		for i := 0; i < 5; i++ {
			idx := int(float64(len(sorted)-1) * float64(i) / 4.0)
			quantiles[i] = sorted[idx]
		}
	}

	return skewness, kurtosis, quantiles
}

// extractDirectionChangeFeatures 提取方向变化特征
func (ebas *EnhancedBehaviorAnalysisService) extractDirectionChangeFeatures(points []BehaviorDataPoint) (frequency float64, regularity float64) {
	if len(points) < 3 {
		return 0, 0
	}

	directionChanges := make([]float64, 0)
	prevAngle := 0.0

	for i := 1; i < len(points); i++ {
		dx := float64(points[i].X - points[i-1].X)
		dy := float64(points[i].Y - points[i-1].Y)
		angle := math.Atan2(dy, dx)

		if i > 1 {
			angleDiff := math.Abs(angle - prevAngle)
			if angleDiff > math.Pi {
				angleDiff = 2*math.Pi - angleDiff
			}
			if angleDiff > 0.3 {
				directionChanges = append(directionChanges, float64(points[i].Timestamp))
			}
		}
		prevAngle = angle
	}

	if len(directionChanges) < 2 {
		return 0, 0
	}

	// 方向变化频率
	totalDuration := float64(points[len(points)-1].Timestamp - points[0].Timestamp)
	if totalDuration > 0 {
		frequency = float64(len(directionChanges)) / totalDuration * 1000
	}

	// 方向变化规律性
	intervals := make([]float64, 0)
	for i := 1; i < len(directionChanges); i++ {
		intervals = append(intervals, directionChanges[i]-directionChanges[i-1])
	}
	if len(intervals) > 0 {
		meanInterval := ebas.meanFloat(intervals)
		stdInterval := math.Sqrt(ebas.varianceFloat(intervals))
		if meanInterval > 0 {
			regularity = 1.0 - math.Min(stdInterval/meanInterval, 1.0)
		}
	}

	return frequency, regularity
}

// extractFourierFeatures 提取傅里叶分析特征
func (ebas *EnhancedBehaviorAnalysisService) extractFourierFeatures(points []BehaviorDataPoint) (dominantFreq float64, energyRatio float64) {
	if len(points) < 4 {
		return 0, 0
	}

	// 确保点数是2的幂次
	n := len(points)
	for n&(n-1) != 0 {
		n--
	}
	if n < 2 {
		return 0, 0
	}

	// 提取X和Y坐标序列
	x := make([]float64, n)
	y := make([]float64, n)
	for i := 0; i < n; i++ {
		x[i] = float64(points[i].X)
		y[i] = float64(points[i].Y)
	}

	// 对X序列进行FFT
	fftX := ebas.fft(x)
	fftY := ebas.fft(y)

	// 计算频谱幅度
	magnitudes := make([]float64, n/2)
	totalEnergy := 0.0
	for i := 0; i < n/2; i++ {
		magX := cmplx.Abs(fftX[i])
		magY := cmplx.Abs(fftY[i])
		magnitudes[i] = math.Sqrt(magX*magX + magY*magY)
		totalEnergy += magnitudes[i] * magnitudes[i]
	}

	// 找主频
	maxMag := 0.0
	dominantIdx := 0
	for i := 1; i < len(magnitudes); i++ {
		if magnitudes[i] > maxMag {
			maxMag = magnitudes[i]
			dominantIdx = i
		}
	}

	// 计算采样频率
	if n >= 2 {
		totalTime := float64(points[n-1].Timestamp - points[0].Timestamp)
		if totalTime > 0 {
			dominantFreq = float64(dominantIdx) / totalTime * 1000
		}
	}

	// 计算能量比（前10%频率的能量占比）
	if len(magnitudes) > 0 {
		top10Percent := int(float64(len(magnitudes)) * 0.1)
		if top10Percent < 1 {
			top10Percent = 1
		}

		// 对幅度排序
		sortedMags := make([]float64, len(magnitudes))
		copy(sortedMags, magnitudes)
		sort.Sort(sort.Reverse(sort.Float64Slice(sortedMags)))

		topEnergy := 0.0
		for i := 0; i < top10Percent && i < len(sortedMags); i++ {
			topEnergy += sortedMags[i] * sortedMags[i]
		}

		if totalEnergy > 0 {
			energyRatio = topEnergy / totalEnergy
		}
	}

	return dominantFreq, energyRatio
}

// fft 快速傅里叶变换（简化实现）
func (ebas *EnhancedBehaviorAnalysisService) fft(v []float64) []complex128 {
	n := len(v)
	if n <= 1 {
		result := make([]complex128, n)
		for i, val := range v {
			result[i] = complex(val, 0)
		}
		return result
	}

	even := make([]float64, n/2)
	odd := make([]float64, n/2)
	for i := 0; i < n/2; i++ {
		even[i] = v[2*i]
		odd[i] = v[2*i+1]
	}

	fftEven := ebas.fft(even)
	fftOdd := ebas.fft(odd)

	result := make([]complex128, n)
	for k := 0; k < n/2; k++ {
		t := cmplx.Exp(complex(0, -2*math.Pi*float64(k)/float64(n))) * fftOdd[k]
		result[k] = fftEven[k] + t
		result[k+n/2] = fftEven[k] - t
	}

	return result
}

// calculateFractalDimension 计算分形维数（盒计数法）
func (ebas *EnhancedBehaviorAnalysisService) calculateFractalDimension(points []BehaviorDataPoint) float64 {
	if len(points) < 10 {
		return 1.0
	}

	// 确定坐标范围
	minX, maxX := points[0].X, points[0].X
	minY, maxY := points[0].Y, points[0].Y
	for _, p := range points {
		if p.X < minX {
			minX = p.X
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}

	width := maxX - minX
	height := maxY - minY
	if width == 0 || height == 0 {
		return 1.0
	}

	// 盒计数法
	maxScale := 6
	logScales := make([]float64, maxScale)
	logCounts := make([]float64, maxScale)

	for scale := 0; scale < maxScale; scale++ {
		boxSize := int(math.Pow(2, float64(maxScale-scale)))
		grid := make(map[string]bool)

		for _, p := range points {
			gx := (p.X - minX) / boxSize
			gy := (p.Y - minY) / boxSize
			key := fmt.Sprintf("%d,%d", gx, gy)
			grid[key] = true
		}

		logScales[scale] = math.Log(1.0 / float64(boxSize))
		logCounts[scale] = math.Log(float64(len(grid)))
	}

	// 最小二乘法拟合直线求斜率（分形维数）
	return ebas.linearRegression(logScales, logCounts)
}

// linearRegression 简单线性回归，返回斜率
func (ebas *EnhancedBehaviorAnalysisService) linearRegression(x, y []float64) float64 {
	n := len(x)
	if n != len(y) || n < 2 {
		return 1.0
	}

	sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0
	for i := 0; i < n; i++ {
		sumX += x[i]
		sumY += y[i]
		sumXY += x[i] * y[i]
		sumX2 += x[i] * x[i]
	}

	denominator := float64(n)*sumX2 - sumX*sumX
	if denominator == 0 {
		return 1.0
	}

	return (float64(n)*sumXY - sumX*sumY) / denominator
}

// extractHoverFeatures 提取悬停特征
func (ebas *EnhancedBehaviorAnalysisService) extractHoverFeatures(points []BehaviorDataPoint) (count int, durationVariance float64) {
	if len(points) < 2 {
		return 0, 0
	}

	hoverDurations := make([]float64, 0)
	hoverStart := -1

	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		distance := math.Sqrt(float64(dx*dx + dy*dy))
		dt := points[i].Timestamp - points[i-1].Timestamp

		if distance < 3 && dt > 100 {
			if hoverStart == -1 {
				hoverStart = i - 1
			}
		} else {
			if hoverStart != -1 {
				duration := float64(points[i-1].Timestamp - points[hoverStart].Timestamp)
				hoverDurations = append(hoverDurations, duration)
				count++
				hoverStart = -1
			}
		}
	}

	// 检查末尾是否有悬停
	if hoverStart != -1 {
		duration := float64(points[len(points)-1].Timestamp - points[hoverStart].Timestamp)
		hoverDurations = append(hoverDurations, duration)
		count++
	}

	if len(hoverDurations) > 1 {
		durationVariance = ebas.varianceFloat(hoverDurations)
	}

	return count, durationVariance
}

// extractKeyboardRhythmFeatures 提取键盘节奏特征
func (ebas *EnhancedBehaviorAnalysisService) extractKeyboardRhythmFeatures(keyStrokes []KeyboardDataPoint) (regularity float64, holdVariance float64) {
	if len(keyStrokes) < 3 {
		return 0, 0
	}

	// 按键间隔规律性
	intervals := make([]float64, 0)
	for i := 1; i < len(keyStrokes); i++ {
		intervals = append(intervals, float64(keyStrokes[i].Timestamp-keyStrokes[i-1].Timestamp))
	}
	if len(intervals) > 0 {
		meanInterval := ebas.meanFloat(intervals)
		stdInterval := math.Sqrt(ebas.varianceFloat(intervals))
		if meanInterval > 0 {
			regularity = 1.0 - math.Min(stdInterval/meanInterval, 1.0)
		}
	}

	// 按键保持时长方差
	holdDurations := make([]float64, 0)
	for _, ks := range keyStrokes {
		if ks.HoldDuration > 0 {
			holdDurations = append(holdDurations, float64(ks.HoldDuration))
		}
	}
	if len(holdDurations) > 1 {
		holdVariance = ebas.varianceFloat(holdDurations)
	}

	return regularity, holdVariance
}

// extractPausePatternFeatures 提取停顿模式特征
func (ebas *EnhancedBehaviorAnalysisService) extractPausePatternFeatures(points []BehaviorDataPoint) (distribution []float64, frequency float64) {
	if len(points) < 3 {
		return nil, 0
	}

	pauseIntervals := make([]float64, 0)
	pauseCount := 0

	for i := 1; i < len(points); i++ {
		dt := points[i].Timestamp - points[i-1].Timestamp
		if dt > 300 {
			pauseIntervals = append(pauseIntervals, float64(dt))
			pauseCount++
		}
	}

	// 停顿间隔分布（分位数）
	if len(pauseIntervals) > 0 {
		sorted := make([]float64, len(pauseIntervals))
		copy(sorted, pauseIntervals)
		sort.Float64s(sorted)

		distribution = make([]float64, 3)
		if len(sorted) >= 3 {
			distribution[0] = sorted[int(float64(len(sorted))*0.25)]
			distribution[1] = sorted[int(float64(len(sorted))*0.5)]
			distribution[2] = sorted[int(float64(len(sorted))*0.75)]
		}
	}

	// 停顿频率
	totalDuration := float64(points[len(points)-1].Timestamp - points[0].Timestamp)
	if totalDuration > 0 {
		frequency = float64(pauseCount) / totalDuration * 1000
	}

	return distribution, frequency
}

// extractJerkFeatures 提取加加速度（加速度的变化率）特征
func (ebas *EnhancedBehaviorAnalysisService) extractJerkFeatures(points []BehaviorDataPoint) (avgJerk float64, varianceJerk float64) {
	if len(points) < 4 {
		return 0, 0
	}

	// 先计算速度
	speeds := make([]float64, 0)
	for i := 1; i < len(points); i++ {
		dx := float64(points[i].X - points[i-1].X)
		dy := float64(points[i].Y - points[i-1].Y)
		distance := math.Sqrt(dx*dx + dy*dy)
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			speeds = append(speeds, distance/dt)
		}
	}

	// 再计算加速度
	accelerations := make([]float64, 0)
	for i := 1; i < len(speeds); i++ {
		dt := float64(points[i+1].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			accel := (speeds[i] - speeds[i-1]) / dt
			accelerations = append(accelerations, accel)
		}
	}

	// 最后计算加加速度（jerk）
	jerks := make([]float64, 0)
	for i := 1; i < len(accelerations); i++ {
		dt := float64(points[i+2].Timestamp - points[i].Timestamp)
		if dt > 0 {
			jerk := (accelerations[i] - accelerations[i-1]) / dt
			jerks = append(jerks, math.Abs(jerk))
		}
	}

	if len(jerks) == 0 {
		return 0, 0
	}

	avgJerk = ebas.meanFloat(jerks)
	varianceJerk = ebas.varianceFloat(jerks)

	return avgJerk, varianceJerk
}

// extractEnhancedSpeedFeatures 提取增强速度特征（任务3.1&3.2）
func (ebas *EnhancedBehaviorAnalysisService) extractEnhancedSpeedFeatures(points []BehaviorDataPoint, features *EnhancedFeatures) {
	if len(points) < 3 {
		return
	}

	// 计算速度序列
	speeds := ebas.calculateSpeedSequence(points)
	if len(speeds) < 3 {
		return
	}

	// 速度统计特征
	sortedSpeeds := make([]float64, len(speeds))
	copy(sortedSpeeds, speeds)
	sort.Float64s(sortedSpeeds)

	// 中位数
	features.SpeedMedian = sortedSpeeds[len(sortedSpeeds)/2]

	// 分位数
	features.SpeedPercentile25 = sortedSpeeds[len(sortedSpeeds)/4]
	features.SpeedPercentile75 = sortedSpeeds[3*len(sortedSpeeds)/4]
	features.SpeedPercentile90 = sortedSpeeds[int(float64(len(sortedSpeeds))*0.9)]

	// 速度范围和IQR
	features.SpeedRange = sortedSpeeds[len(sortedSpeeds)-1] - sortedSpeeds[0]
	features.SpeedIQR = features.SpeedPercentile75 - features.SpeedPercentile25

	// 速度偏度和峰度
	meanSpeed := ebas.meanFloat(speeds)
	stdSpeed := math.Sqrt(ebas.varianceFloat(speeds))
	if stdSpeed > 0 {
		sum := 0.0
		for _, s := range speeds {
			sum += math.Pow((s-meanSpeed)/stdSpeed, 3)
		}
		features.SpeedSkewness = sum / float64(len(speeds))

		sum = 0.0
		for _, s := range speeds {
			sum += math.Pow((s-meanSpeed)/stdSpeed, 4)
		}
		features.SpeedKurtosis = sum/float64(len(speeds)) - 3
	}

	// 速度变异系数
	if meanSpeed > 0 {
		features.SpeedCoefficientVariation = stdSpeed / meanSpeed
	}

	// 匀速运动比例（速度变化小于10%的区间）
	uniformCount := 0
	for i := 1; i < len(speeds); i++ {
		if meanSpeed > 0 && math.Abs(speeds[i]-speeds[i-1])/meanSpeed < 0.1 {
			uniformCount++
		}
	}
	features.UniformMotionRatio = float64(uniformCount) / float64(len(speeds)-1)

	// 计算加速度序列
	accelerations := ebas.calculateAccelerationSequence(points)

	// 加速度方差、偏度、峰度
	if len(accelerations) > 0 {
		features.AccelerationVariance = ebas.varianceFloat(accelerations)
		meanAccel := ebas.meanFloat(accelerations)
		stdAccel := math.Sqrt(features.AccelerationVariance)
		if stdAccel > 0 {
			sum := 0.0
			for _, a := range accelerations {
				sum += math.Pow((a-meanAccel)/stdAccel, 3)
			}
			features.AccelerationSkewness = sum / float64(len(accelerations))

			sum = 0.0
			for _, a := range accelerations {
				sum += math.Pow((a-meanAccel)/stdAccel, 4)
			}
			features.AccelerationKurtosis = sum/float64(len(accelerations)) - 3
		}
	}

	// 人机速度差异指标
	features.HumanMachineSpeedDiff = ebas.calculateHumanMachineSpeedDifference(speeds)

	// 速度曲线复杂度
	features.SpeedCurveComplexity = ebas.calculateSpeedCurveComplexity(speeds)
}

// calculateSpeedSequence 计算速度序列
func (ebas *EnhancedBehaviorAnalysisService) calculateSpeedSequence(points []BehaviorDataPoint) []float64 {
	speeds := make([]float64, 0)
	for i := 1; i < len(points); i++ {
		dx := float64(points[i].X - points[i-1].X)
		dy := float64(points[i].Y - points[i-1].Y)
		distance := math.Sqrt(dx*dx + dy*dy)
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			speeds = append(speeds, distance/dt)
		}
	}
	return speeds
}

// calculateAccelerationSequence 计算加速度序列
func (ebas *EnhancedBehaviorAnalysisService) calculateAccelerationSequence(points []BehaviorDataPoint) []float64 {
	speeds := ebas.calculateSpeedSequence(points)
	accelerations := make([]float64, 0)
	for i := 1; i < len(speeds); i++ {
		dt := float64(points[i+1].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			accel := (speeds[i] - speeds[i-1]) / dt
			accelerations = append(accelerations, accel)
		}
	}
	return accelerations
}

// calculateHumanMachineSpeedDifference 计算人机速度差异指标
func (ebas *EnhancedBehaviorAnalysisService) calculateHumanMachineSpeedDifference(speeds []float64) float64 {
	if len(speeds) < 3 {
		return 0.5
	}

	meanSpeed := ebas.meanFloat(speeds)
	speedVariance := ebas.varianceFloat(speeds)

	// 人类特征：速度有变化，但不是完全规律
	// 机器人特征：速度过于规律或完全恒定
	humanScore := 0.5

	// 速度变化性评分（人类应该有适度的变化）
	normalizedVariance := speedVariance / (meanSpeed*meanSpeed + 0.001)
	if normalizedVariance > 0.01 && normalizedVariance < 1.0 {
		humanScore += 0.2
	}

	// 速度分布评分（人类速度分布应该接近正态分布）
	sortedSpeeds := make([]float64, len(speeds))
	copy(sortedSpeeds, speeds)
	sort.Float64s(sortedSpeeds)
	skewness := 0.0
	if len(sortedSpeeds) > 2 {
		median := sortedSpeeds[len(sortedSpeeds)/2]
		Q1 := sortedSpeeds[len(sortedSpeeds)/4]
		Q3 := sortedSpeeds[3*len(sortedSpeeds)/4]
		if (Q3 - Q1) > 0 {
			skewness = (median - meanSpeed) / (Q3 - Q1)
		}
		if math.Abs(skewness) < 0.5 {
			humanScore += 0.15
		}
	}

	// 速度连续性评分（人类移动不会完全平滑）
	smoothness := 0.0
	for i := 1; i < len(speeds); i++ {
		smoothness += math.Abs(speeds[i] - speeds[i-1])
	}
	smoothness /= float64(len(speeds)-1) * meanSpeed
	if smoothness > 0.05 && smoothness < 0.5 {
		humanScore += 0.15
	}

	return math.Max(0, math.Min(1, humanScore))
}

// calculateSpeedCurveComplexity 计算速度曲线复杂度
func (ebas *EnhancedBehaviorAnalysisService) calculateSpeedCurveComplexity(speeds []float64) float64 {
	if len(speeds) < 4 {
		return 0.0
	}

	// 使用熵来度量速度曲线的复杂度
	bins := 10
	histogram := make([]int, bins)
	minS, maxS := ebas.minFloat(speeds), ebas.maxFloat(speeds)

	if maxS <= minS {
		return 0.0
	}

	for _, s := range speeds {
		bin := int((s - minS) / (maxS - minS) * float64(bins-1))
		if bin >= bins {
			bin = bins - 1
		}
		if bin < 0 {
			bin = 0
		}
		histogram[bin]++
	}

	entropy := 0.0
	total := len(speeds)
	for _, count := range histogram {
		if count > 0 {
			p := float64(count) / float64(total)
			entropy -= p * math.Log2(p)
		}
	}

	// 归一化熵（最大熵为log2(bins)）
	maxEntropy := math.Log2(float64(bins))
	if maxEntropy > 0 {
		return entropy / maxEntropy
	}

	return 0.0
}

// extractEnhancedJitterFeatures 提取增强抖动特征（任务3.3）
func (ebas *EnhancedBehaviorAnalysisService) extractEnhancedJitterFeatures(points []BehaviorDataPoint, features *EnhancedFeatures) {
	if len(points) < 3 {
		return
	}

	// 计算速度序列
	speeds := ebas.calculateSpeedSequence(points)
	if len(speeds) < 2 {
		return
	}

	// 计算抖动（速度变化的绝对值）
	jitterValues := make([]float64, 0)
	for i := 1; i < len(speeds); i++ {
		jitter := math.Abs(speeds[i] - speeds[i-1])
		jitterValues = append(jitterValues, jitter)
	}

	if len(jitterValues) == 0 {
		return
	}

	// 抖动频率（单位时间内的抖动次数）
	totalDuration := float64(points[len(points)-1].Timestamp - points[0].Timestamp)
	if totalDuration > 0 {
		features.JitterFrequency = float64(len(jitterValues)) / totalDuration * 1000
	}

	// 抖动幅度统计
	features.JitterAmplitudeMean = ebas.meanFloat(jitterValues)
	features.JitterAmplitudeMax = ebas.maxFloat(jitterValues)
	features.JitterAmplitudeVariance = ebas.varianceFloat(jitterValues)

	// 微抖动识别（抖动幅度小于均值的50%）
	microJitterThreshold := features.JitterAmplitudeMean * 0.5
	microJitterCount := 0
	for _, j := range jitterValues {
		if j <= microJitterThreshold && j > 0 {
			microJitterCount++
		}
	}
	features.MicroJitterCount = microJitterCount
	features.MicroJitterRatio = float64(microJitterCount) / float64(len(jitterValues))

	// 抖动规律性
	regularity := 1.0 - math.Min(features.JitterAmplitudeVariance/(features.JitterAmplitudeMean+0.001), 1.0)
	features.JitterRegularity = regularity

	// 抖动熵
	features.JitterEntropy = ebas.calculateJitterEntropy(jitterValues)

	// 抖动聚类数（使用简单的聚类方法）
	features.JitterClusterCount = ebas.clusterJitterValues(jitterValues)

	// 抖动波形类型识别
	features.JitterWaveformType = ebas.identifyJitterWaveformType(jitterValues)
}

// calculateJitterEntropy 计算抖动熵
func (ebas *EnhancedBehaviorAnalysisService) calculateJitterEntropy(jitterValues []float64) float64 {
	if len(jitterValues) < 2 {
		return 0.0
	}

	bins := 8
	histogram := make([]int, bins)
	minJ, maxJ := ebas.minFloat(jitterValues), ebas.maxFloat(jitterValues)

	if maxJ <= minJ {
		return 0.0
	}

	for _, j := range jitterValues {
		bin := int((j - minJ) / (maxJ - minJ) * float64(bins-1))
		if bin >= bins {
			bin = bins - 1
		}
		if bin < 0 {
			bin = 0
		}
		histogram[bin]++
	}

	entropy := 0.0
	total := len(jitterValues)
	for _, count := range histogram {
		if count > 0 {
			p := float64(count) / float64(total)
			entropy -= p * math.Log2(p)
		}
	}

	maxEntropy := math.Log2(float64(bins))
	if maxEntropy > 0 {
		return entropy / maxEntropy
	}

	return 0.0
}

// clusterJitterValues 抖动值聚类（简化实现）
func (ebas *EnhancedBehaviorAnalysisService) clusterJitterValues(jitterValues []float64) int {
	if len(jitterValues) < 3 {
		return 1
	}

	// 简单的基于分位数的聚类
	sorted := make([]float64, len(jitterValues))
	copy(sorted, jitterValues)
	sort.Float64s(sorted)

	Q1 := sorted[len(sorted)/4]
	Q2 := sorted[len(sorted)/2]
	Q3 := sorted[3*len(sorted)/4]

	clusters := 1
	if Q1 > 0 {
		clusters++
	}
	if Q3 > Q1*1.5 {
		clusters++
	}

	return clusters
}

// identifyJitterWaveformType 识别抖动波形类型
func (ebas *EnhancedBehaviorAnalysisService) identifyJitterWaveformType(jitterValues []float64) string {
	if len(jitterValues) < 5 {
		return "unknown"
	}

	// 计算导数来识别波形类型
	derivatives := make([]float64, 0)
	for i := 1; i < len(jitterValues); i++ {
		derivatives = append(derivatives, jitterValues[i]-jitterValues[i-1])
	}

	positiveCount := 0
	negativeCount := 0
	for _, d := range derivatives {
		if d > 0 {
			positiveCount++
		} else if d < 0 {
			negativeCount++
		}
	}

	positiveRatio := float64(positiveCount) / float64(len(derivatives))
	negativeRatio := float64(negativeCount) / float64(len(derivatives))

	if positiveRatio > 0.7 {
		return "increasing"
	} else if negativeRatio > 0.7 {
		return "decreasing"
	} else if positiveRatio > 0.4 && negativeRatio > 0.4 {
		return "oscillating"
	} else {
		return "random"
	}
}

// extractEnhancedAccelerationFeatures 提取增强加速度特征（任务3.1）
func (ebas *EnhancedBehaviorAnalysisService) extractEnhancedAccelerationFeatures(points []BehaviorDataPoint, features *EnhancedFeatures) {
	if len(points) < 3 {
		return
	}

	// 计算加速度序列
	accelerations := ebas.calculateAccelerationSequence(points)
	if len(accelerations) < 2 {
		return
	}

	// 加速度均值和范围
	features.AccelerationMean = ebas.meanFloat(accelerations)
	sortedAccel := make([]float64, len(accelerations))
	copy(sortedAccel, accelerations)
	sort.Float64s(sortedAccel)
	features.AccelerationRange = sortedAccel[len(sortedAccel)-1] - sortedAccel[0]

	// 减速比例（加速度为负值的比例）
	decelCount := 0
	for _, a := range accelerations {
		if a < 0 {
			decelCount++
		}
	}
	features.DecelerationRatio = float64(decelCount) / float64(len(accelerations))

	// 加速度峰值数量
	peakCount := 0
	for i := 1; i < len(accelerations)-1; i++ {
		mean := ebas.meanFloat(accelerations)
		std := math.Sqrt(ebas.varianceFloat(accelerations))
		if accelerations[i] > mean+std && accelerations[i] > accelerations[i-1] && accelerations[i] > accelerations[i+1] {
			peakCount++
		}
	}
	features.AccelerationPeakCount = peakCount

	// 加速度过零点数量
	zeroCrossings := 0
	for i := 1; i < len(accelerations); i++ {
		if (accelerations[i] > 0 && accelerations[i-1] < 0) || (accelerations[i] < 0 && accelerations[i-1] > 0) {
			zeroCrossings++
		}
	}
	features.AccelerationZeroCrossing = zeroCrossings

	// 加速度能量
	energy := 0.0
	for _, a := range accelerations {
		energy += a * a
	}
	features.AccelerationEnergy = energy

	// 切向加速度和法向加速度
	tangentialAccel, normalAccel := ebas.calculateTangentialAndNormalAcceleration(points)
	features.TangentialAcceleration = tangentialAccel
	features.NormalAcceleration = normalAccel
}

// calculateTangentialAndNormalAcceleration 计算切向和法向加速度
func (ebas *EnhancedBehaviorAnalysisService) calculateTangentialAndNormalAcceleration(points []BehaviorDataPoint) (tangential, normal float64) {
	if len(points) < 3 {
		return 0, 0
	}

	tangentialSum := 0.0
	normalSum := 0.0
	count := 0

	for i := 1; i < len(points)-1; i++ {
		// 计算速度向量
		vx1 := float64(points[i].X - points[i-1].X)
		vy1 := float64(points[i].Y - points[i-1].Y)
		vx2 := float64(points[i+1].X - points[i].X)
		vy2 := float64(points[i+1].Y - points[i].Y)

		// 切向加速度（速度大小变化）
		dt1 := float64(points[i].Timestamp - points[i-1].Timestamp)
		dt2 := float64(points[i+1].Timestamp - points[i].Timestamp)
		if dt1 > 0 && dt2 > 0 {
			speed1 := math.Sqrt(vx1*vx1+vy1*vy1) / dt1
			speed2 := math.Sqrt(vx2*vx2+vy2*vy2) / dt2
			tangentialSum += math.Abs(speed2 - speed1)
		}

		// 法向加速度（速度方向变化）
		if (vx1 != 0 || vy1 != 0) && (vx2 != 0 || vy2 != 0) {
			dot := vx1*vx2 + vy1*vy2
			mag1 := math.Sqrt(vx1*vx1 + vy1*vy1)
			mag2 := math.Sqrt(vx2*vx2 + vy2*vy2)
			if mag1 > 0 && mag2 > 0 {
				cosAngle := dot / (mag1 * mag2)
				if cosAngle > 1 {
					cosAngle = 1
				}
				if cosAngle < -1 {
					cosAngle = -1
				}
				angle := math.Acos(cosAngle)
				normalSum += math.Abs(angle)
			}
		}
		count++
	}

	if count > 0 {
		tangential = tangentialSum / float64(count)
		normal = normalSum / float64(count)
	}

	return tangential, normal
}

// extractTemporalFeatures 提取时序特征扩展
func (ebas *EnhancedBehaviorAnalysisService) extractTemporalFeatures(points []BehaviorDataPoint, features *EnhancedFeatures) {
	if len(points) < 2 {
		return
	}

	// 计算点间时间间隔
	timeIntervals := make([]float64, 0)
	for i := 1; i < len(points); i++ {
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		timeIntervals = append(timeIntervals, dt)
	}

	if len(timeIntervals) == 0 {
		return
	}

	// 时序统计特征
	features.InterPointTimeMean = ebas.meanFloat(timeIntervals)
	features.InterPointTimeVariance = ebas.varianceFloat(timeIntervals)

	meanInterval := features.InterPointTimeMean
	stdInterval := math.Sqrt(features.InterPointTimeVariance)
	if stdInterval > 0 && meanInterval > 0 {
		// 偏度
		sum := 0.0
		for _, t := range timeIntervals {
			sum += math.Pow((t-meanInterval)/stdInterval, 3)
		}
		features.InterPointTimeSkewness = sum / float64(len(timeIntervals))

		// 变异系数
		features.InterPointTimeCV = stdInterval / meanInterval
	}

	// 时间规律性指数（1 - CV）
	features.TimeRegularityIndex = 1.0 - math.Min(features.InterPointTimeCV, 1.0)

	// 时间突发性指数（使用峰度来度量）
	if len(timeIntervals) > 2 {
		sorted := make([]float64, len(timeIntervals))
		copy(sorted, timeIntervals)
		sort.Float64s(sorted)
		Q1 := sorted[len(sorted)/4]
		Q3 := sorted[3*len(sorted)/4]
		if Q1 > 0 {
			features.TimeBurstinessIndex = (Q3 - Q1) / Q1
		}
	}

	// 长停顿和短停顿统计（使用300ms作为阈值）
	longPauseThreshold := 300.0
	shortPauseThreshold := 50.0

	longPauseCount := 0
	shortPauseCount := 0
	for _, t := range timeIntervals {
		if t > longPauseThreshold {
			longPauseCount++
		} else if t < shortPauseThreshold {
			shortPauseCount++
		}
	}

	features.LongPauseCount = longPauseCount
	features.LongPauseRatio = float64(longPauseCount) / float64(len(timeIntervals))
	features.ShortPauseCount = shortPauseCount
	features.ShortPauseRatio = float64(shortPauseCount) / float64(len(timeIntervals))

	// 时间序列熵
	features.TimeSequenceEntropy = ebas.calculateTimeSequenceEntropy(timeIntervals)

	// 时间序列模式
	features.TimeSequencePattern = ebas.identifyTimeSequencePattern(timeIntervals)
}

// calculateTimeSequenceEntropy 计算时间序列熵
func (ebas *EnhancedBehaviorAnalysisService) calculateTimeSequenceEntropy(timeIntervals []float64) float64 {
	if len(timeIntervals) < 2 {
		return 0.0
	}

	bins := 8
	histogram := make([]int, bins)
	minT, maxT := ebas.minFloat(timeIntervals), ebas.maxFloat(timeIntervals)

	if maxT <= minT {
		return 0.0
	}

	for _, t := range timeIntervals {
		bin := int((t - minT) / (maxT - minT) * float64(bins-1))
		if bin >= bins {
			bin = bins - 1
		}
		if bin < 0 {
			bin = 0
		}
		histogram[bin]++
	}

	entropy := 0.0
	total := len(timeIntervals)
	for _, count := range histogram {
		if count > 0 {
			p := float64(count) / float64(total)
			entropy -= p * math.Log2(p)
		}
	}

	maxEntropy := math.Log2(float64(bins))
	if maxEntropy > 0 {
		return entropy / maxEntropy
	}

	return 0.0
}

// identifyTimeSequencePattern 识别时间序列模式
func (ebas *EnhancedBehaviorAnalysisService) identifyTimeSequencePattern(timeIntervals []float64) string {
	if len(timeIntervals) < 5 {
		return "insufficient_data"
	}

	// 检查趋势
	increasing := 0
	decreasing := 0
	for i := 1; i < len(timeIntervals); i++ {
		if timeIntervals[i] > timeIntervals[i-1] {
			increasing++
		} else if timeIntervals[i] < timeIntervals[i-1] {
			decreasing++
		}
	}

	total := len(timeIntervals) - 1
	increasingRatio := float64(increasing) / float64(total)
	decreasingRatio := float64(decreasing) / float64(total)

	if increasingRatio > 0.6 {
		return "accelerating"
	} else if decreasingRatio > 0.6 {
		return "decelerating"
	} else if increasingRatio > 0.4 && decreasingRatio > 0.4 {
		return "variable"
	} else {
		return "steady"
	}
}

// extractSpatialFeatures 提取空间特征扩展
func (ebas *EnhancedBehaviorAnalysisService) extractSpatialFeatures(points []BehaviorDataPoint, features *EnhancedFeatures) {
	if len(points) < 2 {
		return
	}

	// 计算总位移和净位移
	totalDistance := 0.0
	for i := 1; i < len(points); i++ {
		dx := float64(points[i].X - points[i-1].X)
		dy := float64(points[i].Y - points[i-1].Y)
		totalDistance += math.Sqrt(dx*dx + dy*dy)
	}

	startX, startY := float64(points[0].X), float64(points[0].Y)
	endX, endY := float64(points[len(points)-1].X), float64(points[len(points)-1].Y)
	netDisplacement := math.Sqrt((endX-startX)*(endX-startX) + (endY-startY)*(endY-startY))

	features.TotalDisplacement = totalDistance
	features.NetDisplacement = netDisplacement

	if totalDistance > 0 {
		features.DisplacementRatio = netDisplacement / totalDistance
	}

	// 计算覆盖面积（使用边界框）
	minX, maxX := points[0].X, points[0].X
	minY, maxY := points[0].Y, points[0].Y
	for _, p := range points {
		if p.X < minX {
			minX = p.X
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}

	area := float64((maxX - minX) * (maxY - minY))
	features.AreaCovered = area

	if totalDistance > 0 {
		features.AreaPerDistance = area / totalDistance
	}

	// 空间集中度（轨迹点到质心的平均距离）
	centroidX := 0.0
	centroidY := 0.0
	for _, p := range points {
		centroidX += float64(p.X)
		centroidY += float64(p.Y)
	}
	centroidX /= float64(len(points))
	centroidY /= float64(len(points))

	distancesToCentroid := make([]float64, 0)
	for _, p := range points {
		dx := float64(p.X) - centroidX
		dy := float64(p.Y) - centroidY
		distancesToCentroid = append(distancesToCentroid, math.Sqrt(dx*dx+dy*dy))
	}

	features.SpatialConcentration = ebas.meanFloat(distancesToCentroid)
	features.SpatialSpread = ebas.varianceFloat(distancesToCentroid)

	// 轨迹粗糙度
	features.TrajectoryRoughness = ebas.calculateTrajectoryRoughness(points)

	// 轨迹自相似性
	features.TrajectorySelfSimilarity = ebas.calculateTrajectorySelfSimilarity(points)

	// 路径蜿蜒度
	if totalDistance > 0 && netDisplacement > 0 {
		features.PathSinuosity = totalDistance / netDisplacement
	}
}

// calculateTrajectoryRoughness 计算轨迹粗糙度
func (ebas *EnhancedBehaviorAnalysisService) calculateTrajectoryRoughness(points []BehaviorDataPoint) float64 {
	if len(points) < 3 {
		return 0.0
	}

	roughness := 0.0
	count := 0

	for i := 1; i < len(points)-1; i++ {
		dx1 := float64(points[i].X - points[i-1].X)
		dy1 := float64(points[i].Y - points[i-1].Y)
		dx2 := float64(points[i+1].X - points[i].X)
		dy2 := float64(points[i+1].Y - points[i].Y)

		mag1 := math.Sqrt(dx1*dx1 + dy1*dy1)
		mag2 := math.Sqrt(dx2*dx2 + dy2*dy2)

		if mag1 > 0 && mag2 > 0 {
			dot := dx1*dx2 + dy1*dy2
			cosAngle := dot / (mag1 * mag2)
			if cosAngle > 1 {
				cosAngle = 1
			}
			if cosAngle < -1 {
				cosAngle = -1
			}
			roughness += 1.0 - cosAngle
			count++
		}
	}

	if count > 0 {
		return roughness / float64(count)
	}

	return 0.0
}

// calculateTrajectorySelfSimilarity 计算轨迹自相似性
func (ebas *EnhancedBehaviorAnalysisService) calculateTrajectorySelfSimilarity(points []BehaviorDataPoint) float64 {
	if len(points) < 6 {
		return 0.0
	}

	// 简化的自相似性：比较前半段和后半段
	mid := len(points) / 2
	firstHalf := points[:mid]
	secondHalf := points[mid:]

	// 计算两段的平均曲率
	firstCurvature := ebas.meanFloat(ebas.calculateCurvatures(firstHalf))
	secondCurvature := ebas.meanFloat(ebas.calculateCurvatures(secondHalf))

	// 自相似性 = 1 - 归一化差异
	if firstCurvature+secondCurvature > 0 {
		return 1.0 - math.Abs(firstCurvature-secondCurvature)/(firstCurvature+secondCurvature)
	}

	return 0.0
}

// calculateCurvatures 计算曲率序列
func (ebas *EnhancedBehaviorAnalysisService) calculateCurvatures(points []BehaviorDataPoint) []float64 {
	curvatures := make([]float64, 0)
	for i := 1; i < len(points)-1; i++ {
		curv := ebas.computeCurvature(points[i-1], points[i], points[i+1])
		curvatures = append(curvatures, math.Abs(curv))
	}
	return curvatures
}

// extractStatisticalFeatures 提取统计特征扩展
func (ebas *EnhancedBehaviorAnalysisService) extractStatisticalFeatures(points []BehaviorDataPoint, features *EnhancedFeatures) {
	if len(points) < 2 {
		return
	}

	features.PointCount = len(points)

	// 点间隔统计
	timeIntervals := make([]float64, 0)
	for i := 1; i < len(points); i++ {
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		timeIntervals = append(timeIntervals, dt)
	}

	if len(timeIntervals) > 0 {
		features.MeanPointInterval = ebas.meanFloat(timeIntervals)
		features.PointIntervalConsistency = 1.0 - math.Min(ebas.varianceFloat(timeIntervals)/(features.MeanPointInterval*features.MeanPointInterval+0.001), 1.0)
	}

	// 轨迹持续时间
	if len(points) > 1 {
		features.TrajectoryDuration = float64(points[len(points)-1].Timestamp - points[0].Timestamp)
	}

	// 速度自相关系数
	speeds := ebas.calculateSpeedSequence(points)
	if len(speeds) > 3 {
		features.VelocityAutoCorrelation = ebas.calculateAutocorrelation(speeds, 1)
		features.VelocityLaggedCorrelation = ebas.calculateAutocorrelation(speeds, 2)
	}

	// 加速度自相关系数
	accelerations := ebas.calculateAccelerationSequence(points)
	if len(accelerations) > 3 {
		features.AccelerationAutoCorr = ebas.calculateAutocorrelation(accelerations, 1)
	}

	// 运动节奏评分
	features.MovementRhythmScore = ebas.calculateMovementRhythmScore(speeds, timeIntervals)
}

// calculateAutocorrelation 计算自相关系数
func (ebas *EnhancedBehaviorAnalysisService) calculateAutocorrelation(values []float64, lag int) float64 {
	if len(values) < lag+2 {
		return 0.0
	}

	mean := ebas.meanFloat(values)
	variance := ebas.varianceFloat(values)

	if variance == 0 {
		return 0.0
	}

	covariance := 0.0
	count := 0
	for i := 0; i < len(values)-lag; i++ {
		covariance += (values[i] - mean) * (values[i+lag] - mean)
		count++
	}

	if count > 0 {
		return covariance / (float64(count) * variance)
	}

	return 0.0
}

// calculateMovementRhythmScore 计算运动节奏评分
func (ebas *EnhancedBehaviorAnalysisService) calculateMovementRhythmScore(speeds []float64, timeIntervals []float64) float64 {
	if len(speeds) < 2 || len(timeIntervals) < 2 {
		return 0.5
	}

	// 节奏评分基于速度变化和时间间隔的一致性
	speedCV := math.Sqrt(ebas.varianceFloat(speeds)) / (ebas.meanFloat(speeds) + 0.001)
	timeCV := math.Sqrt(ebas.varianceFloat(timeIntervals)) / (ebas.meanFloat(timeIntervals) + 0.001)

	// 人类有适度的节奏变化
	rhythmScore := 0.5
	if speedCV > 0.1 && speedCV < 1.0 {
		rhythmScore += 0.2
	}
	if timeCV > 0.1 && timeCV < 1.0 {
		rhythmScore += 0.2
	}

	return math.Max(0, math.Min(1, rhythmScore))
}

// extractHumanMachineDifferenceFeatures 识别人机差异特征（任务3.2）
func (ebas *EnhancedBehaviorAnalysisService) extractHumanMachineDifferenceFeatures(points []BehaviorDataPoint, features *EnhancedFeatures) {
	if len(points) < 3 {
		return
	}

	speeds := ebas.calculateSpeedSequence(points)
	if len(speeds) < 2 {
		return
	}

	// 人类相似度评分
	features.HumanLikenessScore = ebas.calculateHumanLikenessScore(points, speeds)

	// 机械模式评分
	features.MechanicalPatternScore = ebas.calculateMechanicalPatternScore(speeds)

	// 自然波动评分
	features.NaturalFluctuationScore = ebas.calculateNaturalFluctuationScore(speeds)

	// 精准度评分
	features.PrecisionScore = ebas.calculatePrecisionScore(points)

	// 一致性评分
	features.ConsistencyScore = ebas.calculateConsistencyScore(speeds)
}

// calculateHumanLikenessScore 计算人类相似度评分
func (ebas *EnhancedBehaviorAnalysisService) calculateHumanLikenessScore(points []BehaviorDataPoint, speeds []float64) float64 {
	score := 0.5

	// 1. 速度变化性（人类应该有适度变化）
	speedVariance := ebas.varianceFloat(speeds)
	meanSpeed := ebas.meanFloat(speeds)
	if meanSpeed > 0 {
		cv := math.Sqrt(speedVariance) / meanSpeed
		if cv > 0.1 && cv < 1.0 {
			score += 0.15
		}
	}

	// 2. 停顿模式（人类会有自然停顿）
	pauseRatio := features.LongPauseRatio + features.ShortPauseRatio
	if pauseRatio > 0.05 && pauseRatio < 0.5 {
		score += 0.15
	}

	// 3. 轨迹不规则性（人类轨迹不会太规则）
	roughness := ebas.calculateTrajectoryRoughness(points)
	if roughness > 0.1 && roughness < 0.8 {
		score += 0.1
	}

	// 4. 时间间隔变化
	timeIntervals := make([]float64, 0)
	for i := 1; i < len(points); i++ {
		timeIntervals = append(timeIntervals, float64(points[i].Timestamp-points[i-1].Timestamp))
	}
	if len(timeIntervals) > 2 {
		timeCV := math.Sqrt(ebas.varianceFloat(timeIntervals)) / (ebas.meanFloat(timeIntervals) + 0.001)
		if timeCV > 0.2 && timeCV < 2.0 {
			score += 0.1
		}
	}

	return math.Max(0, math.Min(1, score))
}

// calculateMechanicalPatternScore 计算机械模式评分
func (ebas *EnhancedBehaviorAnalysisService) calculateMechanicalPatternScore(speeds []float64) float64 {
	score := 0.0

	if len(speeds) < 3 {
		return 0.5
	}

	// 1. 速度恒定性（机械移动倾向于恒定速度）
	speedVariance := ebas.varianceFloat(speeds)
	meanSpeed := ebas.meanFloat(speeds)
	if meanSpeed > 0 {
		cv := math.Sqrt(speedVariance) / meanSpeed
		if cv < 0.1 {
			score += 0.3
		}
	}

	// 2. 速度模式规律性
	sortedSpeeds := make([]float64, len(speeds))
	copy(sortedSpeeds, speeds)
	sort.Float64s(sortedSpeeds)
	Q1 := sortedSpeeds[len(sortedSpeeds)/4]
	Q3 := sortedSpeeds[3*len(sortedSpeeds)/4]
	if Q1 > 0 && Q3/Q1 < 1.5 {
		score += 0.2
	}

	// 3. 速度变化平滑性
	smoothness := 0.0
	for i := 1; i < len(speeds); i++ {
		smoothness += math.Abs(speeds[i] - speeds[i-1])
	}
	smoothness /= float64(len(speeds)-1) * meanSpeed
	if smoothness < 0.1 {
		score += 0.2
	}

	// 4. 速度自相关性（机械移动倾向于高自相关）
	autocorr := ebas.calculateAutocorrelation(speeds, 1)
	if autocorr > 0.8 {
		score += 0.3
	}

	return math.Max(0, math.Min(1, score))
}

// calculateNaturalFluctuationScore 计算自然波动评分
func (ebas *EnhancedBehaviorAnalysisService) calculateNaturalFluctuationScore(speeds []float64) float64 {
	if len(speeds) < 3 {
		return 0.5
	}

	// 自然波动：使用速度的二阶差分来分析波动
	secondDiff := make([]float64, 0)
	for i := 2; i < len(speeds); i++ {
		diff := speeds[i] - 2*speeds[i-1] + speeds[i-2]
		secondDiff = append(secondDiff, math.Abs(diff))
	}

	if len(secondDiff) == 0 {
		return 0.5
	}

	meanFluctuation := ebas.meanFloat(secondDiff)
	fluctuationVariance := ebas.varianceFloat(secondDiff)

	// 自然波动应该有适度的变化
	score := 0.5
	if fluctuationVariance > meanFluctuation*0.1 && fluctuationVariance < meanFluctuation*10 {
		score += 0.3
	}

	// 波动熵
	entropy := ebas.calculateJitterEntropy(secondDiff)
	if entropy > 0.5 {
		score += 0.2
	}

	return math.Max(0, math.Min(1, score))
}

// calculatePrecisionScore 计算精准度评分
func (ebas *EnhancedBehaviorAnalysisService) calculatePrecisionScore(points []BehaviorDataPoint) float64 {
	if len(points) < 3 {
		return 0.5
	}

	// 精准度：基于轨迹的直线度和方向变化
	score := 0.5

	// 1. 路径效率（人类不会走冤枉路）
	totalDistance := 0.0
	for i := 1; i < len(points); i++ {
		dx := float64(points[i].X - points[i-1].X)
		dy := float64(points[i].Y - points[i-1].Y)
		totalDistance += math.Sqrt(dx*dx + dy*dy)
	}

	startX, startY := float64(points[0].X), float64(points[0].Y)
	endX, endY := float64(points[len(points)-1].X), float64(points[len(points)-1].Y)
	netDistance := math.Sqrt((endX-startX)*(endX-startX) + (endY-startY)*(endY-startY))

	if totalDistance > 0 {
		efficiency := netDistance / totalDistance
		if efficiency > 0.3 && efficiency < 0.9 {
			score += 0.2
		}
	}

	// 2. 方向变化频率
	directionChanges := 0
	for i := 1; i < len(points)-1; i++ {
		dx1 := float64(points[i].X - points[i-1].X)
		dy1 := float64(points[i].Y - points[i-1].Y)
		dx2 := float64(points[i+1].X - points[i].X)
		dy2 := float64(points[i+1].Y - points[i].Y)

		mag1 := math.Sqrt(dx1*dx1 + dy1*dy1)
		mag2 := math.Sqrt(dx2*dx2 + dy2*dy2)

		if mag1 > 0 && mag2 > 0 {
			dot := dx1*dx2 + dy1*dy2
			cosAngle := dot / (mag1 * mag2)
			if cosAngle < 0.9 {
				directionChanges++
			}
		}
	}

	changeRatio := float64(directionChanges) / float64(len(points)-2)
	if changeRatio > 0.1 && changeRatio < 0.7 {
		score += 0.15
	}

	// 3. 曲线平滑度
	curvatures := ebas.calculateCurvatures(points)
	meanCurvature := ebas.meanFloat(curvatures)
	curvatureVariance := ebas.varianceFloat(curvatures)
	if curvatureVariance > 0 && meanCurvature < 1.0 {
		score += 0.15
	}

	return math.Max(0, math.Min(1, score))
}

// calculateConsistencyScore 计算一致性评分
func (ebas *EnhancedBehaviorAnalysisService) calculateConsistencyScore(speeds []float64) float64 {
	if len(speeds) < 2 {
		return 0.5
	}

	// 一致性：基于速度的统计特性
	score := 0.5

	// 1. 速度标准差（适度的一致性）
	speedVariance := ebas.varianceFloat(speeds)
	meanSpeed := ebas.meanFloat(speeds)
	if meanSpeed > 0 {
		cv := math.Sqrt(speedVariance) / meanSpeed
		if cv > 0.05 && cv < 0.5 {
			score += 0.25
		}
	}

	// 2. 速度自相关性（适度相关）
	autocorr := ebas.calculateAutocorrelation(speeds, 1)
	if autocorr > 0.3 && autocorr < 0.9 {
		score += 0.25
	}

	return math.Max(0, math.Min(1, score))
}

// computeCurvature 计算三个点的曲率
func (ebas *EnhancedBehaviorAnalysisService) computeCurvature(p1, p2, p3 BehaviorDataPoint) float64 {
	v1x := float64(p2.X - p1.X)
	v1y := float64(p2.Y - p1.Y)
	v2x := float64(p3.X - p2.X)
	v2y := float64(p3.Y - p2.Y)

	dot := v1x*v2x + v1y*v2y
	mag1 := math.Sqrt(v1x*v1x + v1y*v1y)
	mag2 := math.Sqrt(v2x*v2x + v2y*v2y)

	if mag1 == 0 || mag2 == 0 {
		return 0
	}

	cosAngle := dot / (mag1 * mag2)
	if cosAngle > 1 {
		cosAngle = 1
	}
	if cosAngle < -1 {
		cosAngle = -1
	}

	angle := math.Acos(cosAngle)

	cross := v1x*v2y - v1y*v2x
	if cross < 0 {
		angle = -angle
	}

	return angle
}

// ============================================
// 第二部分：集成学习分类器
// ============================================

// RandomForest 随机森林分类器
type RandomForest struct {
	trees           []*DecisionTree
	nTrees          int
	maxDepth        int
	minSamplesSplit int
}

// DecisionTree 决策树（简化实现）
type DecisionTree struct {
	root *TreeNode
}

// TreeNode 树节点
type TreeNode struct {
	isLeaf       bool
	prediction   float64
	featureIndex int
	threshold    float64
	left         *TreeNode
	right        *TreeNode
}

// NewRandomForest 创建新的随机森林
func NewRandomForest(nTrees, maxDepth, minSamplesSplit int) *RandomForest {
	return &RandomForest{
		trees:           make([]*DecisionTree, 0, nTrees),
		nTrees:          nTrees,
		maxDepth:        maxDepth,
		minSamplesSplit: minSamplesSplit,
	}
}

// Train 训练随机森林
func (rf *RandomForest) Train(X [][]float64, y []float64) {
	if len(X) == 0 || len(X) != len(y) {
		return
	}

	nSamples, nFeatures := len(X), len(X[0])

	for i := 0; i < rf.nTrees; i++ {
		// Bootstrap抽样
		sampleIndices := make([]int, nSamples)
		for j := 0; j < nSamples; j++ {
			sampleIndices[j] = rand.Intn(nSamples)
		}

		XBootstrap := make([][]float64, nSamples)
		yBootstrap := make([]float64, nSamples)
		for j := 0; j < nSamples; j++ {
			XBootstrap[j] = X[sampleIndices[j]]
			yBootstrap[j] = y[sampleIndices[j]]
		}

		// 训练决策树
		tree := rf.trainTree(XBootstrap, yBootstrap, 0, nFeatures)
		rf.trees = append(rf.trees, tree)
	}
}

// trainTree 训练单个决策树
func (rf *RandomForest) trainTree(X [][]float64, y []float64, depth, nFeatures int) *DecisionTree {
	tree := &DecisionTree{}
	tree.root = rf.buildTree(X, y, depth, nFeatures)
	return tree
}

// buildTree 构建树
func (rf *RandomForest) buildTree(X [][]float64, y []float64, depth, nFeatures int) *TreeNode {
	node := &TreeNode{}

	nSamples := len(X)
	if nSamples < rf.minSamplesSplit || depth >= rf.maxDepth {
		// 叶子节点
		node.isLeaf = true
		node.prediction = meanFloatSlice(y)
		return node
	}

	// 随机选择特征子集
	featureSubset := make([]int, 0)
	usedFeatures := make(map[int]bool)
	nSelect := int(math.Sqrt(float64(len(X[0])))) + 1
	if nSelect > len(X[0]) {
		nSelect = len(X[0])
	}

	for len(featureSubset) < nSelect {
		f := rand.Intn(len(X[0]))
		if !usedFeatures[f] {
			usedFeatures[f] = true
			featureSubset = append(featureSubset, f)
		}
	}

	// 寻找最佳分割
	bestGain := -1.0
	bestFeature := 0
	bestThreshold := 0.0
	var bestLeftX, bestRightX [][]float64
	var bestLeftY, bestRightY []float64

	for _, f := range featureSubset {
		// 获取该特征的所有值
		values := make([]float64, nSamples)
		for i := 0; i < nSamples; i++ {
			values[i] = X[i][f]
		}

		// 尝试不同阈值
		for i := 0; i < nSamples-1; i++ {
			threshold := (values[i] + values[i+1]) / 2

			// 分割
			leftX, rightX := make([][]float64, 0), make([][]float64, 0)
			leftY, rightY := make([]float64, 0), make([]float64, 0)

			for j := 0; j < nSamples; j++ {
				if X[j][f] <= threshold {
					leftX = append(leftX, X[j])
					leftY = append(leftY, y[j])
				} else {
					rightX = append(rightX, X[j])
					rightY = append(rightY, y[j])
				}
			}

			if len(leftY) == 0 || len(rightY) == 0 {
				continue
			}

			// 计算信息增益（方差减少）
			gain := rf.varianceReduction(y, leftY, rightY)
			if gain > bestGain {
				bestGain = gain
				bestFeature = f
				bestThreshold = threshold
				bestLeftX, bestRightX = leftX, rightX
				bestLeftY, bestRightY = leftY, rightY
			}
		}
	}

	if bestGain > 0 {
		node.featureIndex = bestFeature
		node.threshold = bestThreshold
		node.left = rf.buildTree(bestLeftX, bestLeftY, depth+1, nFeatures)
		node.right = rf.buildTree(bestRightX, bestRightY, depth+1, nFeatures)
	} else {
		// 无法分割，成为叶子节点
		node.isLeaf = true
		node.prediction = meanFloatSlice(y)
	}

	return node
}

// varianceReduction 计算方差减少
func (rf *RandomForest) varianceReduction(parent, left, right []float64) float64 {
	if len(parent) == 0 {
		return 0
	}

	varianceParent := varianceFloatSlice(parent)
	varianceLeft := varianceFloatSlice(left)
	varianceRight := varianceFloatSlice(right)

	weightedVariance := (float64(len(left))/float64(len(parent)))*varianceLeft +
		(float64(len(right))/float64(len(parent)))*varianceRight

	return varianceParent - weightedVariance
}

// Predict 预测
func (rf *RandomForest) Predict(x []float64) float64 {
	if len(rf.trees) == 0 {
		return 0.5
	}

	sum := 0.0
	for _, tree := range rf.trees {
		sum += tree.predict(x)
	}

	return sum / float64(len(rf.trees))
}

// predict 单棵树预测
func (dt *DecisionTree) predict(x []float64) float64 {
	return dt.root.predict(x)
}

func (tn *TreeNode) predict(x []float64) float64 {
	if tn.isLeaf {
		return tn.prediction
	}

	if x[tn.featureIndex] <= tn.threshold {
		return tn.left.predict(x)
	}
	return tn.right.predict(x)
}

// GradientBoostingTree 梯度提升树（简化实现，类似LightGBM风格）
type GradientBoostingTree struct {
	trees        []*DecisionTree
	learningRate float64
	nEstimators  int
	maxDepth     int
}

// NewGradientBoostingTree 创建新的梯度提升树
func NewGradientBoostingTree(learningRate float64, nEstimators, maxDepth int) *GradientBoostingTree {
	return &GradientBoostingTree{
		trees:        make([]*DecisionTree, 0, nEstimators),
		learningRate: learningRate,
		nEstimators:  nEstimators,
		maxDepth:     maxDepth,
	}
}

// Train 训练梯度提升树
func (gbt *GradientBoostingTree) Train(X [][]float64, y []float64) {
	if len(X) == 0 || len(X) != len(y) {
		return
	}

	// 初始化预测为均值
	predictions := make([]float64, len(y))
	initialPred := meanFloatSlice(y)
	for i := range predictions {
		predictions[i] = initialPred
	}

	for i := 0; i < gbt.nEstimators; i++ {
		// 计算梯度（残差）
		residuals := make([]float64, len(y))
		for j := range y {
			residuals[j] = y[j] - predictions[j]
		}

		// 训练弱学习器拟合残差
		tree := gbt.trainWeakLearner(X, residuals)
		gbt.trees = append(gbt.trees, tree)

		// 更新预测
		for j := range predictions {
			predictions[j] += gbt.learningRate * tree.predict(X[j])
		}
	}
}

// trainWeakLearner 训练弱学习器
func (gbt *GradientBoostingTree) trainWeakLearner(X [][]float64, y []float64) *DecisionTree {
	// 简单实现：使用决策树
	rf := NewRandomForest(1, gbt.maxDepth, 2)
	rf.Train(X, y)
	if len(rf.trees) > 0 {
		return rf.trees[0]
	}

	// 退化情况：返回均值
	tree := &DecisionTree{
		root: &TreeNode{
			isLeaf:     true,
			prediction: meanFloatSlice(y),
		},
	}
	return tree
}

// Predict 预测
func (gbt *GradientBoostingTree) Predict(x []float64) float64 {
	if len(gbt.trees) == 0 {
		return 0.5
	}

	pred := 0.0
	for _, tree := range gbt.trees {
		pred += gbt.learningRate * tree.predict(x)
	}

	// Sigmoid 函数处理为概率
	return 1.0 / (1.0 + math.Exp(-pred))
}

// AdvancedEnsembleClassifier 高级集成分类器
type AdvancedEnsembleClassifier struct {
	randomForest     *RandomForest
	gradientBoosting *GradientBoostingTree
	rfWeight         float64
	gbtWeight        float64
}

// NewAdvancedEnsembleClassifier 创建新的高级集成分类器
func NewAdvancedEnsembleClassifier() *AdvancedEnsembleClassifier {
	return &AdvancedEnsembleClassifier{
		randomForest:     NewRandomForest(50, 10, 2),
		gradientBoosting: NewGradientBoostingTree(0.1, 100, 6),
		rfWeight:         0.4,
		gbtWeight:        0.6,
	}
}

// Train 训练集成分类器
func (ec *AdvancedEnsembleClassifier) Train(X [][]float64, y []float64) {
	// 数据预处理：标准化
	XNormalized := ec.normalizeFeatures(X)

	// 训练两个分类器
	ec.randomForest.Train(XNormalized, y)
	ec.gradientBoosting.Train(XNormalized, y)
}

// normalizeFeatures 特征标准化
func (ec *AdvancedEnsembleClassifier) normalizeFeatures(X [][]float64) [][]float64 {
	if len(X) == 0 {
		return X
	}

	nFeatures := len(X[0])
	means := make([]float64, nFeatures)
	stds := make([]float64, nFeatures)

	// 计算均值和标准差
	for f := 0; f < nFeatures; f++ {
		values := make([]float64, len(X))
		for i := 0; i < len(X); i++ {
			values[i] = X[i][f]
		}
		means[f] = meanFloatSlice(values)
		stds[f] = math.Sqrt(varianceFloatSlice(values))
		if stds[f] == 0 {
			stds[f] = 1
		}
	}

	// 标准化
	normalized := make([][]float64, len(X))
	for i := 0; i < len(X); i++ {
		normalized[i] = make([]float64, nFeatures)
		for f := 0; f < nFeatures; f++ {
			normalized[i][f] = (X[i][f] - means[f]) / stds[f]
		}
	}

	return normalized
}

// Predict 预测
func (ec *AdvancedEnsembleClassifier) Predict(x []float64) (float64, bool) {
	// 标准化单个样本（简化版）
	// 实际应用中应该保存训练时的均值和标准差

	rfPred := ec.randomForest.Predict(x)
	gbtPred := ec.gradientBoosting.Predict(x)

	// 加权平均
	ensemblePred := ec.rfWeight*rfPred + ec.gbtWeight*gbtPred

	// 确保在0-1范围内
	ensemblePred = math.Max(0, math.Min(1, ensemblePred))

	// 阈值判断
	isBot := ensemblePred >= 0.5

	return ensemblePred, isBot
}

// ============================================
// 第三部分：异常检测（Isolation Forest）
// ============================================

// IsolationForest 孤立森林异常检测
type IsolationForest struct {
	trees      []*IsolationTree
	nTrees     int
	sampleSize int
}

// IsolationTree 孤立树
type IsolationTree struct {
	root        *IsolationNode
	heightLimit int
}

// IsolationNode 孤立树节点
type IsolationNode struct {
	isLeaf       bool
	size         int
	splitFeature int
	splitValue   float64
	left         *IsolationNode
	right        *IsolationNode
}

// NewIsolationForest 创建新的孤立森林
func NewIsolationForest(nTrees, sampleSize int) *IsolationForest {
	return &IsolationForest{
		trees:      make([]*IsolationTree, 0, nTrees),
		nTrees:     nTrees,
		sampleSize: sampleSize,
	}
}

// Train 训练孤立森林
func (iforest *IsolationForest) Train(X [][]float64) {
	if len(X) == 0 {
		return
	}

	nSamples := len(X)
	heightLimit := int(math.Ceil(math.Log2(float64(iforest.sampleSize))))

	for i := 0; i < iforest.nTrees; i++ {
		// 抽样
		sampleIndices := make([]int, 0, iforest.sampleSize)
		used := make(map[int]bool)
		for len(sampleIndices) < iforest.sampleSize && len(sampleIndices) < nSamples {
			idx := rand.Intn(nSamples)
			if !used[idx] {
				used[idx] = true
				sampleIndices = append(sampleIndices, idx)
			}
		}

		XSample := make([][]float64, len(sampleIndices))
		for j := 0; j < len(sampleIndices); j++ {
			XSample[j] = X[sampleIndices[j]]
		}

		// 构建孤立树
		tree := iforest.buildTree(XSample, 0, heightLimit)
		iforest.trees = append(iforest.trees, tree)
	}
}

// buildTree 构建孤立树
func (iforest *IsolationForest) buildTree(X [][]float64, currentHeight, heightLimit int) *IsolationTree {
	tree := &IsolationTree{
		heightLimit: heightLimit,
	}
	tree.root = iforest.buildNode(X, currentHeight, heightLimit)
	return tree
}

// buildNode 构建节点
func (iforest *IsolationForest) buildNode(X [][]float64, currentHeight, heightLimit int) *IsolationNode {
	node := &IsolationNode{}
	node.size = len(X)

	if len(X) <= 1 || currentHeight >= heightLimit {
		node.isLeaf = true
		return node
	}

	// 随机选择特征和分割值
	nFeatures := len(X[0])
	feature := rand.Intn(nFeatures)

	// 找该特征的最小值和最大值
	minVal := X[0][feature]
	maxVal := X[0][feature]
	for i := 1; i < len(X); i++ {
		if X[i][feature] < minVal {
			minVal = X[i][feature]
		}
		if X[i][feature] > maxVal {
			maxVal = X[i][feature]
		}
	}

	if minVal == maxVal {
		node.isLeaf = true
		return node
	}

	// 随机分割值
	splitValue := minVal + rand.Float64()*(maxVal-minVal)

	// 分割数据
	leftX := make([][]float64, 0)
	rightX := make([][]float64, 0)
	for i := 0; i < len(X); i++ {
		if X[i][feature] < splitValue {
			leftX = append(leftX, X[i])
		} else {
			rightX = append(rightX, X[i])
		}
	}

	node.splitFeature = feature
	node.splitValue = splitValue
	node.left = iforest.buildNode(leftX, currentHeight+1, heightLimit)
	node.right = iforest.buildNode(rightX, currentHeight+1, heightLimit)

	return node
}

// AnomalyScore 计算异常分数
func (iforest *IsolationForest) AnomalyScore(x []float64) float64 {
	if len(iforest.trees) == 0 {
		return 0.5
	}

	totalPathLength := 0.0
	for _, tree := range iforest.trees {
		totalPathLength += tree.pathLength(x, 0)
	}

	avgPathLength := totalPathLength / float64(len(iforest.trees))

	// 计算期望路径长度（用于标准化）
	c := func(n int) float64 {
		if n <= 1 {
			return 0
		}
		return 2*harmonicNumber(n-1) - 2*float64(n-1)/float64(n)
	}(iforest.sampleSize)

	if c == 0 {
		return 0.5
	}

	// 计算异常分数
	score := math.Pow(2, -avgPathLength/c)
	return score
}

// pathLength 计算路径长度
func (it *IsolationTree) pathLength(x []float64, currentHeight int) float64 {
	return it.root.pathLength(x, currentHeight)
}

func (in *IsolationNode) pathLength(x []float64, currentHeight int) float64 {
	if in.isLeaf {
		// 叶子节点，返回调整后的路径长度
		if in.size > 1 {
			return float64(currentHeight) + c(in.size)
		}
		return float64(currentHeight)
	}

	if x[in.splitFeature] < in.splitValue {
		return in.left.pathLength(x, currentHeight+1)
	}
	return in.right.pathLength(x, currentHeight+1)
}

// harmonicNumber 调和数
func harmonicNumber(n int) float64 {
	if n <= 0 {
		return 0
	}
	h := 0.0
	for i := 1; i <= n; i++ {
		h += 1.0 / float64(i)
	}
	return h
}

// c 平均路径长度
func c(n int) float64 {
	if n <= 1 {
		return 0
	}
	return 2*harmonicNumber(n-1) - 2*float64(n-1)/float64(n)
}

// ============================================
// 第四部分：多维度风险评分系统
// ============================================

// MultiDimensionalRiskScore 多维度风险评分
type MultiDimensionalRiskScore struct {
	BehaviorRisk      float64            // 行为风险（0-100）
	EnvironmentalRisk float64            // 环境风险（0-100）
	HistoricalRisk    float64            // 历史风险（0-100）
	AnomalyScore      float64            // 异常分数（0-1）
	OverallRisk       float64            // 综合风险（0-100）
	IsBot             bool               // 是否是机器人
	Confidence        float64            // 置信度（0-1）
	RiskContributors  map[string]float64 // 风险贡献因子
}

// EnhancedAnalysisResult 增强版分析结果
type EnhancedAnalysisResult struct {
	*AnalysisResult                              // 原有结果
	EnhancedFeatures  *EnhancedFeatures          // 新增特征
	EnsembleScore     float64                    // 集成学习评分
	AnomalyScore      float64                    // 异常检测评分
	MultiDimRisk      *MultiDimensionalRiskScore // 多维度风险评分
	FeatureImportance map[string]float64         // 特征重要性
}

// AdaptiveThreshold 自适应阈值
type AdaptiveThreshold struct {
	threshold    float64
	learningRate float64
	minThreshold float64
	maxThreshold float64
	history      []float64
}

// NewAdaptiveThreshold 创建自适应阈值
func NewAdaptiveThreshold(initial, min, max, lr float64) *AdaptiveThreshold {
	return &AdaptiveThreshold{
		threshold:    initial,
		learningRate: lr,
		minThreshold: min,
		maxThreshold: max,
		history:      make([]float64, 0, 1000),
	}
}

// Update 更新阈值
func (at *AdaptiveThreshold) Update(isFalsePositive bool) {
	at.history = append(at.history, at.threshold)
	if len(at.history) > 1000 {
		at.history = at.history[1:]
	}

	if isFalsePositive {
		// 误报：提高阈值
		at.threshold *= (1 + at.learningRate)
	} else {
		// 正常：稍微降低阈值
		at.threshold *= (1 - at.learningRate*0.1)
	}

	// 裁剪
	if at.threshold < at.minThreshold {
		at.threshold = at.minThreshold
	}
	if at.threshold > at.maxThreshold {
		at.threshold = at.maxThreshold
	}
}

// GetThreshold 获取当前阈值
func (at *AdaptiveThreshold) GetThreshold() float64 {
	return at.threshold
}

// ============================================
// 第五部分：增强版行为分析服务
// ============================================

// AnalyzeBehaviorEnhanced 增强版行为分析
func (ebas *EnhancedBehaviorAnalysisService) AnalyzeBehaviorEnhanced(behaviorData []models.BehaviorData) (*EnhancedAnalysisResult, error) {
	// 1. 先进行基础分析
	basicResult, err := ebas.AnalyzeBehavior(behaviorData)
	if err != nil {
		return nil, err
	}

	result := &EnhancedAnalysisResult{
		AnalysisResult:    basicResult,
		FeatureImportance: make(map[string]float64),
	}

	// 2. 解析数据
	var points []BehaviorDataPoint
	var clicks []BehaviorDataPoint
	var keyStrokes []KeyboardDataPoint

	for _, bd := range behaviorData {
		switch bd.DataType {
		case "keyboard":
			var kp KeyboardDataPoint
			if json.Unmarshal([]byte(bd.Data), &kp) == nil {
				keyStrokes = append(keyStrokes, kp)
			}
		default:
			var dp BehaviorDataPoint
			if json.Unmarshal([]byte(bd.Data), &dp) == nil {
				points = append(points, dp)
				if dp.Event == "click" {
					clicks = append(clicks, dp)
				}
			}
		}
	}

	// 3. 提取高级特征
	result.EnhancedFeatures = ebas.ExtractEnhancedFeatures(points, clicks, keyStrokes)

	// 4. 准备特征向量
	featureVector := ebas.createFeatureVector(basicResult, result.EnhancedFeatures)

	// 5. 集成学习预测
	if len(ebas.trainingData) > 10 { // 确保有足够的训练数据
		ensembleScore, isBotEnsemble := ebas.ensembleClassifier.Predict(featureVector)
		result.EnsembleScore = ensembleScore

		// 记录特征重要性
		result.FeatureImportance = ebas.calculateFeatureImportance(featureVector)
		_ = isBotEnsemble
	} else {
		result.EnsembleScore = basicResult.RiskScore / 100.0
	}

	// 6. 异常检测
	if len(featureVector) > 0 {
		result.AnomalyScore = ebas.isolationForest.AnomalyScore(featureVector)
	}

	// 7. 多维度风险评分
	result.MultiDimRisk = ebas.calculateMultiDimensionalRisk(
		basicResult,
		result.EnsembleScore,
		result.AnomalyScore,
	)

	// 8. 更新最终结果
	threshold := ebas.adaptiveThreshold.GetThreshold()
	result.MultiDimRisk.IsBot = result.MultiDimRisk.OverallRisk >= threshold
	result.IsBotLikely = result.MultiDimRisk.IsBot
	result.RiskScore = result.MultiDimRisk.OverallRisk
	result.Confidence = result.MultiDimRisk.Confidence

	// 9. 在线学习（保存样本用于后续训练）
	ebas.addTrainingSample(featureVector, result.IsBotLikely)

	return result, nil
}

// createFeatureVector 创建特征向量（扩展到768维）
func (ebas *EnhancedBehaviorAnalysisService) createFeatureVector(basic *AnalysisResult, enhanced *EnhancedFeatures) []float64 {
	features := make([]float64, 0, 200)

	// ============ 基础特征（15个）============
	features = append(features, basic.RiskScore/100.0)
	features = append(features, basic.Trajectory.PathEfficiency)
	features = append(features, basic.Trajectory.AverageSpeed)
	features = append(features, basic.Trajectory.CurvatureAvg)
	features = append(features, basic.Trajectory.JitterScore)
	features = append(features, float64(basic.Trajectory.PauseCount))
	features = append(features, float64(basic.Trajectory.MicroCorrections))
	features = append(features, basic.ClickPattern.Regularity)
	features = append(features, basic.ClickPattern.PositionEntropy)
	features = append(features, basic.SpeedAnalysis.SpeedVariance)
	features = append(features, basic.SpeedAnalysis.SpeedStdDev)
	features = append(features, basic.SpeedAnalysis.SpeedSkewness)
	features = append(features, basic.SpeedAnalysis.SpeedEntropy)
	features = append(features, basic.SpeedAnalysis.SpeedBurstiness)
	features = append(features, basic.SpeedAnalysis.NormalizedSpeedVariance)

	// ============ 高级曲率特征（3个）============
	if enhanced != nil {
		features = append(features, enhanced.CurvatureVariance)
		features = append(features, float64(enhanced.CurvaturePeaks))
		features = append(features, enhanced.CurvatureEntropy)

		// ============ 点击间隔特征（5个）============
		features = append(features, enhanced.ClickIntervalSkewness)
		features = append(features, enhanced.ClickIntervalKurtosis)
		if len(enhanced.ClickIntervalQuantiles) >= 5 {
			features = append(features, enhanced.ClickIntervalQuantiles[0])
			features = append(features, enhanced.ClickIntervalQuantiles[1])
			features = append(features, enhanced.ClickIntervalQuantiles[2])
			features = append(features, enhanced.ClickIntervalQuantiles[3])
			features = append(features, enhanced.ClickIntervalQuantiles[4])
		} else {
			for i := 0; i < 5; i++ {
				features = append(features, 0.0)
			}
		}

		// ============ 方向变化特征（2个）============
		features = append(features, enhanced.DirectionChangeFrequency)
		features = append(features, enhanced.DirectionChangeRegularity)

		// ============ 频域分析特征（2个）============
		features = append(features, enhanced.FourierDominantFrequency)
		features = append(features, enhanced.FourierEnergyRatio)

		// ============ 分形特征（1个）============
		features = append(features, enhanced.FractalDimension)

		// ============ 悬停特征（2个）============
		features = append(features, float64(enhanced.HoverCount))
		features = append(features, enhanced.HoverDurationVariance)

		// ============ 键盘节奏特征（2个）============
		features = append(features, enhanced.KeyPressIntervalRegularity)
		features = append(features, enhanced.KeyHoldDurationVariance)

		// ============ 停顿特征（2个）============
		features = append(features, enhanced.PauseFrequency)
		features = append(features, enhanced.JerkAverage)

		// ============ 加加速度特征（2个）============
		features = append(features, enhanced.JerkAverage)
		features = append(features, enhanced.JerkVariance)

		// ============ 速度特征扩展（15个）- 任务3.1&3.2 ============
		features = append(features, enhanced.SpeedMedian)
		features = append(features, enhanced.SpeedSkewness)
		features = append(features, enhanced.SpeedKurtosis)
		features = append(features, enhanced.SpeedRange)
		features = append(features, enhanced.SpeedIQR)
		features = append(features, enhanced.SpeedCoefficientVariation)
		features = append(features, enhanced.SpeedPercentile25)
		features = append(features, enhanced.SpeedPercentile75)
		features = append(features, enhanced.SpeedPercentile90)
		features = append(features, enhanced.UniformMotionRatio)
		features = append(features, enhanced.AccelerationVariance)
		features = append(features, enhanced.AccelerationSkewness)
		features = append(features, enhanced.AccelerationKurtosis)
		features = append(features, enhanced.HumanMachineSpeedDiff)
		features = append(features, enhanced.SpeedCurveComplexity)

		// ============ 抖动特征扩展（10个）- 任务3.3 ============
		features = append(features, enhanced.JitterFrequency)
		features = append(features, enhanced.JitterAmplitudeMean)
		features = append(features, enhanced.JitterAmplitudeMax)
		features = append(features, enhanced.JitterAmplitudeVariance)
		features = append(features, float64(enhanced.MicroJitterCount))
		features = append(features, enhanced.MicroJitterRatio)
		features = append(features, enhanced.JitterRegularity)
		features = append(features, enhanced.JitterEntropy)
		features = append(features, float64(enhanced.JitterClusterCount))
		features = append(features, 0.0) // JitterWaveformType是字符串，转换为数值

		// ============ 加速度特征扩展（8个）- 任务3.1 ============
		features = append(features, enhanced.AccelerationMean)
		features = append(features, enhanced.AccelerationRange)
		features = append(features, enhanced.DecelerationRatio)
		features = append(features, float64(enhanced.AccelerationPeakCount))
		features = append(features, float64(enhanced.AccelerationZeroCrossing))
		features = append(features, enhanced.AccelerationEnergy)
		features = append(features, enhanced.TangentialAcceleration)
		features = append(features, enhanced.NormalAcceleration)

		// ============ 时序特征扩展（12个）============
		features = append(features, enhanced.InterPointTimeMean)
		features = append(features, enhanced.InterPointTimeVariance)
		features = append(features, enhanced.InterPointTimeSkewness)
		features = append(features, enhanced.InterPointTimeCV)
		features = append(features, enhanced.TimeRegularityIndex)
		features = append(features, enhanced.TimeBurstinessIndex)
		features = append(features, float64(enhanced.LongPauseCount))
		features = append(features, enhanced.LongPauseRatio)
		features = append(features, float64(enhanced.ShortPauseCount))
		features = append(features, enhanced.ShortPauseRatio)
		features = append(features, enhanced.TimeSequenceEntropy)
		features = append(features, 0.0) // TimeSequencePattern是字符串

		// ============ 空间特征扩展（10个）============
		features = append(features, enhanced.TotalDisplacement)
		features = append(features, enhanced.NetDisplacement)
		features = append(features, enhanced.DisplacementRatio)
		features = append(features, enhanced.AreaCovered)
		features = append(features, enhanced.AreaPerDistance)
		features = append(features, enhanced.SpatialConcentration)
		features = append(features, enhanced.SpatialSpread)
		features = append(features, enhanced.TrajectoryRoughness)
		features = append(features, enhanced.TrajectorySelfSimilarity)
		features = append(features, enhanced.PathSinuosity)

		// ============ 统计特征扩展（8个）============
		features = append(features, float64(enhanced.PointCount))
		features = append(features, enhanced.MeanPointInterval)
		features = append(features, enhanced.PointIntervalConsistency)
		features = append(features, enhanced.TrajectoryDuration)
		features = append(features, enhanced.VelocityAutoCorrelation)
		features = append(features, enhanced.VelocityLaggedCorrelation)
		features = append(features, enhanced.AccelerationAutoCorr)
		features = append(features, enhanced.MovementRhythmScore)

		// ============ 人机差异特征（5个）- 任务3.2 ============
		features = append(features, enhanced.HumanLikenessScore)
		features = append(features, enhanced.MechanicalPatternScore)
		features = append(features, enhanced.NaturalFluctuationScore)
		features = append(features, enhanced.PrecisionScore)
		features = append(features, enhanced.ConsistencyScore)
	}

	// 如果特征向量不够768维，用0填充（预留扩展空间）
	for len(features) < 768 {
		features = append(features, 0.0)
	}

	// 如果超过768维，截断到768维
	if len(features) > 768 {
		features = features[:768]
	}

	return features
}

// calculateFeatureImportance 计算特征重要性
func (ebas *EnhancedBehaviorAnalysisService) calculateFeatureImportance(featureVector []float64) map[string]float64 {
	importance := make(map[string]float64)

	// 简化实现：基于特征值的绝对值
	featureNames := []string{
		"base_risk_score", "path_efficiency", "avg_speed", "curvature_avg", "jitter_score",
		"pause_count", "micro_corrections", "click_regularity", "position_entropy",
		"curvature_variance", "curvature_peaks", "curvature_entropy", "click_skewness",
		"click_kurtosis", "direction_freq", "direction_regularity", "fourier_freq",
		"fourier_energy", "fractal_dim", "hover_count", "hover_var", "key_regularity",
		"pause_freq", "jerk_avg", "jerk_var",
	}

	for i := 0; i < len(featureNames) && i < len(featureVector); i++ {
		importance[featureNames[i]] = math.Abs(featureVector[i])
	}

	return importance
}

// calculateMultiDimensionalRisk 计算多维度风险
func (ebas *EnhancedBehaviorAnalysisService) calculateMultiDimensionalRisk(
	basic *AnalysisResult,
	ensembleScore float64,
	anomalyScore float64,
) *MultiDimensionalRiskScore {

	mdRisk := &MultiDimensionalRiskScore{
		RiskContributors: make(map[string]float64),
	}

	// 1. 行为风险（基于原有分析）
	mdRisk.BehaviorRisk = basic.RiskScore
	mdRisk.RiskContributors["behavior"] = 0.8

	// 2. 环境风险（简化实现，实际应该有IP信誉、设备指纹等）
	mdRisk.EnvironmentalRisk = 10.0 // 默认低风险
	mdRisk.RiskContributors["environmental"] = 0.1

	// 3. 历史风险（简化实现）
	mdRisk.HistoricalRisk = 10.0 // 默认低风险
	mdRisk.RiskContributors["historical"] = 0.05

	// 4. 异常分数
	mdRisk.AnomalyScore = anomalyScore
	mdRisk.RiskContributors["anomaly"] = 0.05

	// 5. 综合风险评分
	mdRisk.OverallRisk =
		mdRisk.BehaviorRisk*mdRisk.RiskContributors["behavior"] +
			mdRisk.EnvironmentalRisk*mdRisk.RiskContributors["environmental"] +
			mdRisk.HistoricalRisk*mdRisk.RiskContributors["historical"] +
			anomalyScore*100*mdRisk.RiskContributors["anomaly"]

	// 结合集成学习分数（如果可用）
	if ensembleScore > 0 {
		mdRisk.OverallRisk = mdRisk.OverallRisk*0.7 + ensembleScore*100*0.3
	}

	// 确保在0-100范围内
	mdRisk.OverallRisk = math.Max(0, math.Min(100, mdRisk.OverallRisk))

	// 6. 计算置信度
	confidence := 0.0
	if basic.RiskScore > 70 || basic.RiskScore < 30 {
		confidence += 0.3
	}
	if anomalyScore > 0.7 || anomalyScore < 0.3 {
		confidence += 0.2
	}
	if ensembleScore > 0.7 || ensembleScore < 0.3 {
		confidence += 0.3
	}
	confidence += 0.2 // 基础置信度
	mdRisk.Confidence = math.Max(0, math.Min(1, confidence))

	return mdRisk
}

// addTrainingSample 添加训练样本（在线学习）
func (ebas *EnhancedBehaviorAnalysisService) addTrainingSample(features []float64, isBot bool) {
	label := 0.0
	if isBot {
		label = 1.0
	}

	ebas.trainingData = append(ebas.trainingData, features)
	ebas.trainingLabels = append(ebas.trainingLabels, label)

	// 限制训练数据量
	if len(ebas.trainingData) > 10000 {
		ebas.trainingData = ebas.trainingData[len(ebas.trainingData)-10000:]
		ebas.trainingLabels = ebas.trainingLabels[len(ebas.trainingLabels)-10000:]
	}

	// 定期重新训练
	if len(ebas.trainingData)%100 == 0 && len(ebas.trainingData) > 50 {
		go func() {
			ebas.ensembleClassifier.Train(ebas.trainingData, ebas.trainingLabels)
			ebas.isolationForest.Train(ebas.trainingData)
		}()
	}
}

// InitializeWithSampleData 使用示例数据初始化训练
func (ebas *EnhancedBehaviorAnalysisService) InitializeWithSampleData() {
	// 生成一些模拟的训练数据
	rand.Seed(time.Now().UnixNano())

	nSamples := 100
	for i := 0; i < nSamples; i++ {
		isBot := i%2 == 0

		features := make([]float64, 25)
		for f := 0; f < 25; f++ {
			if isBot {
				// 机器人特征：更规律
				features[f] = rand.NormFloat64()*0.1 + 0.8
			} else {
				// 人类特征：更多变化
				features[f] = rand.NormFloat64()*0.5 + 0.5
			}
		}

		// 裁剪到0-1
		for f := range features {
			features[f] = math.Max(0, math.Min(1, features[f]))
		}

		label := 0.0
		if isBot {
			label = 1.0
		}

		ebas.trainingData = append(ebas.trainingData, features)
		ebas.trainingLabels = append(ebas.trainingLabels, label)
	}

	// 训练模型
	ebas.ensembleClassifier.Train(ebas.trainingData, ebas.trainingLabels)
	ebas.isolationForest.Train(ebas.trainingData)
}

// VerifyWithEnhancedBehavior 使用增强版分析进行验证
func (ebas *EnhancedBehaviorAnalysisService) VerifyWithEnhancedBehavior(
	captchaSuccess bool,
	behaviorData []models.BehaviorData,
) (bool, float64, string, *EnhancedAnalysisResult) {

	result, err := ebas.AnalyzeBehaviorEnhanced(behaviorData)
	if err != nil {
		// 降级到基础分析
		passed, score, report := ebas.VerifyWithBehaviorAnalysis(captchaSuccess, behaviorData)
		return passed, score, report, nil
	}

	analysisReport := ebas.GenerateEnhancedAnalysisReport(result)

	var finalResult bool
	if result.MultiDimRisk.OverallRisk < 30 {
		finalResult = captchaSuccess
	} else if result.MultiDimRisk.OverallRisk < 70 {
		finalResult = captchaSuccess && result.MultiDimRisk.OverallRisk < ebas.adaptiveThreshold.GetThreshold()
	} else {
		finalResult = false
	}

	return finalResult, result.MultiDimRisk.OverallRisk, analysisReport, result
}

// GenerateEnhancedAnalysisReport 生成增强版分析报告
func (ebas *EnhancedBehaviorAnalysisService) GenerateEnhancedAnalysisReport(result *EnhancedAnalysisResult) string {
	report := "=== 增强版行为分析报告 ===\n"
	report += ebas.GenerateAnalysisReport(result.AnalysisResult)

	report += "\n--- 高级特征分析 ---\n"
	if result.EnhancedFeatures != nil {
		report += fmt.Sprintf("曲率变化方差: %.6f\n", result.EnhancedFeatures.CurvatureVariance)
		report += fmt.Sprintf("曲率峰值数量: %d\n", result.EnhancedFeatures.CurvaturePeaks)
		report += fmt.Sprintf("曲率熵: %.4f\n", result.EnhancedFeatures.CurvatureEntropy)
		report += fmt.Sprintf("分形维数: %.4f\n", result.EnhancedFeatures.FractalDimension)
		report += fmt.Sprintf("傅里叶主频率: %.4f\n", result.EnhancedFeatures.FourierDominantFrequency)
		report += fmt.Sprintf("方向变化频率: %.4f\n", result.EnhancedFeatures.DirectionChangeFrequency)
	}

	report += "\n--- 集成学习 ---\n"
	report += fmt.Sprintf("集成学习评分: %.4f\n", result.EnsembleScore)
	report += fmt.Sprintf("异常检测评分: %.4f\n", result.AnomalyScore)

	if result.MultiDimRisk != nil {
		report += "\n--- 多维度风险评分 ---\n"
		report += fmt.Sprintf("行为风险: %.2f\n", result.MultiDimRisk.BehaviorRisk)
		report += fmt.Sprintf("环境风险: %.2f\n", result.MultiDimRisk.EnvironmentalRisk)
		report += fmt.Sprintf("历史风险: %.2f\n", result.MultiDimRisk.HistoricalRisk)
		report += fmt.Sprintf("综合风险: %.2f\n", result.MultiDimRisk.OverallRisk)
		report += fmt.Sprintf("置信度: %.4f\n", result.MultiDimRisk.Confidence)
	}

	report += "\n--- 特征重要性 ---\n"
	for feature, importance := range result.FeatureImportance {
		report += fmt.Sprintf("  %s: %.4f\n", feature, importance)
	}

	return report
}

// ============================================
// 辅助函数
// ============================================

func (ebas *EnhancedBehaviorAnalysisService) meanFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (ebas *EnhancedBehaviorAnalysisService) varianceFloat(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	mean := ebas.meanFloat(values)
	sum := 0.0
	for _, v := range values {
		sum += math.Pow(v-mean, 2)
	}
	return sum / float64(len(values))
}

func (ebas *EnhancedBehaviorAnalysisService) minFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

func (ebas *EnhancedBehaviorAnalysisService) maxFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

func meanFloatSlice(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func varianceFloatSlice(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	mean := meanFloatSlice(values)
	sum := 0.0
	for _, v := range values {
		sum += math.Pow(v-mean, 2)
	}
	return sum / float64(len(values))
}
