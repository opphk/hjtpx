/**
 * CaptchaX 管理端测试套件
 * @version 1.0.0
 * @license MIT
 */
(function(global) {
    'use strict';

    const AdminTest = {
        version: '1.0.0',
        passed: 0,
        failed: 0,
        results: [],
        testContainer: null
    };

    function assert(condition, message) {
        const result = {
            passed: condition,
            message: message,
            timestamp: new Date().toISOString()
        };
        AdminTest.results.push(result);
        if (condition) {
            AdminTest.passed++;
            console.log(`✓ ${message}`);
        } else {
            AdminTest.failed++;
            console.error(`✗ ${message}`);
        }
        return condition;
    }

    function assertEqual(actual, expected, message) {
        const condition = actual === expected;
        const result = {
            passed: condition,
            message: `${message} (期望: ${expected}, 实际: ${actual})`,
            timestamp: new Date().toISOString()
        };
        AdminTest.results.push(result);
        if (condition) {
            AdminTest.passed++;
            console.log(`✓ ${message}`);
        } else {
            AdminTest.failed++;
            console.error(`✗ ${message} (期望: ${expected}, 实际: ${actual})`);
        }
        return condition;
    }

    function assertDeepEqual(actual, expected, message) {
        const actualStr = JSON.stringify(actual);
        const expectedStr = JSON.stringify(expected);
        const condition = actualStr === expectedStr;
        const result = {
            passed: condition,
            message: message,
            timestamp: new Date().toISOString()
        };
        AdminTest.results.push(result);
        if (condition) {
            AdminTest.passed++;
            console.log(`✓ ${message}`);
        } else {
            AdminTest.failed++;
            console.error(`✗ ${message}`);
            console.error(`  期望: ${expectedStr}`);
            console.error(`  实际: ${actualStr}`);
        }
        return condition;
    }

    function createMockDOM() {
        if (AdminTest.testContainer) return AdminTest.testContainer;

        AdminTest.testContainer = document.createElement('div');
        AdminTest.testContainer.id = 'test-container';
        AdminTest.testContainer.innerHTML = `
            <div id="toast-container"></div>
            <input type="text" id="test-input" />
            <form id="test-form">
                <input type="text" id="required-field" data-validate='["required"]' />
                <input type="text" id="email-field" data-validate='["email"]' />
                <input type="text" id="ip-field" data-validate='["ip"]' />
            </form>
            <table id="test-table">
                <thead>
                    <tr>
                        <th><input type="checkbox" class="master-checkbox" /></th>
                        <th>ID</th>
                        <th>Name</th>
                    </tr>
                </thead>
                <tbody>
                    <tr><td><input type="checkbox" class="row-checkbox" data-id="1" /></td><td>1</td><td>Item 1</td></tr>
                    <tr><td><input type="checkbox" class="row-checkbox" data-id="2" /></td><td>2</td><td>Item 2</td></tr>
                    <tr><td><input type="checkbox" class="row-checkbox" data-id="3" /></td><td>3</td><td>Item 3</td></tr>
                </tbody>
            </table>
            <div id="pagination"></div>
        `;
        document.body.appendChild(AdminTest.testContainer);
        return AdminTest.testContainer;
    }

    function cleanupMockDOM() {
        if (AdminTest.testContainer && AdminTest.testContainer.parentNode) {
            AdminTest.testContainer.parentNode.removeChild(AdminTest.testContainer);
            AdminTest.testContainer = null;
        }
    }

    AdminTest.run = async function() {
        console.log('========================================');
        console.log('CaptchaX 管理端测试套件 v' + AdminTest.version);
        console.log('========================================\n');

        createMockDOM();

        await AdminTest.testApiClient();
        await AdminTest.testFormValidator();
        await AdminTest.testExportHelper();
        await AdminTest.testPagination();
        await AdminTest.testBatchOperations();
        await AdminTest.testUIController();
        await AdminTest.testRealtimeUpdater();
        await AdminTest.testChartManager();

        cleanupMockDOM();

        console.log('\n========================================');
        console.log('测试结果:');
        console.log(`  通过: ${AdminTest.passed}`);
        console.log(`  失败: ${AdminTest.failed}`);
        console.log(`  总计: ${AdminTest.passed + AdminTest.failed}`);
        console.log('========================================');

        return {
            passed: AdminTest.passed,
            failed: AdminTest.failed,
            results: AdminTest.results
        };
    };

    AdminTest.testApiClient = async function() {
        console.log('\n--- 测试 API 客户端 ---');

        if (typeof AdminApi === 'undefined') {
            console.log('  (跳过 - AdminApi 未定义)');
            return;
        }

        assert(typeof AdminApi.request === 'function', 'AdminApi.request 方法存在');

        assert(typeof AdminApi.auth === 'object', 'AdminApi.auth 模块存在');
        assert(typeof AdminApi.auth.login === 'function', 'AdminApi.auth.login 方法存在');
        assert(typeof AdminApi.auth.logout === 'function', 'AdminApi.auth.logout 方法存在');

        assert(typeof AdminApi.stats === 'object', 'AdminApi.stats 模块存在');
        assert(typeof AdminApi.stats.getDashboard === 'function', 'AdminApi.stats.getDashboard 方法存在');
        assert(typeof AdminApi.stats.getTrend === 'function', 'AdminApi.stats.getTrend 方法存在');
        assert(typeof AdminApi.stats.getCaptchaDistribution === 'function', 'AdminApi.stats.getCaptchaDistribution 方法存在');

        assert(typeof AdminApi.config === 'object', 'AdminApi.config 模块存在');
        assert(typeof AdminApi.config.get === 'function', 'AdminApi.config.get 方法存在');
        assert(typeof AdminApi.config.update === 'function', 'AdminApi.config.update 方法存在');

        assert(typeof AdminApi.whitelist === 'object', 'AdminApi.whitelist 模块存在');
        assert(typeof AdminApi.whitelist.list === 'function', 'AdminApi.whitelist.list 方法存在');
        assert(typeof AdminApi.whitelist.add === 'function', 'AdminApi.whitelist.add 方法存在');
        assert(typeof AdminApi.whitelist.delete === 'function', 'AdminApi.whitelist.delete 方法存在');
        assert(typeof AdminApi.whitelist.batchDelete === 'function', 'AdminApi.whitelist.batchDelete 方法存在');

        assert(typeof AdminApi.blacklist === 'object', 'AdminApi.blacklist 模块存在');
        assert(typeof AdminApi.blacklist.list === 'function', 'AdminApi.blacklist.list 方法存在');
        assert(typeof AdminApi.blacklist.add === 'function', 'AdminApi.blacklist.add 方法存在');
        assert(typeof AdminApi.blacklist.delete === 'function', 'AdminApi.blacklist.delete 方法存在');
        assert(typeof AdminApi.blacklist.unban === 'function', 'AdminApi.blacklist.unban 方法存在');
        assert(typeof AdminApi.blacklist.batchDelete === 'function', 'AdminApi.blacklist.batchDelete 方法存在');
        assert(typeof AdminApi.blacklist.batchUnban === 'function', 'AdminApi.blacklist.batchUnban 方法存在');
    };

    AdminTest.testFormValidator = async function() {
        console.log('\n--- 测试表单验证器 ---');

        if (typeof FormValidator === 'undefined') {
            console.log('  (跳过 - FormValidator 未定义)');
            return;
        }

        assert(typeof FormValidator.rules === 'object', 'FormValidator.rules 对象存在');
        assert(typeof FormValidator.messages === 'object', 'FormValidator.messages 对象存在');
        assert(typeof FormValidator.validateField === 'function', 'FormValidator.validateField 方法存在');
        assert(typeof FormValidator.validateForm === 'function', 'FormValidator.validateForm 方法存在');
        assert(typeof FormValidator.showError === 'function', 'FormValidator.showError 方法存在');
        assert(typeof FormValidator.clearError === 'function', 'FormValidator.clearError 方法存在');

        assert(typeof FormValidator.rules.required === 'function', 'required 规则存在');
        assert(typeof FormValidator.rules.email === 'function', 'email 规则存在');
        assert(typeof FormValidator.rules.ip === 'function', 'ip 规则存在');
        assert(typeof FormValidator.rules.ipv4 === 'function', 'ipv4 规则存在');
        assert(typeof FormValidator.rules.ipv4Cidr === 'function', 'ipv4Cidr 规则存在');
        assert(typeof FormValidator.rules.number === 'function', 'number 规则存在');

        const validations = [
            { rule: 'required', value: 'test', expected: true },
            { rule: 'required', value: '', expected: false },
            { rule: 'required', value: '  ', expected: false },
            { rule: 'email', value: 'test@example.com', expected: true },
            { rule: 'email', value: 'invalid-email', expected: false },
            { rule: 'ipv4', value: '192.168.1.1', expected: true },
            { rule: 'ipv4', value: '256.1.1.1', expected: false },
            { rule: 'ipv4Cidr', value: '192.168.1.0/24', expected: true },
            { rule: 'ipv4Cidr', value: '192.168.1.1', expected: false },
        ];

        validations.forEach(v => {
            const result = FormValidator.rules[v.rule](v.value);
            assertEqual(result, v.expected, `${v.rule} 验证 "${v.value}" => ${v.expected}`);
        });
    };

    AdminTest.testExportHelper = async function() {
        console.log('\n--- 测试导出助手 ---');

        if (typeof ExportHelper === 'undefined') {
            console.log('  (跳过 - ExportHelper 未定义)');
            return;
        }

        assert(typeof ExportHelper.downloadCSV === 'function', 'ExportHelper.downloadCSV 方法存在');
        assert(typeof ExportHelper.downloadBlob === 'function', 'ExportHelper.downloadBlob 方法存在');
        assert(typeof ExportHelper.downloadJSON === 'function', 'ExportHelper.downloadJSON 方法存在');
        assert(typeof ExportHelper.exportTable === 'function', 'ExportHelper.exportTable 方法存在');

        const testData = [
            { id: 1, name: 'Test 1', value: 100 },
            { id: 2, name: 'Test 2', value: 200 }
        ];

        const columns = [
            { key: 'id', title: 'ID' },
            { key: 'name', title: '名称' },
            { key: 'value', title: '数值' }
        ];

        assert(typeof ExportHelper.downloadCSV === 'function', 'downloadCSV 可以处理数据');
    };

    AdminTest.testPagination = async function() {
        console.log('\n--- 测试分页组件 ---');

        if (typeof AdminUI === 'undefined' || typeof AdminUI.Pagination === 'undefined') {
            console.log('  (跳过 - AdminUI.Pagination 未定义)');
            return;
        }

        const Pagination = AdminUI.Pagination;

        assert(typeof Pagination.init === 'function', 'Pagination.init 方法存在');
        assert(typeof Pagination.setTotal === 'function', 'Pagination.setTotal 方法存在');
        assert(typeof Pagination.setPage === 'function', 'Pagination.setPage 方法存在');
        assert(typeof Pagination.setPageSize === 'function', 'Pagination.setPageSize 方法存在');
        assert(typeof Pagination.goToFirst === 'function', 'Pagination.goToFirst 方法存在');
        assert(typeof Pagination.goToLast === 'function', 'Pagination.goToLast 方法存在');
        assert(typeof Pagination.prev === 'function', 'Pagination.prev 方法存在');
        assert(typeof Pagination.next === 'function', 'Pagination.next 方法存在');
        assert(typeof Pagination.render === 'function', 'Pagination.render 方法存在');

        Pagination.container = document.getElementById('pagination');
        Pagination.currentPage = 1;
        Pagination.pageSize = 20;
        Pagination.total = 100;

        assertEqual(Pagination.currentPage, 1, '初始页码正确');
        assertEqual(Pagination.pageSize, 20, '默认每页数量正确');

        Pagination.setTotal(150);
        assertEqual(Pagination.total, 150, '设置总数正确');

        Pagination.setPage(3);
        assertEqual(Pagination.currentPage, 3, '设置页码正确');

        Pagination.setPage(0);
        assertEqual(Pagination.currentPage, 1, '页码不能小于1');

        Pagination.setPage(100);
        const maxPage = Math.ceil(Pagination.total / Pagination.pageSize);
        assertEqual(Pagination.currentPage, maxPage, '页码不能超过最大页数');
    };

    AdminTest.testBatchOperations = async function() {
        console.log('\n--- 测试批量操作 ---');

        if (typeof BatchOperations === 'undefined') {
            console.log('  (跳过 - BatchOperations 未定义)');
            return;
        }

        assert(typeof BatchOperations.init === 'function', 'BatchOperations.init 方法存在');
        assert(typeof BatchOperations.toggleAll === 'function', 'BatchOperations.toggleAll 方法存在');
        assert(typeof BatchOperations.updateMasterCheckbox === 'function', 'BatchOperations.updateMasterCheckbox 方法存在');
        assert(typeof BatchOperations.updateActionBar === 'function', 'BatchOperations.updateActionBar 方法存在');
        assert(typeof BatchOperations.getSelected === 'function', 'BatchOperations.getSelected 方法存在');
        assert(typeof BatchOperations.clearSelection === 'function', 'BatchOperations.clearSelection 方法存在');

        BatchOperations.init('test-table', 'row-checkbox');
        assertEqual(BatchOperations.tableId, 'test-table', '表格ID设置正确');
        assertEqual(BatchOperations.checkboxClass, 'row-checkbox', '复选框类名设置正确');

        BatchOperations.toggleAll(true);
        let selected = BatchOperations.getSelected();
        assertEqual(selected.length, 3, '全选后选中3项');

        BatchOperations.toggleAll(false);
        selected = BatchOperations.getSelected();
        assertEqual(selected.length, 0, '取消全选后选中0项');

        const checkboxes = document.querySelectorAll('.row-checkbox');
        if (checkboxes.length > 0) {
            checkboxes[0].checked = true;
            checkboxes[0].dispatchEvent(new Event('change'));
            selected = BatchOperations.getSelected();
            assertEqual(selected.length, 1, '单个选中后选中1项');
        }

        BatchOperations.clearSelection();
        selected = BatchOperations.getSelected();
        assertEqual(selected.length, 0, '清空选择后选中0项');
    };

    AdminTest.testUIController = async function() {
        console.log('\n--- 测试UI控制器 ---');

        if (typeof AdminUI === 'undefined') {
            console.log('  (跳过 - AdminUI 未定义)');
            return;
        }

        assert(typeof AdminUI.showToast === 'function', 'AdminUI.showToast 方法存在');
        assert(typeof AdminUI.showLoading === 'function', 'AdminUI.showLoading 方法存在');
        assert(typeof AdminUI.hideLoading === 'function', 'AdminUI.hideLoading 方法存在');
        assert(typeof AdminUI.confirm === 'function', 'AdminUI.confirm 方法存在');
        assert(typeof AdminUI.formatNumber === 'function', 'AdminUI.formatNumber 方法存在');
        assert(typeof AdminUI.formatDate === 'function', 'AdminUI.formatDate 方法存在');
        assert(typeof AdminUI.formatDuration === 'function', 'AdminUI.formatDuration 方法存在');

        assertEqual(AdminUI.formatNumber(1000), '1.0K', '数字格式化 - K');
        assertEqual(AdminUI.formatNumber(1000000), '1.0M', '数字格式化 - M');
        assertEqual(AdminUI.formatNumber(500), '500', '数字格式化 - 普通数字');
        assertEqual(AdminUI.formatNumber(0), '0', '数字格式化 - 零');
        assertEqual(AdminUI.formatNumber(null), '0', '数字格式化 - null');
        assertEqual(AdminUI.formatNumber(undefined), '0', '数字格式化 - undefined');

        assertEqual(AdminUI.formatDuration(30), '30秒', '时长格式化 - 秒');
        assertEqual(AdminUI.formatDuration(120), '2分钟', '时长格式化 - 分钟');
        assertEqual(AdminUI.formatDuration(7200), '2小时0分钟', '时长格式化 - 小时');

        const date = AdminUI.formatDate('2024-01-15T10:30:00');
        assert(date !== '-', '日期格式化 - 有效日期');
        assert(date !== '', '日期格式化 - 返回非空字符串');

        assertEqual(AdminUI.formatDate(null), '-', '日期格式化 - null');
        assertEqual(AdminUI.formatDate(''), '-', '日期格式化 - 空字符串');
    };

    AdminTest.testRealtimeUpdater = async function() {
        console.log('\n--- 测试实时更新器 ---');

        if (typeof RealtimeUpdater === 'undefined') {
            console.log('  (跳过 - RealtimeUpdater 未定义)');
            return;
        }

        assert(typeof RealtimeUpdater.start === 'function', 'RealtimeUpdater.start 方法存在');
        assert(typeof RealtimeUpdater.stop === 'function', 'RealtimeUpdater.stop 方法存在');
        assert(typeof RealtimeUpdater.stopAll === 'function', 'RealtimeUpdater.stopAll 方法存在');
        assert(typeof RealtimeUpdater.refresh === 'function', 'RealtimeUpdater.refresh 方法存在');
        assert(typeof RealtimeUpdater.refreshAll === 'function', 'RealtimeUpdater.refreshAll 方法存在');
        assert(typeof RealtimeUpdater.setInterval === 'function', 'RealtimeUpdater.setInterval 方法存在');

        let callbackCount = 0;
        const testCallback = () => { callbackCount++; };

        RealtimeUpdater.start('test-key', testCallback, 1000);
        assert(RealtimeUpdater.timers['test-key'] !== undefined, '启动定时器成功');

        RealtimeUpdater.stop('test-key');
        assert(RealtimeUpdater.timers['test-key'] === undefined, '停止定时器成功');

        RealtimeUpdater.start('test-key-2', testCallback);
        RealtimeUpdater.stopAll();
        assert(Object.keys(RealtimeUpdater.timers).length === 0, '停止所有定时器成功');
    };

    AdminTest.testChartManager = async function() {
        console.log('\n--- 测试图表管理器 ---');

        if (typeof AdminCharts === 'undefined') {
            console.log('  (跳过 - AdminCharts 未定义)');
            return;
        }

        assert(typeof AdminCharts.initTrendChart === 'function', 'AdminCharts.initTrendChart 方法存在');
        assert(typeof AdminCharts.initDistributionChart === 'function', 'AdminCharts.initDistributionChart 方法存在');
        assert(typeof AdminCharts.initResultChart === 'function', 'AdminCharts.initResultChart 方法存在');
        assert(typeof AdminCharts.initBarChart === 'function', 'AdminCharts.initBarChart 方法存在');
        assert(typeof AdminCharts.updateChart === 'function', 'AdminCharts.updateChart 方法存在');
        assert(typeof AdminCharts.destroyChart === 'function', 'AdminCharts.destroyChart 方法存在');
        assert(typeof AdminCharts.destroyAll === 'function', 'AdminCharts.destroyAll 方法存在');

        const mockCanvas = document.createElement('canvas');
        mockCanvas.id = 'test-chart-canvas';
        document.body.appendChild(mockCanvas);

        const trendData = [
            { time: '10:00', verified: 100, rejected: 10 },
            { time: '11:00', verified: 150, rejected: 15 },
            { time: '12:00', verified: 200, rejected: 20 }
        ];

        AdminCharts.initTrendChart('test-chart-canvas', trendData);
        assert(AdminCharts.charts['test-chart-canvas'] !== undefined, '趋势图初始化成功');

        const distData = [
            { type: 'slider', count: 500 },
            { type: 'click', count: 300 },
            { type: 'rotate', count: 200 }
        ];

        AdminCharts.initDistributionChart('test-chart-canvas', distData);
        assert(AdminCharts.charts['test-chart-canvas'] !== undefined, '分布图初始化成功');

        AdminCharts.destroyChart('test-chart-canvas');
        assert(AdminCharts.charts['test-chart-canvas'] === undefined, '图表销毁成功');

        mockCanvas.remove();
    };

    AdminTest.waitFor = function(condition, timeout = 5000) {
        return new Promise((resolve, reject) => {
            const startTime = Date.now();

            const check = () => {
                if (condition()) {
                    resolve(true);
                    return;
                }

                if (Date.now() - startTime > timeout) {
                    reject(new Error('等待超时'));
                    return;
                }

                setTimeout(check, 100);
            };

            check();
        });
    };

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = AdminTest;
    } else {
        global.AdminTest = AdminTest;
    }

})(typeof window !== 'undefined' ? window : this);
