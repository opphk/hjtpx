let configEditor = null;
let envVars = [];
let backups = [];
let servicesStatus = [];
let batchOpsHistory = [];
let scheduledTasks = [];
let configHistory = [];
let envVarHistory = [];
let pendingChanges = [];

document.addEventListener('DOMContentLoaded', function() {
    initAceEditor();
    setupEventListeners();
    loadInitialData();
    startHealthCheckPolling();
});

function initAceEditor() {
    const editorContainer = document.getElementById('configEditor');
    if (!editorContainer) return;

    configEditor = ace.edit(editorContainer);
    configEditor.setTheme('ace/theme/github');
    configEditor.session.setMode('ace/mode/yaml');
    configEditor.setShowPrintMargin(false);
    configEditor.setFontSize(13);
    configEditor.setValue(getDefaultConfigContent('app'));

    const configFileSelect = document.getElementById('configFileSelect');
    if (configFileSelect) {
        configFileSelect.addEventListener('change', function() {
            configEditor.setValue(getDefaultConfigContent(this.value));
        });
    }
}

function getDefaultConfigContent(type) {
    const configs = {
        app: `# 应用配置
app:
  name: "墨盾验证系统"
  version: "2.1.3"
  environment: "production"
  
server:
  host: "0.0.0.0"
  port: 8080
  max_connections: 10000
  timeout: 30s
  
logging:
  level: "info"
  format: "json"
  output: "stdout"
  
verification:
  default_timeout: 20s
  max_attempts: 3
  enable_captcha: true`,
        security: `# 安全配置
security:
  enable_rate_limit: true
  rate_limit:
    requests_per_minute: 60
    burst: 10
    
  enable_ip_blacklist: true
  enable_fingerprint_check: true
  
  cors:
    allow_origins:
      - "https://example.com"
      - "https://www.example.com"
    allow_methods:
      - "GET"
      - "POST"
    allow_headers:
      - "Content-Type"
      - "Authorization"`,
        cache: `# 缓存配置
cache:
  enabled: true
  backend: "redis"
  
  redis:
    host: "127.0.0.1"
    port: 6379
    password: ""
    db: 0
    max_connections: 100
    
  ttl:
    default: 3600
    captcha: 300
    token: 7200`,
        database: `# 数据库配置
database:
  driver: "mysql"
  dsn: "root:password@tcp(localhost:3306)/hjtpx?charset=utf8mb4"
  
  max_open_conns: 100
  max_idle_conns: 10
  conn_max_lifetime: 3600s
  
  enable_log: true
  slow_threshold: 100ms`,
        logging: `# 日志配置
logging:
  level: "info"
  format: "json"
  
  outputs:
    - type: "stdout"
      format: "json"
    - type: "file"
      path: "/var/log/hjtpx/app.log"
      max_size: "100MB"
      max_backups: 30
      max_age: 7
      compress: true`
    };
    return configs[type] || configs.app;
}

function setupEventListeners() {
    const validateConfigBtn = document.getElementById('validateConfigBtn');
    if (validateConfigBtn) {
        validateConfigBtn.addEventListener('click', validateConfig);
    }

    const diffConfigBtn = document.getElementById('diffConfigBtn');
    if (diffConfigBtn) {
        diffConfigBtn.addEventListener('click', showConfigDiff);
    }

    const saveConfigBtn = document.getElementById('saveConfigBtn');
    if (saveConfigBtn) {
        saveConfigBtn.addEventListener('click', saveConfig);
    }

    const reloadConfigBtn = document.getElementById('reloadConfigBtn');
    if (reloadConfigBtn) {
        reloadConfigBtn.addEventListener('click', reloadConfig);
    }

    const runHealthCheckBtn = document.getElementById('runHealthCheckBtn');
    if (runHealthCheckBtn) {
        runHealthCheckBtn.addEventListener('click', runHealthCheck);
    }

    const addEnvVarBtn = document.getElementById('addEnvVarBtn');
    if (addEnvVarBtn) {
        addEnvVarBtn.addEventListener('click', showAddEnvVarModal);
    }

    const saveEnvVarBtn = document.getElementById('saveEnvVarBtn');
    if (saveEnvVarBtn) {
        saveEnvVarBtn.addEventListener('click', saveEnvVar);
    }

    const createBackupBtn = document.getElementById('createBackupBtn');
    if (createBackupBtn) {
        createBackupBtn.addEventListener('click', createBackup);
    }

    const refreshBackupsBtn = document.getElementById('refreshBackupsBtn');
    if (refreshBackupsBtn) {
        refreshBackupsBtn.addEventListener('click', loadBackups);
    }

    const searchEnvVar = document.getElementById('searchEnvVar');
    if (searchEnvVar) {
        searchEnvVar.addEventListener('input', filterEnvVars);
    }

    const filterEnvVarType = document.getElementById('filterEnvVarType');
    if (filterEnvVarType) {
        filterEnvVarType.addEventListener('change', filterEnvVars);
    }

    const showEnvVarValues = document.getElementById('showEnvVarValues');
    if (showEnvVarValues) {
        showEnvVarValues.addEventListener('change', toggleEnvVarValues);
    }
}

