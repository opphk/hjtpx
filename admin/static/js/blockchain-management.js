document.addEventListener('DOMContentLoaded', function() {
    initBlockchainManagement();
});

function initBlockchainManagement() {
    const container = document.getElementById('blockchain-management');
    if (!container) return;

    renderBlockchainUI(container);
    attachEventHandlers();
}

function renderBlockchainUI(container) {
    container.innerHTML = `
        <div class="blockchain-management-panel">
            <div class="panel-header">
                <h2><i class="icon">⛓️</i> Blockchain Verification Management</h2>
                <div class="header-actions">
                    <button class="btn-refresh" id="btn-refresh-blockchain">
                        <span>↻</span> Refresh
                    </button>
                </div>
            </div>

            <div class="stats-grid">
                <div class="stat-card">
                    <div class="stat-icon">📊</div>
                    <div class="stat-content">
                        <div class="stat-value" id="stat-total-records">0</div>
                        <div class="stat-label">Total Records</div>
                    </div>
                </div>
                <div class="stat-card">
                    <div class="stat-icon">✓</div>
                    <div class="stat-content">
                        <div class="stat-value" id="stat-verified">0</div>
                        <div class="stat-label">Verified</div>
                    </div>
                </div>
                <div class="stat-card">
                    <div class="stat-icon">🔗</div>
                    <div class="stat-content">
                        <div class="stat-value" id="stat-identities">0</div>
                        <div class="stat-label">Cross-Chain Identities</div>
                    </div>
                </div>
                <div class="stat-card">
                    <div class="stat-icon">📝</div>
                    <div class="stat-content">
                        <div class="stat-value" id="stat-audit-logs">0</div>
                        <div class="stat-label">Audit Logs</div>
                    </div>
                </div>
            </div>

            <div class="management-tabs">
                <button class="tab-btn active" data-tab="records">Verification Records</button>
                <button class="tab-btn" data-tab="identities">Cross-Chain Identities</button>
                <button class="tab-btn" data-tab="audit">Audit Logs</button>
                <button class="tab-btn" data-tab="settings">Chain Settings</button>
            </div>

            <div class="tab-content active" id="tab-records">
                <div class="content-header">
                    <div class="filter-group">
                        <input type="text" id="filter-app-id" placeholder="App ID">
                        <select id="filter-risk-level">
                            <option value="">All Risk Levels</option>
                            <option value="low">Low</option>
                            <option value="medium">Medium</option>
                            <option value="high">High</option>
                        </select>
                        <button class="btn-filter" id="btn-filter-records">Filter</button>
                    </div>
                    <div class="action-group">
                        <button class="btn-export" id="btn-export-records">Export Records</button>
                    </div>
                </div>
                <div class="table-container">
                    <table class="data-table" id="records-table">
                        <thead>
                            <tr>
                                <th>Record ID</th>
                                <th>App ID</th>
                                <th>Event Type</th>
                                <th>Risk Score</th>
                                <th>TX Hash</th>
                                <th>Block #</th>
                                <th>Timestamp</th>
                                <th>Actions</th>
                            </tr>
                        </thead>
                        <tbody id="records-tbody">
                            <tr class="loading-row">
                                <td colspan="8">Loading...</td>
                            </tr>
                        </tbody>
                    </table>
                </div>
                <div class="pagination" id="records-pagination"></div>
            </div>

            <div class="tab-content" id="tab-identities">
                <div class="content-header">
                    <button class="btn-primary" id="btn-add-identity">
                        <span>+</span> Add Cross-Chain Identity
                    </button>
                </div>
                <div class="table-container">
                    <table class="data-table" id="identities-table">
                        <thead>
                            <tr>
                                <th>Identity</th>
                                <th>Chain Type</th>
                                <th>Trust Score</th>
                                <th>Status</th>
                                <th>Linked IDs</th>
                                <th>Created</th>
                                <th>Actions</th>
                            </tr>
                        </thead>
                        <tbody id="identities-tbody">
                            <tr class="loading-row">
                                <td colspan="7">Loading...</td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            </div>

            <div class="tab-content" id="tab-audit">
                <div class="content-header">
                    <div class="filter-group">
                        <input type="date" id="audit-start-date" placeholder="Start Date">
                        <input type="date" id="audit-end-date" placeholder="End Date">
                        <input type="text" id="audit-app-filter" placeholder="App ID">
                        <button class="btn-filter" id="btn-filter-audit">Filter</button>
                    </div>
                    <div class="action-group">
                        <button class="btn-export" id="btn-export-audit">Export Audit Trail</button>
                    </div>
                </div>
                <div class="table-container">
                    <table class="data-table" id="audit-table">
                        <thead>
                            <tr>
                                <th>Log ID</th>
                                <th>App ID</th>
                                <th>User ID</th>
                                <th>Action</th>
                                <th>Resource</th>
                                <th>Result</th>
                                <th>Hash</th>
                                <th>Timestamp</th>
                                <th>Verify</th>
                            </tr>
                        </thead>
                        <tbody id="audit-tbody">
                            <tr class="loading-row">
                                <td colspan="9">Loading...</td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            </div>

            <div class="tab-content" id="tab-settings">
                <div class="settings-section">
                    <h3>Supported Chains</h3>
                    <div class="chains-list" id="chains-list">
                        <div class="chain-item" data-chain="ethereum">
                            <div class="chain-info">
                                <span class="chain-name">Ethereum</span>
                                <span class="chain-status active">Active</span>
                            </div>
                            <div class="chain-stats">
                                <span>Latest Block: <strong id="eth-latest-block">-</strong></span>
                                <span>Total Records: <strong id="eth-total-records">-</strong></span>
                            </div>
                        </div>
                        <div class="chain-item" data-chain="polygon">
                            <div class="chain-info">
                                <span class="chain-name">Polygon</span>
                                <span class="chain-status">Inactive</span>
                            </div>
                            <div class="chain-stats">
                                <span>Latest Block: <strong>-</strong></span>
                                <span>Total Records: <strong>-</strong></span>
                            </div>
                        </div>
                        <div class="chain-item" data-chain="bsc">
                            <div class="chain-info">
                                <span class="chain-name">BNB Chain (BSC)</span>
                                <span class="chain-status">Inactive</span>
                            </div>
                            <div class="chain-stats">
                                <span>Latest Block: <strong>-</strong></span>
                                <span>Total Records: <strong>-</strong></span>
                            </div>
                        </div>
                    </div>
                </div>
                <div class="settings-section">
                    <h3>Verification Settings</h3>
                    <div class="setting-item">
                        <label>
                            <input type="checkbox" id="setting-auto-verify" checked>
                            Auto-verify proofs on creation
                        </label>
                    </div>
                    <div class="setting-item">
                        <label>
                            <input type="checkbox" id="setting-auto-audit" checked>
                            Enable automatic audit logging
                        </label>
                    </div>
                    <div class="setting-item">
                        <label>Minimum Trust Score for Cross-Chain:</label>
                        <input type="number" id="setting-min-trust" value="50" min="0" max="100">
                    </div>
                </div>
            </div>
        </div>

        <div class="modal" id="identity-modal">
            <div class="modal-content">
                <div class="modal-header">
                    <h3>Add Cross-Chain Identity</h3>
                    <button class="modal-close" id="modal-close">&times;</button>
                </div>
                <div class="modal-body">
                    <form id="identity-form">
                        <div class="form-group">
                            <label for="identity-address">Identity Address *</label>
                            <input type="text" id="identity-address" required placeholder="0x...">
                        </div>
                        <div class="form-group">
                            <label for="identity-chain">Chain Type *</label>
                            <select id="identity-chain" required>
                                <option value="">Select Chain</option>
                                <option value="ethereum">Ethereum</option>
                                <option value="polygon">Polygon</option>
                                <option value="bsc">BNB Chain (BSC)</option>
                            </select>
                        </div>
                        <div class="form-group">
                            <label for="identity-public-key">Public Key</label>
                            <input type="text" id="identity-public-key" placeholder="0x...">
                        </div>
                        <div class="form-group">
                            <label for="identity-linked">Linked IDs (one per line)</label>
                            <textarea id="identity-linked" rows="3" placeholder="twitter:user1&#10;github:user2"></textarea>
                        </div>
                    </form>
                </div>
                <div class="modal-footer">
                    <button class="btn-cancel" id="modal-cancel">Cancel</button>
                    <button class="btn-primary" id="modal-submit">Submit</button>
                </div>
            </div>
        </div>
    `;
}

