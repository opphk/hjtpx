#!/bin/bash

echo "========================================="
echo "HJTPX 监控配置验证报告"
echo "========================================="
echo ""

echo "1. 检查监控目录结构..."
echo "----------------------------------------"
monitoring_dirs=(
    "monitoring/prometheus/rules"
    "monitoring/grafana/provisioning/dashboards"
    "monitoring/grafana/provisioning/datasources"
    "monitoring/alertmanager/template"
    "monitoring/loki"
    "monitoring/promtail"
)

for dir in "${monitoring_dirs[@]}"; do
    if [ -d "$dir" ]; then
        echo "✓ $dir"
    else
        echo "✗ $dir (不存在)"
    fi
done
echo ""

echo "2. 检查配置文件..."
echo "----------------------------------------"
config_files=(
    "monitoring/prometheus/prometheus.yml"
    "monitoring/prometheus/rules/hjtpx.rules"
    "monitoring/alertmanager/alertmanager.yml"
    "monitoring/alertmanager/template/default.tmpl"
    "monitoring/grafana/provisioning/dashboards/hjtpx-dashboard.json"
    "monitoring/grafana/provisioning/dashboards/hjtpx-dashboard-extended.json"
    "monitoring/grafana/provisioning/datasources/datasources.yml"
    "monitoring/loki/loki.yml"
    "monitoring/promtail/promtail.yml"
)

for file in "${config_files[@]}"; do
    if [ -f "$file" ]; then
        size=$(stat -f%z "$file" 2>/dev/null || stat -c%s "$file" 2>/dev/null)
        echo "✓ $file (${size} bytes)"
    else
        echo "✗ $file (不存在)"
    fi
done
echo ""

echo "3. Prometheus配置统计..."
echo "----------------------------------------"
if [ -f "monitoring/prometheus/prometheus.yml" ]; then
    scrape_jobs=$(grep -c "job_name:" monitoring/prometheus/prometheus.yml || echo "0")
    echo "Scrape Jobs: $scrape_jobs"
fi
echo ""

echo "4. 告警规则统计..."
echo "----------------------------------------"
if [ -f "monitoring/prometheus/rules/hjtpx.rules" ]; then
    alert_groups=$(grep -c "name:" monitoring/prometheus/rules/hjtpx.rules || echo "0")
    total_alerts=$(grep -c "alert:" monitoring/prometheus/rules/hjtpx.rules || echo "0")
    echo "告警组: $alert_groups"
    echo "告警规则总数: $total_alerts"
    echo ""
    echo "告警分组:"
    grep "name:" monitoring/prometheus/rules/hjtpx.rules | sed 's/.*name: //' | sed 's/^/  - /'
fi
echo ""

echo "5. AlertManager接收器统计..."
echo "----------------------------------------"
if [ -f "monitoring/alertmanager/alertmanager.yml" ]; then
    receivers=$(grep -c "name:" monitoring/alertmanager/alertmanager.yml || echo "0")
    echo "接收器总数: $receivers"
    echo ""
    echo "接收器列表:"
    grep "name:" monitoring/alertmanager/alertmanager.yml | sed 's/.*name: //' | sed 's/^/  - /'
fi
echo ""

echo "6. Grafana仪表盘面板统计..."
echo "----------------------------------------"
if [ -f "monitoring/grafana/provisioning/dashboards/hjtpx-dashboard-extended.json" ]; then
    total_panels=$(grep -o '"title":' monitoring/grafana/provisioning/dashboards/hjtpx-dashboard-extended.json | wc -l)
    rows=$(grep -o '"type": "row"' monitoring/grafana/provisioning/dashboards/hjtpx-dashboard-extended.json | wc -l)
    echo "面板总数: $total_panels"
    echo "行分组: $rows"
    echo ""
    echo "仪表盘板块:"
    grep '"title":' monitoring/grafana/provisioning/dashboards/hjtpx-dashboard-extended.json | grep -v "Annotations" | sed 's/.*"title": "//' | sed 's/",//' | sed 's/^/  - /'
fi
echo ""

echo "7. Prometheus指标统计..."
echo "----------------------------------------"
if [ -f "backend/pkg/metrics/metrics.go" ]; then
    total_metrics=$(grep -c "prometheus.New" backend/pkg/metrics/metrics.go || echo "0")
    counter_metrics=$(grep -c "prometheus.NewCounter" backend/pkg/metrics/metrics.go || echo "0")
    gauge_metrics=$(grep -c "prometheus.NewGauge" backend/pkg/metrics/metrics.go || echo "0")
    histogram_metrics=$(grep -c "prometheus.NewHistogram" backend/pkg/metrics/metrics.go || echo "0")
    vec_metrics=$(grep -c "prometheus.New.*Vec" backend/pkg/metrics/metrics.go || echo "0")

    echo "指标定义总数: $total_metrics"
    echo "  - Counter指标: $counter_metrics"
    echo "  - Gauge指标: $gauge_metrics"
    echo "  - Histogram指标: $histogram_metrics"
    echo "  - Vec指标: $vec_metrics"
    echo ""
    echo "指标分类:"
    echo "  - HTTP请求指标"
    echo "  - 数据库连接指标"
    echo "  - 缓存指标"
    echo "  - 验证码指标"
    echo "  - 安全指标"
    echo "  - 认证指标"
    echo "  - WebSocket指标"
    echo "  - 业务应用指标"
    echo "  - Bot检测指标"
    echo "  - 风险评分指标"
fi
echo ""

echo "8. 测试文件统计..."
echo "----------------------------------------"
if [ -f "backend/internal/monitoring/monitoring_test.go" ]; then
    test_functions=$(grep -c "func Test" backend/internal/monitoring/monitoring_test.go || echo "0")
    echo "测试函数总数: $test_functions"
    echo ""
    echo "测试覆盖:"
    grep "func Test" backend/internal/monitoring/monitoring_test.go | sed 's/func //' | sed 's/(.*//' | sed 's/^/  - /'
fi
echo ""

echo "========================================="
echo "验证完成"
echo "========================================="
