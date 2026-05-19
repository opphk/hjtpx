/**
 * 高级分析面板测试用例
 */

const ANALYTICS_TESTS = (function() {
    const testResults = [];

    const runAllTests = function() {
        console.log('=== 开始运行高级分析面板测试 ===');
        
        testRealtimeDashboard();
        testCustomReports();
        testDataExport();
        testPredictiveAnalytics();
        testUtilityFunctions();

        printTestResults();
        return testResults;
    };

    const testRealtimeDashboard = function() {
        console.log('\n--- 实时仪表板测试 ---');

        test('初始化实时请求图表', function() {
            const ctx = document.createElement('canvas');
            ctx.id = 'test-chart';
            document.body.appendChild(ctx);
            
            const chart = new Chart(ctx, {
                type: 'line',
                data: { labels: [], datasets: [{ data: [] }] }
            });
            
            assert(chart instanceof Chart, '图表实例创建成功');
            assert(chart.config.type === 'line', '图表类型正确');
            
            document.body.removeChild(ctx);
            chart.destroy();
        });

        test('模拟实时数据更新', function() {
            const payload = {
                totalRequests: 854321,
                successRate: '94.7',
                avgResponseTime: 58,
                blockedAttacks: 12345
            };
            
            assert(typeof payload.totalRequests === 'number', '总请求数应为数字');
            assert(typeof payload.successRate === 'string', '成功率应为字符串');
            assert(payload.avgResponseTime > 0, '响应时间应大于0');
            assert(payload.blockedAttacks >= 0, '拦截数应大于等于0');
        });

        test('请求趋势数据生成', function() {
            const data = generateRequestTrendData();
            
            assert(Array.isArray(data), '返回应为数组');
            assert(data.length === 20, '数据点数量应为20个');
            assert(data[0].hasOwnProperty('time'), '数据点应包含时间字段');
            assert(data[0].hasOwnProperty('value'), '数据点应包含数值字段');
        });
    };

    const testCustomReports = function() {
        console.log('\n--- 自定义报表测试 ---');

        test('报表配置验证', function() {
            const config = {
                id: 'config-test',
                name: '测试报表',
                metrics: ['totalRequests', 'successRate'],
                timeRange: { type: 'weekly' },
                visualization: 'dashboard'
            };
            
            assert(config.name.length > 0, '报表名称不能为空');
            assert(Array.isArray(config.metrics), '指标应为数组');
            assert(['daily', 'weekly', 'monthly', 'custom'].includes(config.timeRange.type), '时间范围类型有效');
        });

        test('报表配置列表渲染', function() {
            const configs = getMockReportConfigs();
            
            assert(configs.length >= 1, '至少应有一个报表配置');
            configs.forEach(config => {
                assert(config.hasOwnProperty('id'), '配置必须有ID');
                assert(config.hasOwnProperty('name'), '配置必须有名称');
            });
        });

        test('定时发送配置', function() {
            const schedule = {
                enabled: true,
                frequency: 'daily',
                time: '09:00',
                email: 'test@example.com'
            };
            
            assert(['daily', 'weekly', 'monthly'].includes(schedule.frequency), '发送频率有效');
            assert(/^\d{2}:\d{2}$/.test(schedule.time), '发送时间格式正确');
            assert(schedule.email.includes('@'), '邮箱格式正确');
        });
    };

    const testDataExport = function() {
        console.log('\n--- 数据导出测试 ---');

        test('导出字段验证', function() {
            const fields = ['timestamp', 'requestId', 'captchaType', 'result'];
            const validFields = ['timestamp', 'requestId', 'userId', 'captchaType', 'result', 'responseTime', 'ipAddress', 'riskScore'];
            
            fields.forEach(field => {
                assert(validFields.includes(field), `${field} 是有效的导出字段`);
            });
        });

        test('CSV转换', function() {
            const data = [
                { name: '测试1', value: 100 },
                { name: '测试2', value: 200 }
            ];
            const csv = convertToCSV(data);
            
            assert(csv.includes('name,value'), 'CSV包含表头');
            assert(csv.includes('测试1'), 'CSV包含数据');
            assert(csv.includes('测试2'), 'CSV包含数据');
        });

        test('日期范围验证', function() {
            const startDate = '2024-01-01';
            const endDate = '2024-01-15';
            
            assert(/^\d{4}-\d{2}-\d{2}$/.test(startDate), '开始日期格式正确');
            assert(/^\d{4}-\d{2}-\d{2}$/.test(endDate), '结束日期格式正确');
            assert(startDate <= endDate, '开始日期应小于等于结束日期');
        });

        test('导出格式支持', function() {
            const formats = ['excel', 'csv', 'json', 'pdf'];
            const formatLabels = getFormatLabels();
            
            formats.forEach(format => {
                assert(formatLabels.hasOwnProperty(format), `${format} 格式有对应的标签`);
            });
        });
    };

    const testPredictiveAnalytics = function() {
        console.log('\n--- 预测分析测试 ---');

        test('预测数据结构', function() {
            const forecast = {
                requests: 1200000,
                successRate: 96.5,
                attacks: 8520,
                riskLevel: 'medium'
            };
            
            assert(typeof forecast.requests === 'number', '请求量应为数字');
            assert(forecast.successRate >= 0 && forecast.successRate <= 100, '成功率应在0-100之间');
            assert(['low', 'medium', 'high'].includes(forecast.riskLevel), '风险等级有效');
        });

        test('异常预测验证', function() {
            const anomaly = {
                time: '明天 02:00-04:00',
                type: 'spike',
                description: '请求量异常峰值',
                severity: 'high'
            };
            
            assert(['spike', 'pattern', 'trend'].includes(anomaly.type), '异常类型有效');
            assert(['high', 'medium', 'low'].includes(anomaly.severity), '严重等级有效');
            assert(anomaly.description.length > 0, '描述不能为空');
        });

        test('智能建议优先级', function() {
            const recommendations = [
                { priority: 'high', text: '高优先级建议' },
                { priority: 'medium', text: '中优先级建议' },
                { priority: 'low', text: '低优先级建议' }
            ];
            
            recommendations.forEach(rec => {
                assert(['high', 'medium', 'low'].includes(rec.priority), '优先级有效');
                assert(rec.text.length > 0, '建议内容不能为空');
            });
        });
    };

    const testUtilityFunctions = function() {
        console.log('\n--- 工具函数测试 ---');

        test('数字格式化', function() {
            assert(formatNumber(1234) === '1.2K', '千位数格式化正确');
            assert(formatNumber(1234567) === '1.2M', '百万位数格式化正确');
            assert(formatNumber(123) === '123', '普通数字格式化正确');
        });

        test('HTML转义', function() {
            const unsafe = '<script>alert("test")</script>';
            const safe = escapeHtml(unsafe);
            
            assert(!safe.includes('<script'), 'HTML标签被转义');
            assert(safe.includes('&lt;'), '使用HTML实体');
        });

        test('日期标签生成', function() {
            const labels = generateDayLabels();
            
            assert(labels.length === 11, '生成11个日期标签');
            labels.forEach(label => {
                assert(/^\d+\/\d+$/.test(label), '日期格式正确');
            });
        });
    };

    const test = function(name, fn) {
        try {
            fn();
            console.log(`✓ ${name}`);
            testResults.push({ name, passed: true });
        } catch (error) {
            console.log(`✗ ${name}: ${error.message}`);
            testResults.push({ name, passed: false, error: error.message });
        }
    };

    const assert = function(condition, message) {
        if (!condition) {
            throw new Error(message || '断言失败');
        }
    };

    const generateRequestTrendData = function() {
        const data = [];
        const now = new Date();
        for (let i = 19; i >= 0; i--) {
            const time = new Date(now.getTime() - i * 60000);
            data.push({
                time: time.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }),
                value: Math.floor(1000 + Math.random() * 2000)
            });
        }
        return data;
    };

    const getMockReportConfigs = function() {
        return [
            { id: 'config-1', name: '测试报表', metrics: ['totalRequests'], timeRange: { type: 'daily' }, visualization: 'dashboard' }
        ];
    };

    const convertToCSV = function(data) {
        if (!data.length) return '';
        const headers = Object.keys(data[0]);
        return [headers.join(','), ...data.map(row => headers.map(h => `"${row[h] || ''}"`).join(','))].join('\n');
    };

    const getFormatLabels = function() {
        return { excel: 'Excel', csv: 'CSV', pdf: 'PDF', json: 'JSON' };
    };

    const formatNumber = function(num) {
        if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';
        if (num >= 1000) return (num / 1000).toFixed(1) + 'K';
        return num.toString();
    };

    const escapeHtml = function(text) {
        if (!text) return '';
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    };

    const generateDayLabels = function() {
        const labels = [];
        for (let i = 7; i >= 0; i--) {
            const date = new Date();
            date.setDate(date.getDate() - i);
            labels.push(`${date.getMonth() + 1}/${date.getDate()}`);
        }
        for (let i = 1; i <= 3; i++) {
            const date = new Date();
            date.setDate(date.getDate() + i);
            labels.push(`${date.getMonth() + 1}/${date.getDate()}`);
        }
        return labels;
    };

    const printTestResults = function() {
        const passed = testResults.filter(t => t.passed).length;
        const total = testResults.length;
        
        console.log('\n=== 测试结果 ===');
        console.log(`通过: ${passed}/${total}`);
        
        if (passed < total) {
            console.log('\n失败的测试:');
            testResults.filter(t => !t.passed).forEach(t => {
                console.log(`  - ${t.name}: ${t.error}`);
            });
        }
    };

    return {
        runAllTests,
        testResults
    };
})();

// 导出测试函数供外部调用
if (typeof module !== 'undefined' && module.exports) {
    module.exports = ANALYTICS_TESTS;
}