function attachEventHandlers() {
    document.querySelectorAll('.tab-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            const tab = e.target.getAttribute('data-tab');
            document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
            document.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));
            e.target.classList.add('active');
            document.getElementById(`tab-${tab}`).classList.add('active');
        });
    });

    const refreshBtn = document.getElementById('btn-refresh-blockchain');
    if (refreshBtn) {
        refreshBtn.addEventListener('click', loadBlockchainStats);
    }

    const addIdentityBtn = document.getElementById('btn-add-identity');
    const modal = document.getElementById('identity-modal');
    const modalClose = document.getElementById('modal-close');
    const modalCancel = document.getElementById('modal-cancel');
    const modalSubmit = document.getElementById('modal-submit');

    if (addIdentityBtn) {
        addIdentityBtn.addEventListener('click', () => {
            modal.style.display = 'flex';
        });
    }

    if (modalClose) {
        modalClose.addEventListener('click', () => {
            modal.style.display = 'none';
        });
    }

    if (modalCancel) {
        modalCancel.addEventListener('click', () => {
            modal.style.display = 'none';
        });
    }

    if (modalSubmit) {
        modalSubmit.addEventListener('click', submitIdentityForm);
    }

    loadBlockchainStats();
    loadVerificationRecords();
    loadCrossChainIdentities();
    loadAuditLogs();
}

