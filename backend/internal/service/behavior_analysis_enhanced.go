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

// EnhancedFeatures 包含所有新增的高级特征
type EnhancedFeatures struct {
	CurvatureVariance        float64   // 曲率变化方差
	CurvaturePeaks         int       // 曲率峰值数量
	CurvatureEntropy       float64   // 曲率熵
	ClickIntervalSkewness   float64   // 点击间隔偏度
	ClickIntervalKurtosis   float64   // 点击间隔峰度
	ClickIntervalQuantiles []float64 // 点击间隔分位数
	DirectionChangeFrequency   float64   // 方向变化频率
	DirectionChangeRegularity float64 // 方向变化规律性
	FourierDominantFrequency float64 // 傅里叶主频率
	FourierEnergyRatio    float64   // 傅里叶能量比
	FractalDimension      float64   // 分形维数
	HoverCount            int       // 悬停次数
	HoverDurationVariance float64  // 悬停时长方差
	ScrollSpeedVariance   float64  // 滚动速度方差
	ScrollDirectionChanges int      // 滚动方向变化次数
	KeyPressIntervalRegularity float64 // 按键间隔规律性
	KeyHoldDurationVariance float64  // 按键保持时长方差
	PauseIntervalDistribution []float64 // 停顿间隔分布
	PauseFrequency        float64   // 停顿频率
	JerkAverage         float64   // 平均加加速度（加速度的变化率）
	JerkVariance     float64   // 加加速度方差
}

// EnhancedBehaviorAnalysisService 增强版行为分析服务
type EnhancedBehaviorAnalysisService struct {
	*BehaviorAnalysisService
	ensembleClassifier  *AdvancedEnsembleClassifier
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
		isolationForest:      NewIsolationForest(100, 256),
		adaptiveThreshold:    NewAdaptiveThreshold(65.0, 40.0, 85.0, 0.01),
		trainingData:       make([][]float64, 0),
		trainingLabels:     make([]float64, 0),
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
		intervals = append(intervals, directionChanges[i] - directionChanges[i-1])
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
		boxSize := int(math.Pow(2, float64(maxScale - scale)))
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
		intervals = append(intervals, float64(keyStrokes[i].Timestamp - keyStrokes[i-1].Timestamp))
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
	trees            []*DecisionTree
	nTrees           int
	maxDepth         int
	minSamplesSplit int
}

// DecisionTree 决策树（简化实现）
type DecisionTree struct {
	root *TreeNode
}

// TreeNode 树节点
type TreeNode struct {
	isLeaf        bool
	prediction   float64
	featureIndex  int
	threshold    float64
	left         *TreeNode
	right         *TreeNode
}

