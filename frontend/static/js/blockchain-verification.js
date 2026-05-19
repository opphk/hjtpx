(function(window) {
    'use strict';

    const BlockchainVerification = {
        API_BASE: '/api/blockchain',

        async recordVerification(data) {
            const payload = {
                app_id: data.appId || 'default',
                session_id: data.sessionId,
                event_type: data.eventType || 'verification',
                event_data: JSON.stringify(data.eventData || {}),
                risk_level: data.riskLevel || 'low',
                risk_score: data.riskScore || 0.5,
                user_agent: navigator.userAgent,
                ip_address: data.ipAddress || '',
                device_fingerprint: data.deviceFingerprint || ''
            };

            const response = await fetch(`${this.API_BASE}/record`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(payload)
            });

            if (!response.ok) {
                throw new Error('Failed to record verification on blockchain');
            }

            return response.json();
        },

        async verifyProof(proofId) {
            const response = await fetch(`${this.API_BASE}/verify/${proofId}`);

            if (!response.ok) {
                throw new Error('Failed to verify proof');
            }

            return response.json();
        },

        async getVerificationHistory(appId, limit = 50, offset = 0) {
            const params = new URLSearchParams({
                app_id: appId,
                limit: limit,
                offset: offset
            });

            const response = await fetch(`${this.API_BASE}/history?${params}`);

            if (!response.ok) {
                throw new Error('Failed to fetch verification history');
            }

            return response.json();
        },

        async registerCrossChainIdentity(identityData) {
            const payload = {
                identity: identityData.identity,
                chain_type: identityData.chainType,
                public_key: identityData.publicKey || '',
                linked_ids: identityData.linkedIds || []
            };

            const response = await fetch(`${this.API_BASE}/identity/register`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(payload)
            });

            if (!response.ok) {
                throw new Error('Failed to register cross-chain identity');
            }

            return response.json();
        },

        async verifyCrossChain(request) {
            const payload = {
                source_chain: request.sourceChain,
                target_chain: request.targetChain,
                identity: request.identity,
                verification_type: request.verificationType || 'basic',
                proof: request.proof || ''
            };

            const response = await fetch(`${this.API_BASE}/identity/verify`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(payload)
            });

            if (!response.ok) {
                throw new Error('Failed to verify cross-chain identity');
            }

            return response.json();
        },

        async createAuditLog(logData) {
            const payload = {
                app_id: logData.appId,
                user_id: logData.userId,
                action: logData.action,
                resource: logData.resource || '',
                details: logData.details || '',
                ip_address: logData.ipAddress || '',
                user_agent: navigator.userAgent,
                result: logData.result || 'unknown'
            };

            const response = await fetch(`${this.API_BASE}/audit/log`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(payload)
            });

            if (!response.ok) {
                throw new Error('Failed to create audit log');
            }

            return response.json();
        },

        async getAuditLogs(appId, startDate, endDate) {
            const params = new URLSearchParams({
                app_id: appId
            });

            if (startDate) {
                params.append('start', startDate.toISOString());
            }
            if (endDate) {
                params.append('end', endDate.toISOString());
            }

            const response = await fetch(`${this.API_BASE}/audit/logs?${params}`);

            if (!response.ok) {
                throw new Error('Failed to fetch audit logs');
            }

            return response.json();
        },

        async exportAuditTrail(appId) {
            const response = await fetch(`${this.API_BASE}/audit/export?app_id=${appId}`);

            if (!response.ok) {
                throw new Error('Failed to export audit trail');
            }

            return response.blob();
        },

        generateProofQRCode(proof) {
            const qrData = JSON.stringify({
                proof_id: proof.proof_id,
                tx_hash: proof.tx_hash,
                block_number: proof.block_number,
                timestamp: proof.timestamp
            });

            return this.generateQRCode(qrData);
        },

        generateQRCode(data) {
            const canvas = document.createElement('canvas');
            const ctx = canvas.getContext('2d');
            const size = 200;
            canvas.width = size;
            canvas.height = size;

            ctx.fillStyle = 'white';
            ctx.fillRect(0, 0, size, size);

            ctx.fillStyle = 'black';
            const gridSize = 5;
            const cellSize = size / (gridSize * 8 + 2);

            let dataIndex = 0;
            const pattern = data.split('').map(c => c.charCodeAt(0) % 2);

            for (let y = 0; y < gridSize * 8; y++) {
                for (let x = 0; x < gridSize * 8; x++) {
                    const idx = (y * gridSize * 8 + x) % pattern.length;
                    if (pattern[idx] === 1) {
                        ctx.fillRect(
                            cellSize + x * cellSize,
                            cellSize + y * cellSize,
                            cellSize,
                            cellSize
                        );
                    }
                }
            }

            ctx.strokeStyle = 'black';
            ctx.lineWidth = 2;
            ctx.strokeRect(cellSize, cellSize, gridSize * 8 * cellSize, gridSize * 8 * cellSize);

            return canvas.toDataURL('image/png');
        },

        displayVerificationResult(proof, container) {
            const resultHTML = `
                <div class="blockchain-verification-result">
                    <div class="verification-header">
                        <h3>Blockchain Verification Complete</h3>
                        <span class="status-badge ${proof.status}">${proof.status}</span>
                    </div>
                    <div class="verification-details">
                        <div class="detail-row">
                            <label>Proof ID:</label>
                            <span class="hash-value">${proof.proof_id}</span>
                        </div>
                        <div class="detail-row">
                            <label>Transaction Hash:</label>
                            <span class="hash-value">${proof.tx_hash}</span>
                        </div>
                        <div class="detail-row">
                            <label>Block Number:</label>
                            <span>${proof.block_number}</span>
                        </div>
                        <div class="detail-row">
                            <label>Chain:</label>
                            <span>${proof.chain_id}</span>
                        </div>
                        <div class="detail-row">
                            <label>Timestamp:</label>
                            <span>${new Date(proof.timestamp).toLocaleString()}</span>
                        </div>
                        <div class="detail-row">
                            <label>Confirmations:</label>
                            <span>${proof.confirmations}</span>
                        </div>
                    </div>
                    <div class="verification-actions">
                        <button class="btn-copy" data-copy="${proof.tx_hash}">Copy TX Hash</button>
                        <button class="btn-view" data-hash="${proof.tx_hash}">View on Explorer</button>
                    </div>
                </div>
            `;

            if (container) {
                container.innerHTML = resultHTML;
                this.attachResultActions(container);
            }

            return resultHTML;
        },

        attachResultActions(container) {
            const copyBtn = container.querySelector('.btn-copy');
            if (copyBtn) {
                copyBtn.addEventListener('click', () => {
                    const hash = copyBtn.getAttribute('data-copy');
                    navigator.clipboard.writeText(hash).then(() => {
                        alert('Copied to clipboard!');
                    });
                });
            }

            const viewBtn = container.querySelector('.btn-view');
            if (viewBtn) {
                viewBtn.addEventListener('click', () => {
                    const hash = viewBtn.getAttribute('data-hash');
                    const explorerUrl = `https://etherscan.io/tx/${hash}`;
                    window.open(explorerUrl, '_blank');
                });
            }
        },

        displayCrossChainStatus(identity, container) {
            const statusHTML = `
                <div class="crosschain-status">
                    <div class="identity-info">
                        <h3>Cross-Chain Identity</h3>
                        <div class="identity-address">${identity.identity}</div>
                    </div>
                    <div class="trust-score-section">
                        <label>Trust Score</label>
                        <div class="score-bar">
                            <div class="score-fill" style="width: ${identity.trust_score}%"></div>
                        </div>
                        <span class="score-value">${identity.trust_score.toFixed(1)}%</span>
                    </div>
                    <div class="identity-details">
                        <div class="detail-row">
                            <label>Chain Type:</label>
                            <span>${identity.chain_type}</span>
                        </div>
                        <div class="detail-row">
                            <label>Status:</label>
                            <span class="status-badge ${identity.status}">${identity.status}</span>
                        </div>
                        <div class="detail-row">
                            <label>Verified:</label>
                            <span>${identity.verified ? 'Yes' : 'No'}</span>
                        </div>
                        ${identity.linked_ids.length > 0 ? `
                        <div class="detail-row">
                            <label>Linked IDs:</label>
                            <ul class="linked-list">
                                ${identity.linked_ids.map(id => `<li>${id}</li>`).join('')}
                            </ul>
                        </div>
                        ` : ''}
                    </div>
                </div>
            `;

            if (container) {
                container.innerHTML = statusHTML;
            }

            return statusHTML;
        },

        initBlockExplorer() {
            const explorerContainer = document.getElementById('blockchain-explorer');
            if (!explorerContainer) return;

            this.renderExplorerUI(explorerContainer);
        },

        renderExplorerUI(container) {
            container.innerHTML = `
                <div class="blockchain-explorer-panel">
                    <div class="panel-header">
                        <h3>Blockchain Verification Explorer</h3>
                    </div>
                    <div class="panel-tabs">
                        <button class="tab-btn active" data-tab="search">Search</button>
                        <button class="tab-btn" data-tab="history">History</button>
                        <button class="tab-btn" data-tab="audit">Audit Logs</button>
                    </div>
                    <div class="panel-content">
                        <div class="tab-content active" id="tab-search">
                            <div class="search-form">
                                <input type="text" id="search-proof-id" placeholder="Enter Proof ID or TX Hash">
                                <button id="btn-search-proof">Search</button>
                            </div>
                            <div id="search-result" class="result-container"></div>
                        </div>
                        <div class="tab-content" id="tab-history">
                            <div class="history-controls">
                                <input type="text" id="history-app-id" placeholder="App ID">
                                <button id="btn-fetch-history">Fetch History</button>
                            </div>
                            <div id="history-result" class="result-container"></div>
                        </div>
                        <div class="tab-content" id="tab-audit">
                            <div class="audit-controls">
                                <input type="text" id="audit-app-id" placeholder="App ID">
                                <button id="btn-fetch-audit">Fetch Audit Logs</button>
                                <button id="btn-export-audit">Export</button>
                            </div>
                            <div id="audit-result" class="result-container"></div>
                        </div>
                    </div>
                </div>
            `;

            this.attachExplorerEvents();
        },

        attachExplorerEvents() {
            document.querySelectorAll('.tab-btn').forEach(btn => {
                btn.addEventListener('click', (e) => {
                    const tab = e.target.getAttribute('data-tab');
                    document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
                    document.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));
                    e.target.classList.add('active');
                    document.getElementById(`tab-${tab}`).classList.add('active');
                });
            });

            const searchBtn = document.getElementById('btn-search-proof');
            if (searchBtn) {
                searchBtn.addEventListener('click', async () => {
                    const proofId = document.getElementById('search-proof-id').value;
                    if (!proofId) return;

                    try {
                        const proof = await this.verifyProof(proofId);
                        this.displayVerificationResult(proof, document.getElementById('search-result'));
                    } catch (err) {
                        document.getElementById('search-result').innerHTML = `<div class="error">${err.message}</div>`;
                    }
                });
            }

            const historyBtn = document.getElementById('btn-fetch-history');
            if (historyBtn) {
                historyBtn.addEventListener('click', async () => {
                    const appId = document.getElementById('history-app-id').value;
                    if (!appId) return;

                    try {
                        const history = await this.getVerificationHistory(appId);
                        this.displayHistory(history, document.getElementById('history-result'));
                    } catch (err) {
                        document.getElementById('history-result').innerHTML = `<div class="error">${err.message}</div>`;
                    }
                });
            }

            const auditBtn = document.getElementById('btn-fetch-audit');
            if (auditBtn) {
                auditBtn.addEventListener('click', async () => {
                    const appId = document.getElementById('audit-app-id').value;
                    if (!appId) return;

                    try {
                        const logs = await this.getAuditLogs(appId);
                        this.displayAuditLogs(logs, document.getElementById('audit-result'));
                    } catch (err) {
                        document.getElementById('audit-result').innerHTML = `<div class="error">${err.message}</div>`;
                    }
                });
            }

            const exportBtn = document.getElementById('btn-export-audit');
            if (exportBtn) {
                exportBtn.addEventListener('click', async () => {
                    const appId = document.getElementById('audit-app-id').value;
                    if (!appId) return;

                    try {
                        const blob = await this.exportAuditTrail(appId);
                        const url = URL.createObjectURL(blob);
                        const a = document.createElement('a');
                        a.href = url;
                        a.download = `audit-trail-${appId}-${Date.now()}.json`;
                        a.click();
                        URL.revokeObjectURL(url);
                    } catch (err) {
                        alert('Export failed: ' + err.message);
                    }
                });
            }
        },

        displayHistory(history, container) {
            if (!history || history.length === 0) {
                container.innerHTML = '<div class="no-data">No verification history found</div>';
                return;
            }

            let html = '<div class="history-list">';
            history.forEach(record => {
                html += `
                    <div class="history-item">
                        <div class="item-header">
                            <span class="record-id">${record.record_id}</span>
                            <span class="risk-badge ${record.risk_level}">${record.risk_level}</span>
                        </div>
                        <div class="item-details">
                            <div>Event: ${record.event_type}</div>
                            <div>Session: ${record.session_id}</div>
                            <div>Score: ${record.risk_score}</div>
                            <div>TX: ${record.chain_tx_hash}</div>
                        </div>
                    </div>
                `;
            });
            html += '</div>';

            container.innerHTML = html;
        },

        displayAuditLogs(logs, container) {
            if (!logs || logs.length === 0) {
                container.innerHTML = '<div class="no-data">No audit logs found</div>';
                return;
            }

            let html = '<div class="audit-list"><table>';
            html += '<thead><tr><th>Time</th><th>Action</th><th>User</th><th>Result</th><th>Hash</th></tr></thead><tbody>';

            logs.forEach(log => {
                html += `
                    <tr>
                        <td>${new Date(log.timestamp).toLocaleString()}</td>
                        <td>${log.action}</td>
                        <td>${log.user_id}</td>
                        <td class="result-${log.result}">${log.result}</td>
                        <td class="hash-cell">${log.hash}</td>
                    </tr>
                `;
            });

            html += '</tbody></table></div>';
            container.innerHTML = html;
        }
    };

    window.BlockchainVerification = BlockchainVerification;

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = BlockchainVerification;
    }

})(typeof window !== 'undefined' ? window : global);