function loadInitialData() {
    loadConfigHistory();
    loadEnvVars();
    loadBackups();
    loadBatchOpsHistory();
    loadScheduledTasks();
    updateSystemMetrics();
}

function loadConfigHistory() {
    configHistory = [
        { time: '2024-01-15 14:30:25', file: 'app.yaml', action: '修改', operator: 'admin', status: 'success' },
        { time: '2024-01-15 10:15:00', file: 'security.yaml', action: '热更新', operator: 'admin', status: 'success' },
        { time: '2024-01-14 16:45:30', file: 'cache.yaml', action: '修改', operator: 'admin', status: 'success' },
        { time: '2024-01-14 09:00:00', file: 'database.yaml', action: '热更新', operator: 'admin', status: 'success' }
    ];
    renderConfigHistory();
}

function renderConfigHistory() {
    const tbody = document.getElementById('configHistoryTable');
    if (!tbody) return;

    tbody.innerHTML = configHistory.map(item => `
        <tr>
            <td><small>${item.time}</small></td>
            <td><code>${item.file}</code></td>
            <td><span class="badge bg-${item.action === '热更新' ? 'danger' : 'primary'}">${item.action}</span></td>
            <td>${item.operator}</td>
            <td><button class="btn btn-sm btn-outline-secondary"><i class="fas fa-undo"></i></button></td>
        </tr>
    `).join('');
}

function validateConfig() {
    const configContent = configEditor ? configEditor.getValue() : '';
    const resultDiv = document.getElementById('validationResult');
    const messageSpan = document.getElementById('validationMessage');

    try {
        const lines = configContent.split('\n');
        let isValid = true;
        let errors = [];

        lines.forEach((line, index) => {
            if (line.includes(':') && !line.trim().startsWith('#')) {
                const indent = line.search(/\S/);
                if (indent === -1) return;
                if (line.trim().endsWith(':')) return;

                const parts = line.split(':');
                if (parts.length < 2 || parts[1].trim() === '') {
                    errors.push(`第 ${index + 1} 行: 缺少值`);
                    isValid = false;
                }
            }
        });

        if (isValid) {
            resultDiv.className = 'alert alert-success';
            messageSpan.textContent = '配置验证通过！YAML 格式正确。';
        } else {
            resultDiv.className = 'alert alert-danger';
            messageSpan.textContent = errors.join('; ');
        }
        resultDiv.classList.remove('d-none');
    } catch (e) {
        resultDiv.className = 'alert alert-danger';
        messageSpan.textContent = '配置验证失败：' + e.message;
        resultDiv.classList.remove('d-none');
    }
}

function showConfigDiff() {
    alert('对比功能：显示当前配置与上次保存的差异');
}

function saveConfig() {
    const configFile = document.getElementById('configFileSelect').value;
    const configContent = configEditor ? configEditor.getValue() : '';

    const change = {
        file: configFile + '.yaml',
        content: configContent,
        timestamp: new Date().toISOString()
    };

    pendingChanges.push(change);
    renderPendingChanges();

    showToast('配置已保存，等待热更新生效', 'success');
}

