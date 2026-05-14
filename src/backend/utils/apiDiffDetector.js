const fs = require('fs').promises;
const path = require('path');
const crypto = require('crypto');

class APIDiffDetector {
  constructor(options = {}) {
    this.docsDir = options.docsDir || path.join(__dirname, '../../docs/api');
    this.versionHistoryFile = options.versionHistoryFile || path.join(this.docsDir, 'version-history.json');
    this.currentSpecFile = options.currentSpecFile || path.join(this.docsDir, 'current-spec.json');
  }

  async initialize() {
    try {
      await fs.mkdir(this.docsDir, { recursive: true });
    } catch (error) {
      console.error('Failed to initialize docs directory:', error);
    }
  }

  async saveCurrentSpec(swaggerSpec) {
    const specWithHash = {
      ...swaggerSpec,
      _metadata: {
        savedAt: new Date().toISOString(),
        version: swaggerSpec.info?.version || '1.0.0',
        specHash: this.generateSpecHash(swaggerSpec)
      }
    };

    await fs.writeFile(
      this.currentSpecFile,
      JSON.stringify(specWithHash, null, 2),
      'utf8'
    );

    return specWithHash._metadata.specHash;
  }

  generateSpecHash(spec) {
    const specString = JSON.stringify(spec, Object.keys(spec).sort());
    return crypto.createHash('sha256').update(specString).digest('hex');
  }

  async loadPreviousSpec() {
    try {
      const data = await fs.readFile(this.currentSpecFile, 'utf8');
      return JSON.parse(data);
    } catch (error) {
      return null;
    }
  }

  async detectChanges(currentSpec) {
    const previousSpec = await this.loadPreviousSpec();
    
    if (!previousSpec) {
      return {
        hasChanges: true,
        isBreaking: true,
        changes: ['Initial API documentation'],
        details: {
          added: this.extractEndpoints(currentSpec),
          removed: [],
          modified: []
        }
      };
    }

    const currentEndpoints = this.extractEndpoints(currentSpec);
    const previousEndpoints = this.extractEndpoints(previousSpec);

    const added = currentEndpoints.filter(e => 
      !previousEndpoints.some(p => p.path === e.path && p.method === e.method)
    );

    const removed = previousEndpoints.filter(e =>
      !currentEndpoints.some(c => c.path === e.path && c.method === e.method)
    );

    const modified = currentEndpoints.filter(e => {
      const prev = previousEndpoints.find(p => p.path === e.path && p.method === e.method);
      if (!prev) return false;
      
      const currentSchema = JSON.stringify(this.normalizeSchema(e));
      const prevSchema = JSON.stringify(this.normalizeSchema(prev));
      
      return currentSchema !== prevSchema;
    });

    const isBreaking = this.detectBreakingChanges(added, removed, modified, currentSpec, previousSpec);

    const changes = [];
    if (added.length > 0) changes.push(`Added ${added.length} new endpoints`);
    if (removed.length > 0) changes.push(`Removed ${removed.length} endpoints`);
    if (modified.length > 0) changes.push(`Modified ${modified.length} endpoints`);

    return {
      hasChanges: changes.length > 0,
      isBreaking,
      changes,
      details: {
        added,
        removed,
        modified,
        currentHash: this.generateSpecHash(currentSpec),
        previousHash: previousSpec._metadata?.specHash
      }
    };
  }

  normalizeSchema(endpoint) {
    return {
      parameters: endpoint.parameters,
      requestBody: endpoint.requestBody,
      responses: endpoint.responses
    };
  }

  extractEndpoints(spec) {
    const endpoints = [];
    
    if (spec.paths) {
      for (const [path, methods] of Object.entries(spec.paths)) {
        for (const [method, details] of Object.entries(methods)) {
          if (['get', 'post', 'put', 'delete', 'patch', 'options', 'head'].includes(method)) {
            endpoints.push({
              path,
              method: method.toUpperCase(),
              summary: details.summary || '',
              tags: details.tags || [],
              parameters: details.parameters || [],
              requestBody: details.requestBody,
              responses: details.responses
            });
          }
        }
      }
    }

    return endpoints;
  }

  detectBreakingChanges(added, removed, modified, currentSpec, previousSpec) {
    const breakingPatterns = [
      { type: 'removed_endpoint', count: removed.length },
      { type: 'required_param_added', count: 0 },
      { type: 'response_changed', count: 0 }
    ];

    for (const mod of modified) {
      const prev = previousSpec.paths?.[mod.path]?.[mod.method.toLowerCase()];
      if (prev) {
        if (this.hasRequiredParamChanges(mod, prev)) {
          breakingPatterns[1].count++;
        }
        if (this.hasResponseChanges(mod, prev)) {
          breakingPatterns[2].count++;
        }
      }
    }

    return breakingPatterns.some(p => p.count > 0);
  }

  hasRequiredParamChanges(current, previous) {
    const currentRequired = (current.parameters || [])
      .filter(p => p.required === true);
    const prevRequired = (previous.parameters || [])
      .filter(p => p.required === true);

    return currentRequired.length !== prevRequired.length ||
           !currentRequired.every(p => prevRequired.some(pr => pr.name === p.name));
  }

  hasResponseChanges(current, previous) {
    const currentCodes = Object.keys(current.responses || {});
    const prevCodes = Object.keys(previous.responses || {});

    return !currentCodes.every(c => prevCodes.includes(c));
  }

  async recordVersion(swaggerSpec, changes) {
    const versionHistory = await this.loadVersionHistory();
    
    const newVersion = {
      version: swaggerSpec.info?.version || '1.0.0',
      timestamp: new Date().toISOString(),
      changes: changes,
      specHash: this.generateSpecHash(swaggerSpec),
      breaking: changes.isBreaking
    };

    versionHistory.versions.unshift(newVersion);
    versionHistory.current = newVersion;

    await fs.writeFile(
      this.versionHistoryFile,
      JSON.stringify(versionHistory, null, 2),
      'utf8'
    );

    return newVersion;
  }

  async loadVersionHistory() {
    try {
      const data = await fs.readFile(this.versionHistoryFile, 'utf8');
      return JSON.parse(data);
    } catch (error) {
      return {
        versions: [],
        current: null
      };
    }
  }

  async generateChangeReport(changes) {
    const report = {
      generatedAt: new Date().toISOString(),
      summary: {
        totalChanges: changes.details.added.length + 
                     changes.details.removed.length + 
                     changes.details.modified.length,
        breaking: changes.isBreaking
      },
      details: changes.details,
      recommendations: this.generateRecommendations(changes)
    };

    const reportFile = path.join(this.docsDir, `change-report-${Date.now()}.json`);
    await fs.writeFile(reportFile, JSON.stringify(report, null, 2), 'utf8');

    return report;
  }

  generateRecommendations(changes) {
    const recommendations = [];

    if (changes.details.removed.length > 0) {
      recommendations.push({
        type: 'warning',
        message: 'Removed endpoints detected. Ensure backward compatibility or provide deprecation timeline.'
      });
    }

    if (changes.isBreaking) {
      recommendations.push({
        type: 'critical',
        message: 'Breaking changes detected. Update API versioning strategy.'
      });
    }

    if (changes.details.modified.length > 0) {
      recommendations.push({
        type: 'info',
        message: 'Modified endpoints detected. Consider updating changelog and notifying consumers.'
      });
    }

    return recommendations;
  }
}

module.exports = new APIDiffDetector();