// NewRandomForest 创建新的随机森林
func NewRandomForest(nTrees, maxDepth, minSamplesSplit int) *RandomForest {
	return &RandomForest{
		trees:            make([]*DecisionTree, 0, nTrees),
		nTrees:           nTrees,
		maxDepth:         maxDepth,
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
	trees            []*DecisionTree
	learningRate    float64
	nEstimators      int
	maxDepth         int
}

// NewGradientBoostingTree 创建新的梯度提升树
func NewGradientBoostingTree(learningRate float64, nEstimators, maxDepth int) *GradientBoostingTree {
	return &GradientBoostingTree{
		trees:            make([]*DecisionTree, 0, nEstimators),
		learningRate:    learningRate,
		nEstimators:      nEstimators,
		maxDepth:         maxDepth,
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
			isLeaf: true,
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
	randomForest      *RandomForest
	gradientBoosting  *GradientBoostingTree
	rfWeight         float64
	gbtWeight        float64
}

// NewAdvancedEnsembleClassifier 创建新的高级集成分类器
func NewAdvancedEnsembleClassifier() *AdvancedEnsembleClassifier {
	return &AdvancedEnsembleClassifier{
		randomForest:      NewRandomForest(50, 10, 2),
		gradientBoosting:  NewGradientBoostingTree(0.1, 100, 6),
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
	trees        []*IsolationTree
	nTrees      int
	sampleSize  int
}

// IsolationTree 孤立树
type IsolationTree struct {
	root        *IsolationNode
	heightLimit int
}

// IsolationNode 孤立树节点
type IsolationNode struct {
	isLeaf        bool
	size        int
	splitFeature int
	splitValue float64
	left        *IsolationNode
	right        *IsolationNode
}

// NewIsolationForest 创建新的孤立森林
func NewIsolationForest(nTrees, sampleSize int) *IsolationForest {
	return &IsolationForest{
		trees:        make([]*IsolationTree, 0, nTrees),
		nTrees:      nTrees,
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
	BehaviorRisk        float64            // 行为风险（0-100）
	EnvironmentalRisk float64            // 环境风险（0-100）
	HistoricalRisk    float64            // 历史风险（0-100）
	AnomalyScore     float64            // 异常分数（0-1）
	OverallRisk       float64            // 综合风险（0-100）
	IsBot            bool               // 是否是机器人
	Confidence       float64            // 置信度（0-1）
	RiskContributors map[string]float64 // 风险贡献因子
}

// EnhancedAnalysisResult 增强版分析结果
type EnhancedAnalysisResult struct {
	*AnalysisResult            // 原有结果
	EnhancedFeatures   *EnhancedFeatures      // 新增特征
	EnsembleScore        float64            // 集成学习评分
	AnomalyScore     float64            // 异常检测评分
	MultiDimRisk       *MultiDimensionalRiskScore // 多维度风险评分
	FeatureImportance map[string]float64  // 特征重要性
}

// AdaptiveThreshold 自适应阈值
type AdaptiveThreshold struct {
	threshold        float64
	learningRate    float64
	minThreshold    float64
	maxThreshold    float64
	history        []float64
}

// NewAdaptiveThreshold 创建自适应阈值
func NewAdaptiveThreshold(initial, min, max, lr float64) *AdaptiveThreshold {
	return &AdaptiveThreshold{
		threshold:        initial,
		learningRate:    lr,
		minThreshold:    min,
		maxThreshold:    max,
		history:        make([]float64, 0, 1000),
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

// createFeatureVector 创建特征向量
func (ebas *EnhancedBehaviorAnalysisService) createFeatureVector(basic *AnalysisResult, enhanced *EnhancedFeatures) []float64 {
	features := make([]float64, 0)
	
	// 基础特征
	features = append(features, basic.RiskScore/100.0)
	features = append(features, basic.Trajectory.PathEfficiency)
	features = append(features, basic.Trajectory.AverageSpeed)
	features = append(features, basic.Trajectory.CurvatureAvg)
	features = append(features, basic.Trajectory.JitterScore)
	features = append(features, float64(basic.Trajectory.PauseCount))
	features = append(features, float64(basic.Trajectory.MicroCorrections))
	features = append(features, basic.ClickPattern.Regularity)
	features = append(features, basic.ClickPattern.PositionEntropy)
	
	// 高级特征
	if enhanced != nil {
		features = append(features, enhanced.CurvatureVariance)
		features = append(features, float64(enhanced.CurvaturePeaks))
		features = append(features, enhanced.CurvatureEntropy)
		features = append(features, enhanced.ClickIntervalSkewness)
		features = append(features, enhanced.ClickIntervalKurtosis)
		features = append(features, enhanced.DirectionChangeFrequency)
		features = append(features, enhanced.DirectionChangeRegularity)
		features = append(features, enhanced.FourierDominantFrequency)
		features = append(features, enhanced.FourierEnergyRatio)
		features = append(features, enhanced.FractalDimension)
		features = append(features, float64(enhanced.HoverCount))
		features = append(features, enhanced.HoverDurationVariance)
		features = append(features, enhanced.KeyPressIntervalRegularity)
		features = append(features, enhanced.PauseFrequency)
		features = append(features, enhanced.JerkAverage)
		features = append(features, enhanced.JerkVariance)
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
		mdRisk.BehaviorRisk * mdRisk.RiskContributors["behavior"] +
		mdRisk.EnvironmentalRisk * mdRisk.RiskContributors["environmental"] +
		mdRisk.HistoricalRisk * mdRisk.RiskContributors["historical"] +
		anomalyScore * 100 * mdRisk.RiskContributors["anomaly"]
	
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