async function loadBlockchainStats() {
    document.getElementById('stat-total-records').textContent = '...';
    document.getElementById('stat-verified').textContent = '...';
    document.getElementById('stat-identities').textContent = '...';
    document.getElementById('stat-audit-logs').textContent = '...';

    setTimeout(() => {
        document.getElementById('stat-total-records').textContent = Math.floor(Math.random() * 1000);
        document.getElementById('stat-verified').textContent = Math.floor(Math.random() * 800);
        document.getElementById('stat-identities').textContent = Math.floor(Math.random() * 100);
        document.getElementById('stat-audit-logs').textContent = Math.floor(Math.random() * 500);
    }, 500);
}

async function loadVerificationRecords() {
    const tbody = document.getElementById('records-tbody');
    if (!tbody) return;

    try {
        const mockRecords = generateMockRecords(10);
        renderRecordsTable(mockRecords, tbody);
    } catch (err) {
        tbody.innerHTML = `<tr class="error-row"><td colspan="8">Error: ${err.message}</td></tr>`;
    }
}

function generateMockRecords(count) {
    const records = [];
    const eventTypes = ['login_attempt', 'login_success', 'payment', 'signup', 'password_reset'];
    const riskLevels = ['low', 'medium', 'high'];

    for (let i = 0; i < count; i++) {
        records.push({
            record_id: `rec_${Math.random().toString(36).substr(2, 9)}`,
            app_id: `app_${Math.floor(Math.random() * 5)}`,
            event_type: eventTypes[Math.floor(Math.random() * eventTypes.length)],
            risk_score: (Math.random() * 0.9).toFixed(2),
            chain_tx_hash: '0x' + Math.random().toString(16).substr(2, 64),
            block_number: 15000000 + Math.floor(Math.random() * 100000),
            timestamp: new Date(Date.now() - Math.random() * 7 * 24 * 60 * 60 * 1000).toISOString()
        });
    }
    return records;
}

function renderRecordsTable(records, tbody) {
    if (records.length === 0) {
        tbody.innerHTML = '<tr><td colspan="8" class="no-data">No records found</td></tr>';
        return;
    }

    tbody.innerHTML = records.map(record => `
        <tr data-record-id="${record.record_id}">
            <td><code class="hash">${record.record_id.substring(0, 12)}...</code></td>
            <td>${record.app_id}</td>
            <td><span class="badge">${record.event_type}</span></td>
            <td><span class="score">${record.risk_score}</span></td>
            <td><code class="hash tx-hash">${record.chain_tx_hash.substring(0, 16)}...</code></td>
            <td>${record.block_number.toLocaleString()}</td>
            <td>${new Date(record.timestamp).toLocaleDateString()}</td>
            <td>
                <button class="btn-action btn-view" data-hash="${record.chain_tx_hash}">View</button>
                <button class="btn-action btn-verify" data-id="${record.record_id}">Verify</button>
            </td>
        </tr>
    `).join('');

    tbody.querySelectorAll('.btn-view').forEach(btn => {
        btn.addEventListener('click', () => {
            window.open(`https://etherscan.io/tx/${btn.getAttribute('data-hash')}`, '_blank');
        });
    });

    tbody.querySelectorAll('.btn-verify').forEach(btn => {
        btn.addEventListener('click', () => {
            alert('Verification feature coming soon!');
        });
    });
}

async function loadCrossChainIdentities() {
    const tbody = document.getElementById('identities-tbody');
    if (!tbody) return;

    try {
        const mockIdentities = generateMockIdentities(5);
        renderIdentitiesTable(mockIdentities, tbody);
    } catch (err) {
        tbody.innerHTML = `<tr class="error-row"><td colspan="7">Error: ${err.message}</td></tr>`;
    }
}

function generateMockIdentities(count) {
    const identities = [];
    const chainTypes = ['ethereum', 'polygon', 'bsc'];
    const statuses = ['active', 'pending', 'suspended'];

    for (let i = 0; i < count; i++) {
        const trustScore = Math.floor(Math.random() * 100);
        identities.push({
            identity: '0x' + Math.random().toString(16).substr(2, 40),
            chain_type: chainTypes[Math.floor(Math.random() * chainTypes.length)],
            trust_score: trustScore,
            status: statuses[Math.floor(Math.random() * statuses.length)],
            linked_ids: ['twitter:user' + i, 'github:user' + i],
            created_at: new Date(Date.now() - Math.random() * 365 * 24 * 60 * 60 * 1000).toISOString()
        });
    }
    return identities;
}