function reloadConfig() {
    const reloadIndicator = document.getElementById('reloadIndicator');
    reloadIndicator.classList.add('active');
    reloadIndicator.querySelector('span').textContent = '热更新中...';

    setTimeout(() => {
        reloadIndicator.classList.remove('active');
        reloadIndicator.querySelector('span').textContent = '热更新完成';

        const statusBadge = document.getElementById('configStatus');
        statusBadge.textContent = '已更新';
        statusBadge.className = 'badge bg-info';

        document.getElementById('lastUpdateTime').textContent = new Date().toLocaleString('zh-CN');

        pendingChanges = [];
        renderPendingChanges();

        showToast('热更新成功！配置已生效', 'success');
    }, 2000);
}

function renderPendingChanges() {
    const container = document.getElementById('pendingChangesList');
    const noChanges = document.getElementById('noPendingChanges');

    if (pendingChanges.length === 0) {
        container.innerHTML = '';
        noChanges.classList.remove('d-none');
        return;
    }

    noChanges.classList.add('d-none');
    container.innerHTML = pendingChanges.map(change => `
        <div class="d-flex justify-content-between align-items-center p-2 bg-light rounded">
            <div>
                <div class="fw-bold">${change.file}</div>
                <div class="text-muted small">${new Date(change.timestamp).toLocaleString('zh-CN')}</div>
            </div>
            <button class="btn btn-sm btn-outline-danger" onclick="removePendingChange('${change.file}')">
                <i class="fas fa-times"></i>
            </button>
        </div>
    `).join('');
}

function removePendingChange(file) {
    pendingChanges = pendingChanges.filter(c => c.file !== file);
    renderPendingChanges();
}

function runHealthCheck() {
    const btn = document.getElementById('runHealthCheckBtn');
    btn.disabled = true;
    btn.innerHTML = '<i class="fas fa-spinner fa-spin me-1"></i>检查中...';

    setTimeout(() => {
        servicesStatus = [
            { name: 'API 服务', status: 'healthy', latency: 12 },
            { name: '数据库', status: 'healthy', latency: 8 },
            { name: 'Redis 缓存', status: 'warning', latency: 45 },
            { name: '负载均衡器', status: 'healthy', latency: 5 },
            { name: '文件存储', status: 'healthy', latency: 15 }
        ];

        renderServicesStatus();
        updateHealthCounts();

        document.getElementById('lastHealthCheck').textContent = new Date().toLocaleTimeString('zh-CN');

        btn.disabled = false;
        btn.innerHTML = '<i class="fas fa-sync me-1"></i>立即检查';

        showToast('健康检查完成', 'success');
    }, 1500);
}

function renderServicesStatus() {
    const container = document.getElementById('servicesList');
    if (!container) return;

    container.innerHTML = servicesStatus.map(service => `
        <div class="service-item ${service.status} p-3 rounded bg-light">
            <div class="d-flex justify-content-between align-items-center">
                <div class="d-flex align-items-center gap-2">
                    <div class="health-indicator ${service.status}"></div>
                    <span class="fw-bold">${service.name}</span>
                </div>
                <div class="d-flex align-items-center gap-3">
                    <span class="text-muted small">延迟: ${service.latency}ms</span>
                    <span class="badge bg-${service.status === 'healthy' ? 'success' : service.status === 'warning' ? 'warning' : 'danger'}">
                        ${service.status === 'healthy' ? '健康' : service.status === 'warning' ? '警告' : '异常'}
                    </span>
                </div>
            </div>
        </div>
    `).join('');
}

function updateHealthCounts() {
    const healthy = servicesStatus.filter(s => s.status === 'healthy').length;
    const warning = servicesStatus.filter(s => s.status === 'warning').length;
    const error = servicesStatus.filter(s => s.status === 'error').length;

    document.getElementById('healthyCount').textContent = healthy;
    document.getElementById('warningCount').textContent = warning;
    document.getElementById('errorCount').textContent = error;
}

function updateSystemMetrics() {
    const cpu = Math.random() * 30 + 40;
    const memory = Math.random() * 20 + 55;
    const disk = Math.random() * 10 + 30;
    const network = Math.random() * 50 + 20;

    document.getElementById('cpuUsagePercent').textContent = cpu.toFixed(1) + '%';
    document.getElementById('cpuUsageBar').style.width = cpu + '%';

    document.getElementById('memoryUsagePercent').textContent = memory.toFixed(1) + '%';
    document.getElementById('memoryUsageBar').style.width = memory + '%';

    document.getElementById('diskUsagePercent').textContent = disk.toFixed(1) + '%';
    document.getElementById('diskUsageBar').style.width = disk + '%';

    document.getElementById('networkUsage').textContent = network.toFixed(0) + ' Mbps';
    document.getElementById('networkUsageBar').style.width = (network / 100 * 100) + '%';

    renderDbConnections();
    renderExternalDeps();
}

