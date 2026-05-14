const fs = require('fs').promises;
const path = require('path');

class APIVersionManager {
  constructor(options = {}) {
    this.versionsDir = options.versionsDir || path.join(__dirname, '../../docs/api/versions');
    this.currentVersionFile = path.join(this.versionsDir, 'current.json');
    this.versionsIndexFile = path.join(this.versionsDir, 'index.json');
    this.supportedVersions = ['v1', 'v2', 'v3'];
    this.deprecatedVersions = [];
  }

  async initialize() {
    try {
      await fs.mkdir(this.versionsDir, { recursive: true });
      await this.ensureIndexExists();
    } catch (error) {
      console.error('Failed to initialize version manager:', error);
    }
  }

  async ensureIndexExists() {
    try {
      await fs.access(this.versionsIndexFile);
    } catch (error) {
      const initialIndex = {
        versions: [],
        current: null,
        latest: null,
        supported: this.supportedVersions,
        deprecated: this.deprecatedVersions,
        updatedAt: new Date().toISOString()
      };
      await fs.writeFile(
        this.versionsIndexFile,
        JSON.stringify(initialIndex, null, 2),
        'utf8'
      );
    }
  }

  async createVersion(version, swaggerSpec, metadata = {}) {
    const versionDir = path.join(this.versionsDir, version);
    await fs.mkdir(versionDir, { recursive: true });

    const versionData = {
      version,
      spec: swaggerSpec,
      metadata: {
        ...metadata,
        createdAt: new Date().toISOString(),
        createdBy: metadata.createdBy || 'system'
      },
      endpoints: this.extractEndpoints(swaggerSpec),
      schemas: swaggerSpec.components?.schemas || {}
    };

    const specFile = path.join(versionDir, 'openapi.json');
    await fs.writeFile(specFile, JSON.stringify(versionData.spec, null, 2), 'utf8');

    const metadataFile = path.join(versionDir, 'metadata.json');
    await fs.writeFile(metadataFile, JSON.stringify(versionData.metadata, null, 2), 'utf8');

    await this.updateIndex(version, versionData);

    return versionData;
  }

  async updateIndex(version, versionData) {
    const index = await this.loadIndex();
    
    const existingIndex = index.versions.findIndex(v => v.version === version);
    const versionSummary = {
      version,
      createdAt: versionData.metadata.createdAt,
      endpointCount: versionData.endpoints.length,
      schemaCount: Object.keys(versionData.schemas).length
    };

    if (existingIndex >= 0) {
      index.versions[existingIndex] = versionSummary;
    } else {
      index.versions.push(versionSummary);
    }

    index.versions.sort((a, b) => {
      const aNum = parseInt(a.version.replace('v', ''));
      const bNum = parseInt(b.version.replace('v', ''));
      return bNum - aNum;
    });

    index.current = version;
    index.latest = version;
    index.updatedAt = new Date().toISOString();

    await fs.writeFile(
      this.versionsIndexFile,
      JSON.stringify(index, null, 2),
      'utf8'
    );

    return index;
  }

  async loadIndex() {
    try {
      const data = await fs.readFile(this.versionsIndexFile, 'utf8');
      return JSON.parse(data);
    } catch (error) {
      return {
        versions: [],
        current: null,
        latest: null,
        supported: this.supportedVersions,
        deprecated: this.deprecatedVersions,
        updatedAt: new Date().toISOString()
      };
    }
  }

  async getVersion(version) {
    const versionDir = path.join(this.versionsDir, version);
    const specFile = path.join(versionDir, 'openapi.json');
    const metadataFile = path.join(versionDir, 'metadata.json');

    try {
      const [specData, metadataData] = await Promise.all([
        fs.readFile(specFile, 'utf8'),
        fs.readFile(metadataFile, 'utf8')
      ]);

      return {
        version,
        spec: JSON.parse(specData),
        metadata: JSON.parse(metadataData),
        endpoints: this.extractEndpoints(JSON.parse(specData)),
        schemas: JSON.parse(specData).components?.schemas || {}
      };
    } catch (error) {
      return null;
    }
  }

  async getCurrentVersion() {
    const index = await this.loadIndex();
    if (!index.current) return null;
    return this.getVersion(index.current);
  }

  async listVersions() {
    return this.loadIndex();
  }

  async deprecateVersion(version, sunsetDate = null) {
    const versionData = await this.getVersion(version);
    if (!versionData) {
      throw new Error(`Version ${version} not found`);
    }

    versionData.metadata.deprecated = true;
    versionData.metadata.deprecatedAt = new Date().toISOString();
    versionData.metadata.sunsetDate = sunsetDate;

    const metadataFile = path.join(this.versionsDir, version, 'metadata.json');
    await fs.writeFile(metadataFile, JSON.stringify(versionData.metadata, null, 2), 'utf8');

    const index = await this.loadIndex();
    const versionIndex = index.versions.findIndex(v => v.version === version);
    if (versionIndex >= 0) {
      index.versions[versionIndex].deprecated = true;
      index.versions[versionIndex].sunsetDate = sunsetDate;
    }

    if (!index.deprecated.includes(version)) {
      index.deprecated.push(version);
    }

    index.deprecated = [...new Set(index.deprecated)];
    index.updatedAt = new Date().toISOString();

    await fs.writeFile(
      this.versionsIndexFile,
      JSON.stringify(index, null, 2),
      'utf8'
    );

    return versionData.metadata;
  }