function renderIdentitiesTable(identities, tbody) {
    if (identities.length === 0) {
        tbody.innerHTML = '<tr><td colspan="7" class="no-data">No identities found</td></tr>';
        return;
    }

    tbody.innerHTML = identities.map(identity => `
        <tr data-identity="${identity.identity}">
            <td><code class="hash">${identity.identity.substring(0, 16)}...</code></td>
            <td><span class="badge chain">${identity.chain_type}</span></td>
            <td>
                <div class="trust-score">
                    <div class="score-bar"><div class="score-fill" style="width: ${identity.trust_score}%"></div></div>
                    <span>${identity.trust_score}%</span>
                </div>
            </td>
            <td><span class="status-badge ${identity.status}">${identity.status}</span></td>
            <td>${identity.linked_ids.length} linked</td>
            <td>${new Date(identity.created_at).toLocaleDateString()}</td>
            <td>
                <button class="btn-action btn-verify-chain" data-identity="${identity.identity}">Verify</button>
                <button class="btn-action btn-link" data-identity="${identity.identity}">Link</button>
            </td>
        </tr>
    `).join('');
}

async function loadAuditLogs() {
    const tbody = document.getElementById('audit-tbody');
    if (!tbody) return;

    try {
        const mockLogs = generateMockAuditLogs(10);
        renderAuditTable(mockLogs, tbody);
    } catch (err) {
        tbody.innerHTML = `<tr class="error-row"><td colspan="9">Error: ${err.message}</td></tr>`;
    }
}

function generateMockAuditLogs(count) {
    const logs = [];
    const actions = ['login', 'logout', 'view_data', 'update_settings', 'export_data'];
    const results = ['success', 'failure', 'pending'];

    for (let i = 0; i < count; i++) {
        logs.push({
            log_id: `log_${Math.random().toString(36).substr(2, 9)}`,
            app_id: `app_${Math.floor(Math.random() * 5)}`,
            user_id: `user_${Math.floor(Math.random() * 100)}`,
            action: actions[Math.floor(Math.random() * actions.length)],
            resource: `/api/${actions[Math.floor(Math.random() * actions.length)]}`,
            result: results[Math.floor(Math.random() * results.length)],
            hash: Math.random().toString(16).substr(2, 64),
            timestamp: new Date(Date.now() - Math.random() * 7 * 24 * 60 * 60 * 1000).toISOString()
        });
    }
    return logs;
}

function renderAuditTable(logs, tbody) {
    if (logs.length === 0) {
        tbody.innerHTML = '<tr><td colspan="9" class="no-data">No audit logs found</td></tr>';
        return;
    }

    tbody.innerHTML = logs.map(log => `
        <tr data-log-id="${log.log_id}">
            <td><code>${log.log_id.substring(0, 12)}...</code></td>
            <td>${log.app_id}</td>
            <td>${log.user_id}</td>
            <td>${log.action}</td>
            <td><code>${log.resource}</code></td>
            <td><span class="result-badge ${log.result}">${log.result}</span></td>
            <td><code class="hash">${log.hash.substring(0, 12)}...</code></td>
            <td>${new Date(log.timestamp).toLocaleString()}</td>
            <td>
                <button class="btn-action btn-verify-log" data-log-id="${log.log_id}">Verify</button>
            </td>
        </tr>
    `).join('');

    tbody.querySelectorAll('.btn-verify-log').forEach(btn => {
        btn.addEventListener('click', () => {
            alert('Log verification feature coming soon!');
        });
    });
}

async function submitIdentityForm() {
    const address = document.getElementById('identity-address').value;
    const chain = document.getElementById('identity-chain').value;
    const publicKey = document.getElementById('identity-public-key').value;
    const linkedText = document.getElementById('identity-linked').value;

    if (!address || !chain) {
        alert('Please fill in required fields');
        return;
    }

    const linkedIds = linkedText.split('\n').filter(id => id.trim());

    console.log('Submitting identity:', { address, chain, publicKey, linkedIds });

    document.getElementById('identity-modal').style.display = 'none';
    document.getElementById('identity-form').reset();

    alert('Identity registration feature coming soon!');
}

window.BlockchainManagement = {
    loadStats: loadBlockchainStats,
    loadRecords: loadVerificationRecords,
    loadIdentities: loadCrossChainIdentities,
    loadAuditLogs: loadAuditLogs
};