function renderDbConnections() {
    const container = document.getElementById('dbConnectionsList');
    if (!container) return;

    const connections = [
        { name: '主库', connections: 45, max: 100, status: 'healthy' },
        { name: '从库1', connections: 23, max: 100, status: 'healthy' },
        { name: '从库2', connections: 18, max: 100, status: 'healthy' }
    ];

    container.innerHTML = connections.map(conn => `
        <div class="mb-3">
            <div class="d-flex justify-content-between mb-1">
                <span>${conn.name}</span>
                <span>${conn.connections}/${conn.max}</span>
            </div>
            <div class="progress progress-thin">
                <div class="progress-bar bg-${conn.status === 'healthy' ? 'success' : 'danger'}" style="width: ${(conn.connections / conn.max) * 100}%"></div>
            </div>
        </div>
    `).join('');
}

function renderExternalDeps() {
    const container = document.getElementById('externalDepsList');
    if (!container) return;

    const deps = [
        { name: '阿里云 OSS', status: 'healthy', latency: 25 },
        { name: '短信网关', status: 'healthy', latency: 120 },
        { name: '邮件服务', status: 'healthy', latency: 80 }
    ];

    container.innerHTML = deps.map(dep => `
        <div class="d-flex justify-content-between align-items-center py-2 border-bottom">
            <div class="d-flex align-items-center gap-2">
                <div class="health-indicator ${dep.status}"></div>
                <span>${dep.name}</span>
            </div>
            <span class="text-muted small">${dep.latency}ms</span>
        </div>
    `).join('');
}

function startHealthCheckPolling() {
    const autoRefresh = document.getElementById('autoRefreshHealth');
    if (autoRefresh && autoRefresh.checked) {
        setInterval(() => {
            updateSystemMetrics();
        }, 10000);
    }
}

function executeBatchAction(action) {
    const actionNames = {
        enable: '启用',
        disable: '禁用',
        delete: '删除',
        export: '导出'
    };

    const messages = {
        enable: '选中的项目将被启用',
        disable: '选中的项目将被禁用',
        delete: '选中的项目将被删除，此操作不可恢复',
        export: '选中的数据将被导出'
    };

    if (action === 'delete') {
        if (!confirm('确定要删除选中的项目吗？此操作不可恢复！')) {
            return;
        }
    } else if (action === 'export') {
        showToast('正在导出数据...', 'info');
        setTimeout(() => {
            showToast('导出成功！文件已准备就绪', 'success');
        }, 2000);
        return;
    }

    showToast(`${actionNames[action]}操作已提交`, 'success');
}

function loadBatchOpsHistory() {
    batchOpsHistory = [
        { time: '2024-01-15 14:00:00', type: '批量启用', count: 15, status: 'success' },
        { time: '2024-01-15 10:30:00', type: '批量禁用', count: 8, status: 'success' },
        { time: '2024-01-14 16:00:00', type: '批量删除', count: 3, status: 'success' },
        { time: '2024-01-14 09:00:00', type: '批量导出', count: 25, status: 'success' }
    ];
    renderBatchOpsHistory();
}

function renderBatchOpsHistory() {
    const tbody = document.getElementById('batchOpsHistoryTable');
    if (!tbody) return;

    tbody.innerHTML = batchOpsHistory.map(item => `
        <tr>
            <td><small>${item.time}</small></td>
            <td>${item.type}</td>
            <td>${item.count}</td>
            <td><span class="badge bg-${item.status === 'success' ? 'success' : 'danger'}">${item.status === 'success' ? '成功' : '失败'}</span></td>
        </tr>
    `).join('');
}

