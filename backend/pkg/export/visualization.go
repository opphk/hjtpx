package export

import (
	"bytes"
	"fmt"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/hjtpx/hjtpx/pkg/models"
)

// VisualizationData 可视化数据
type VisualizationData struct {
	Title string
	Charts []ChartData
}

// ChartData 单个图表数据
type ChartData struct {
	Title string
	Type string
	Labels []string
	Values []interface{}
}

// VisualizationExporter 可视化导出器
type VisualizationExporter struct{}

// NewVisualizationExporter 创建可视化导出器
func NewVisualizationExporter() *VisualizationExporter {
	return &VisualizationExporter{}
}

// ExportHTML 导出HTML格式的可视化报告
func (v *VisualizationExporter) ExportHTML(data VisualizationData) ([]byte, error) {
	var buf bytes.Buffer
	
	// 创建页面
	_, _ = buf.WriteString(fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>%s</title>
    <script src="https://cdn.jsdelivr.net/npm/echarts@5.4.3/dist/echarts.min.js"></script>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .chart-container { margin: 30px 0; }
        .chart-title { font-size: 18px; font-weight: bold; margin-bottom: 10px; }
        .chart { width: 100%%; height: 400px; }
        .report-title { text-align: center; font-size: 24px; margin-bottom: 30px; }
        .export-time { text-align: right; color: #666; margin-bottom: 20px; }
    </style>
</head>
<body>
    <h1 class="report-title">%s</h1>
    <div class="export-time">Export Time: %s</div>
`, data.Title, data.Title, time.Now().Format("2006-01-02 15:04:05")))
	
	// 生成图表
	for i, chart := range data.Charts {
		chartID := fmt.Sprintf("chart_%d", i)
		_, _ = buf.WriteString(fmt.Sprintf(`
    <div class="chart-container">
        <div class="chart-title">%s</div>
        <div id="%s" class="chart"></div>
    </div>
    <script>
        var chartDom%d = document.getElementById('%s');
        var myChart%d = echarts.init(chartDom%d);
        var option%d = %s;
        myChart%d.setOption(option%d);
    </script>
`, chart.Title, chartID, i, chartID, i, i, i, generateEChartsOption(chart), i, i))
	}
	
	_, _ = buf.WriteString(`
</body>
</html>`)
	
	return buf.Bytes(), nil
}

// GenerateLogVisualization 生成日志数据的可视化
func GenerateLogVisualization(logs []models.VerificationLog, title string) VisualizationData {
	// 按状态统计
	statusStats := make(map[string]int)
	// 按验证码类型统计
	typeStats := make(map[string]int)
	// 按日期统计
	dateStats := make(map[string]int)

	for _, log := range logs {
		statusStats[log.Status]++
		typeStats[log.CaptchaType]++
		date := log.CreatedAt.Format("2006-01-02")
		dateStats[date]++
	}

	var charts []ChartData

	// 状态分布图
	charts = append(charts, createPieChart("Verification Status Distribution", statusStats))

	// 验证码类型统计图
	charts = append(charts, createBarChart("Captcha Type Distribution", typeStats))

	// 日期趋势图
	charts = append(charts, createLineChart("Daily Verification Trend", dateStats))

	return VisualizationData{
		Title: title,
		Charts: charts,
	}
}

// 使用go-echarts创建图表
func (v *VisualizationExporter) ExportWithECharts(data VisualizationData) ([]byte, error) {
	// 这里可以使用go-echarts生成更复杂的图表
	// 为了简化，我们使用上面的HTML导出方式
	return v.ExportHTML(data)
}

func generateEChartsOption(chart ChartData) string {
	switch chart.Type {
	case "pie":
		seriesData := make([]map[string]interface{}, len(chart.Labels))
		for i, label := range chart.Labels {
			seriesData[i] = map[string]interface{}{
				"value": chart.Values[i],
				"name": label,
			}
		}
		return fmt.Sprintf(`{
			title: { text: '%s', left: 'center' },
			tooltip: { trigger: 'item' },
			series: [{
				type: 'pie',
				radius: '50%%',
				data: %s
			}]
		}`, chart.Title, toJSON(seriesData))
	
	case "line":
		return fmt.Sprintf(`{
			title: { text: '%s', left: 'center' },
			tooltip: { trigger: 'axis' },
			xAxis: { type: 'category', data: %s },
			yAxis: { type: 'value' },
			series: [{
				type: 'line',
				data: %s,
				smooth: true
			}]
		}`, chart.Title, toJSON(chart.Labels), toJSON(chart.Values))
	
	default: // bar
		return fmt.Sprintf(`{
			title: { text: '%s', left: 'center' },
			tooltip: { trigger: 'axis' },
			xAxis: { type: 'category', data: %s },
			yAxis: { type: 'value' },
			series: [{
				type: 'bar',
				data: %s
			}]
		}`, chart.Title, toJSON(chart.Labels), toJSON(chart.Values))
	}
}

func createPieChart(title string, data map[string]int) ChartData {
	labels := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))
	for k, v := range data {
		labels = append(labels, k)
		values = append(values, v)
	}
	return ChartData{
		Title: title,
		Type: "pie",
		Labels: labels,
		Values: values,
	}
}

func createBarChart(title string, data map[string]int) ChartData {
	labels := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))
	for k, v := range data {
		labels = append(labels, k)
		values = append(values, v)
	}
	return ChartData{
		Title: title,
		Type: "bar",
		Labels: labels,
		Values: values,
	}
}

func createLineChart(title string, data map[string]int) ChartData {
	labels := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))
	// 按日期排序
	for k, v := range data {
		labels = append(labels, k)
		values = append(values, v)
	}
	return ChartData{
		Title: title,
		Type: "line",
		Labels: labels,
		Values: values,
	}
}

func toJSON(v interface{}) string {
	// 简化的JSON转换
	switch val := v.(type) {
	case []string:
		result := "["
		for i, s := range val {
			if i > 0 {
				result += ","
			}
			result += fmt.Sprintf(`"%s"`, s)
		}
		result += "]"
		return result
	case []interface{}:
		result := "["
		for i, s := range val {
			if i > 0 {
				result += ","
			}
			result += fmt.Sprintf("%v", s)
		}
		result += "]"
		return result
	case []map[string]interface{}:
		result := "["
		for i, m := range val {
			if i > 0 {
				result += ","
			}
			result += "{"
			j := 0
			for k, v := range m {
				if j > 0 {
					result += ","
				}
				result += fmt.Sprintf(`"%s":%v`, k, v)
				j++
			}
			result += "}"
		}
		result += "]"
		return result
	default:
		return fmt.Sprintf("%v", val)
	}
}

// CreateGoEChartsBar 使用go-echarts创建柱状图
func CreateGoEChartsBar(title string, data map[string]int) *charts.Bar {
	bar := charts.NewBar()
	bar.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{Title: title}),
	)

	labels := make([]string, 0, len(data))
	values := make([]opts.BarData, 0, len(data))
	for k, v := range data {
		labels = append(labels, k)
		values = append(values, opts.BarData{Value: v})
	}

	bar.SetXAxis(labels).AddSeries("Count", values)
	return bar
}

// CreateGoEChartsPie 使用go-echarts创建饼图
func CreateGoEChartsPie(title string, data map[string]int) *charts.Pie {
	pie := charts.NewPie()
	pie.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{Title: title}),
	)

	var values []opts.PieData
	for k, v := range data {
		values = append(values, opts.PieData{Name: k, Value: v})
	}

	pie.AddSeries("Distribution", values)
	return pie
}