  async compareVersions(version1, version2) {
    const [v1Data, v2Data] = await Promise.all([
      this.getVersion(version1),
      this.getVersion(version2)
    ]);

    if (!v1Data || !v2Data) {
      throw new Error('One or both versions not found');
    }

    const comparison = {
      versions: {
        from: version1,
        to: version2
      },
      added: {
        endpoints: [],
        schemas: []
      },
      removed: {
        endpoints: [],
        schemas: []
      },
      modified: {
        endpoints: [],
        schemas: []
      }
    };

    const v1Endpoints = new Map(v1Data.endpoints.map(e => [`${e.method} ${e.path}`, e]));
    const v2Endpoints = new Map(v2Data.endpoints.map(e => [`${e.method} ${e.path}`, e]));

    for (const [key, endpoint] of v2Endpoints) {
      if (!v1Endpoints.has(key)) {
        comparison.added.endpoints.push(endpoint);
      } else {
        const v1Endpoint = v1Endpoints.get(key);
        if (JSON.stringify(v1Endpoint) !== JSON.stringify(endpoint)) {
          comparison.modified.endpoints.push({
            from: v1Endpoint,
            to: endpoint
          });
        }
      }
    }

    for (const key of v1Endpoints.keys()) {
      if (!v2Endpoints.has(key)) {
        comparison.removed.endpoints.push(v1Endpoints.get(key));
      }
    }

    const v1Schemas = new Set(Object.keys(v1Data.schemas));
    const v2Schemas = new Set(Object.keys(v2Data.schemas));

    for (const schema of v2Schemas) {
      if (!v1Schemas.has(schema)) {
        comparison.added.schemas.push(schema);
      }
    }

    for (const schema of v1Schemas) {
      if (!v2Schemas.has(schema)) {
        comparison.removed.schemas.push(schema);
      }
    }

    return comparison;
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
              description: details.description || '',
              tags: details.tags || [],
              deprecated: details.deprecated || false,
              parameters: details.parameters || [],
              responses: Object.keys(details.responses || {})
            });
          }
        }
      }
    }

    return endpoints;
  }

  async generateVersionReport(version) {
    const versionData = await this.getVersion(version);
    if (!versionData) {
      throw new Error(`Version ${version} not found`);
    }

    const report = {
      version,
      summary: {
        totalEndpoints: versionData.endpoints.length,
        totalSchemas: Object.keys(versionData.schemas).length,
        deprecated: versionData.metadata.deprecated || false,
        createdAt: versionData.metadata.createdAt
      },
      endpoints: {
        byMethod: this.groupEndpointsByMethod(versionData.endpoints),
        byTag: this.groupEndpointsByTag(versionData.endpoints),
        deprecated: versionData.endpoints.filter(e => e.deprecated)
      },
      schemas: {
        list: Object.keys(versionData.schemas),
        used: this.findUsedSchemas(versionData)
      }
    };

    const reportFile = path.join(this.versionsDir, version, 'report.json');
    await fs.writeFile(reportFile, JSON.stringify(report, null, 2), 'utf8');

    return report;
  }

  groupEndpointsByMethod(endpoints) {
    const grouped = {};
    for (const endpoint of endpoints) {
      if (!grouped[endpoint.method]) {
        grouped[endpoint.method] = [];
      }
      grouped[endpoint.method].push(endpoint);
    }
    return grouped;
  }

  groupEndpointsByTag(endpoints) {
    const grouped = {};
    for (const endpoint of endpoints) {
      const tag = endpoint.tags[0] || 'uncategorized';
      if (!grouped[tag]) {
        grouped[tag] = [];
      }
      grouped[tag].push(endpoint);
    }
    return grouped;
  }

  findUsedSchemas(spec) {
    const usedSchemas = new Set();
    
    const searchInSchema = (schema) => {
      if (!schema) return;
      
      if (schema.$ref) {
        const ref = schema.$ref.split('/').pop();
        usedSchemas.add(ref);
      }
      
      if (schema.allOf) schema.allOf.forEach(searchInSchema);
      if (schema.anyOf) schema.anyOf.forEach(searchInSchema);
      if (schema.oneOf) schema.oneOf.forEach(searchInSchema);
      if (schema.items) searchInSchema(schema.items);
      if (schema.properties) {
        Object.values(schema.properties).forEach(searchInSchema);
      }
    };

    for (const endpoint of spec.endpoints || []) {
      if (endpoint.parameters) {
        endpoint.parameters.forEach(p => searchInSchema(p.schema || p));
      }
      if (endpoint.requestBody) {
        searchInSchema(endpoint.requestBody);
      }
      if (endpoint.responses) {
        Object.values(endpoint.responses).forEach(r => searchInSchema(r));
      }
    }

    return Array.from(usedSchemas);
  }
}

module.exports = new APIVersionManager();