function loadScheduledTasks() {
    scheduledTasks = [
        { name: '每日数据统计', schedule: '0 2 * * *', nextRun: '明天 02:00', status: 'active' },
        { name: '日志清理', schedule: '0 0 * * 0', nextRun: '周日 00:00', status: 'active' },
        { name: '配置备份', schedule: '0 */6 * * *', nextRun: '6小时后', status: 'active' }
    ];
    renderScheduledTasks();
}

function renderScheduledTasks() {
    const container = document.getElementById('scheduledTasksList');
    if (!container) return;

    container.innerHTML = scheduledTasks.map(task => `
        <div class="d-flex justify-content-between align-items-center p-2 bg-light rounded">
            <div>
                <div class="fw-bold">${task.name}</div>
                <div class="text-muted small">${task.schedule} | 下次: ${task.nextRun}</div>
            </div>
            <div class="form-check form-switch">
                <input class="form-check-input" type="checkbox" ${task.status === 'active' ? 'checked' : ''}>
            </div>
        </div>
    `).join('');
}

function loadEnvVars() {
    envVars = [
        { name: 'API_SECRET_KEY', value: 'sk_live_xxxxxxxxxxxxx', type: 'secret', env: 'production', description: 'API密钥' },
        { name: 'DATABASE_URL', value: 'mysql://user:pass@host:3306/db', type: 'secret', env: 'all', description: '数据库连接' },
        { name: 'REDIS_HOST', value: '127.0.0.1', type: 'string', env: 'all', description: 'Redis主机' },
        { name: 'REDIS_PORT', value: '6379', type: 'number', env: 'all', description: 'Redis端口' },
        { name: 'ENABLE_DEBUG', value: 'false', type: 'boolean', env: 'development', description: '调试模式' },
        { name: 'LOG_LEVEL', value: 'info', type: 'string', env: 'all', description: '日志级别' }
    ];
    renderEnvVars();
    loadEnvVarHistory();
}

function renderEnvVars() {
    const tbody = document.getElementById('envVarsTable');
    if (!tbody) return;

    tbody.innerHTML = envVars.map(envVar => `
        <tr class="env-var-row">
            <td><code>${envVar.name}</code></td>
            <td><code class="${envVar.type === 'secret' ? 'text-danger' : ''}">${maskValue(envVar.value, envVar.type)}</code></td>
            <td><span class="badge bg-${getTypeBadgeColor(envVar.type)}">${getTypeName(envVar.type)}</span></td>
            <td>${getEnvName(envVar.env)}</td>
            <td>${envVar.description}</td>
            <td>
                <button class="btn btn-sm btn-outline-secondary me-1" onclick="editEnvVar('${envVar.name}')"><i class="fas fa-edit"></i></button>
                <button class="btn btn-sm btn-outline-danger" onclick="deleteEnvVar('${envVar.name}')"><i class="fas fa-trash"></i></button>
            </td>
        </tr>
    `).join('');
}

function maskValue(value, type) {
    if (type === 'secret') {
        return value.substring(0, 8) + '****' + value.substring(value.length - 4);
    }
    return value;
}

function getTypeBadgeColor(type) {
    const colors = { secret: 'danger', string: 'secondary', number: 'info', boolean: 'success' };
    return colors[type] || 'secondary';
}

function getTypeName(type) {
    const names = { secret: '密钥', string: '字符串', number: '数字', boolean: '布尔' };
    return names[type] || type;
}

function getEnvName(env) {
    const names = { all: '全部', development: '开发', staging: '测试', production: '生产' };
    return names[env] || env;
}

function filterEnvVars() {
    const search = document.getElementById('searchEnvVar').value.toLowerCase();
    const type = document.getElementById('filterEnvVarType').value;

    const filtered = envVars.filter(envVar => {
        const matchSearch = envVar.name.toLowerCase().includes(search) || envVar.description.toLowerCase().includes(search);
        const matchType = !type || envVar.type === type;
        return matchSearch && matchType;
    });

    const tbody = document.getElementById('envVarsTable');
    tbody.innerHTML = filtered.map(envVar => `
        <tr class="env-var-row">
            <td><code>${envVar.name}</code></td>
            <td><code class="${envVar.type === 'secret' ? 'text-danger' : ''}">${maskValue(envVar.value, envVar.type)}</code></td>
            <td><span class="badge bg-${getTypeBadgeColor(envVar.type)}">${getTypeName(envVar.type)}</span></td>
            <td>${getEnvName(envVar.env)}</td>
            <td>${envVar.description}</td>
            <td>
                <button class="btn btn-sm btn-outline-secondary me-1" onclick="editEnvVar('${envVar.name}')"><i class="fas fa-edit"></i></button>
                <button class="btn btn-sm btn-outline-danger" onclick="deleteEnvVar('${envVar.name}')"><i class="fas fa-trash"></i></button>
            </td>
        </tr>
    `).join('');
}

function toggleEnvVarValues() {
    const show = document.getElementById('showEnvVarValues').checked;
    renderEnvVars();
}

function showAddEnvVarModal() {
    document.getElementById('envVarModalTitle').textContent = '添加环境变量';
    document.getElementById('envVarName').value = '';
    document.getElementById('envVarValue').value = '';
    document.getElementById('envVarType').value = 'string';
    document.getElementById('envVarEnv').value = 'all';
    document.getElementById('envVarDescription').value = '';

    const modal = new bootstrap.Modal(document.getElementById('envVarModal'));
    modal.show();
}

function editEnvVar(name) {
    const envVar = envVars.find(e => e.name === name);
    if (!envVar) return;

    document.getElementById('envVarModalTitle').textContent = '编辑环境变量';
    document.getElementById('envVarName').value = envVar.name;
    document.getElementById('envVarName').disabled = true;
    document.getElementById('envVarValue').value = envVar.value;
    document.getElementById('envVarType').value = envVar.type;
    document.getElementById('envVarEnv').value = envVar.env;
    document.getElementById('envVarDescription').value = envVar.description;

    const modal = new bootstrap.Modal(document.getElementById('envVarModal'));
    modal.show();
}

function saveEnvVar() {
    const name = document.getElementById('envVarName').value;
    const value = document.getElementById('envVarValue').value;
    const type = document.getElementById('envVarType').value;
    const env = document.getElementById('envVarEnv').value;
    const description = document.getElementById('envVarDescription').value;

    if (!name || !value) {
        showToast('请填写完整的变量信息', 'warning');
        return;
    }

    const existing = envVars.find(e => e.name === name);
    if (existing) {
        existing.value = value;
        existing.type = type;
        existing.env = env;
        existing.description = description;
    } else {
        envVars.push({ name, value, type, env, description });
    }

    renderEnvVars();

    const historyEntry = {
        time: new Date().toLocaleString('zh-CN'),
        name: name,
        action: existing ? '修改' : '添加',
        operator: 'admin'
    };
    envVarHistory.unshift(historyEntry);
    renderEnvVarHistory();

    bootstrap.Modal.getInstance(document.getElementById('envVarModal')).hide();
    showToast('环境变量保存成功', 'success');
}

function deleteEnvVar(name) {
    if (!confirm(`确定要删除环境变量 ${name} 吗？`)) {
        return;
    }

    envVars = envVars.filter(e => e.name !== name);
    renderEnvVars();

    const historyEntry = {
        time: new Date().toLocaleString('zh-CN'),
        name: name,
        action: '删除',
        operator: 'admin'
    };
    envVarHistory.unshift(historyEntry);
    renderEnvVarHistory();

    showToast('环境变量已删除', 'success');
}

function loadEnvVarHistory() {
    envVarHistory = [
        { time: '2024-01-15 14:30:00', name: 'LOG_LEVEL', action: '修改', operator: 'admin' },
        { time: '2024-01-14 10:00:00', name: 'API_SECRET_KEY', action: '添加', operator: 'admin' }
    ];
    renderEnvVarHistory();
}

function renderEnvVarHistory() {
    const tbody = document.getElementById('envVarHistoryTable');
    if (!tbody) return;

    tbody.innerHTML = envVarHistory.map(item => `
        <tr>
            <td><small>${item.time}</small></td>
            <td><code>${item.name}</code></td>
            <td><span class="badge bg-${item.action === '删除' ? 'danger' : 'primary'}">${item.action}</span></td>
            <td>${item.operator}</td>
        </tr>
    `).join('');
}

function loadBackups() {
    backups = [
        { name: 'backup_20240115_143000', type: '全量', size: '2.5 GB', time: '2024-01-15 14:30:00', status: 'success' },
        { name: 'backup_20240115_083000', type: '增量', size: '520 MB', time: '2024-01-15 08:30:00', status: 'success' },
        { name: 'backup_20240114_143000', type: '全量', size: '2.4 GB', time: '2024-01-14 14:30:00', status: 'success' },
        { name: 'backup_20240114_083000', type: '增量', size: '480 MB', time: '2024-01-14 08:30:00', status: 'success' }
    ];

    document.getElementById('totalBackups').textContent = backups.length;
    document.getElementById('lastBackupTime').textContent = backups[0]?.time.split(' ')[1] || '--';
    document.getElementById('backupStorageSize').textContent = calculateStorageSize();

    renderBackups();
}

function calculateStorageSize() {
    let total = 0;
    backups.forEach(backup => {
        const size = backup.size;
        if (size.includes('GB')) {
            total += parseFloat(size);
        } else if (size.includes('MB')) {
            total += parseFloat(size) / 1024;
        }
    });
    return total.toFixed(2) + ' GB';
}

function renderBackups() {
    const tbody = document.getElementById('backupsTable');
    if (!tbody) return;

    tbody.innerHTML = backups.map(backup => `
        <tr>
            <td><code>${backup.name}</code></td>
            <td><span class="badge bg-${backup.type === '全量' ? 'primary' : 'info'}">${backup.type}</span></td>
            <td>${backup.size}</td>
            <td><small>${backup.time}</small></td>
            <td><span class="badge bg-${backup.status === 'success' ? 'success' : 'danger'}">${backup.status === 'success' ? '完成' : '失败'}</span></td>
            <td>
                <button class="btn btn-sm btn-outline-secondary me-1" onclick="restoreBackup('${backup.name}')"><i class="fas fa-undo"></i></button>
                <button class="btn btn-sm btn-outline-danger" onclick="deleteBackup('${backup.name}')"><i class="fas fa-trash"></i></button>
            </td>
        </tr>
    `).join('');
}

function createBackup() {
    const btn = document.getElementById('createBackupBtn');
    btn.disabled = true;
    btn.innerHTML = '<i class="fas fa-spinner fa-spin me-1"></i>备份中...';

    setTimeout(() => {
        const now = new Date();
        const backupName = `backup_${now.getFullYear()}${String(now.getMonth() + 1).padStart(2, '0')}${String(now.getDate()).padStart(2, '0')}_${String(now.getHours()).padStart(2, '0')}${String(now.getMinutes()).padStart(2, '0')}${String(now.getSeconds()).padStart(2, '0')}`;

        backups.unshift({
            name: backupName,
            type: '手动',
            size: '1.2 GB',
            time: now.toLocaleString('zh-CN'),
            status: 'success'
        });

        loadBackups();

        btn.disabled = false;
        btn.innerHTML = '<i class="fas fa-plus me-1"></i>立即备份';

        showToast('备份创建成功', 'success');
    }, 3000);
}

function restoreBackup(name) {
    if (!confirm(`确定要恢复备份 ${name} 吗？当前数据将被覆盖。`)) {
        return;
    }

    showToast('正在恢复备份...', 'info');

    setTimeout(() => {
        showToast('备份恢复成功', 'success');
    }, 3000);
}

function deleteBackup(name) {
    if (!confirm(`确定要删除备份 ${name} 吗？此操作不可恢复。`)) {
        return;
    }

    backups = backups.filter(b => b.name !== name);
    loadBackups();

    showToast('备份已删除', 'success');
}

function showToast(message, type) {
    const container = document.getElementById('toastContainer');
    const toast = document.createElement('div');
    toast.className = `toast align-items-center text-white bg-${type === 'success' ? 'success' : type === 'danger' ? 'danger' : type === 'warning' ? 'warning' : 'info'} border-0`;
    toast.setAttribute('role', 'alert');
    toast.innerHTML = `
        <div class="d-flex">
            <div class="toast-body">${message}</div>
            <button type="button" class="btn-close btn-close-white me-2 m-auto" data-bs-dismiss="toast"></button>
        </div>
    `;
    container.appendChild(toast);

    const bsToast = new bootstrap.Toast(toast);
    bsToast.show();

    toast.addEventListener('hidden.bs.toast', () => {
        toast.remove();
    });
}
