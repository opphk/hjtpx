const { generateSwaggerSpec, saveSwaggerSpec } = require('../../config/swagger-auto');
const ApiChangeDetector = require('../../utils/apiChangeDetector');
const ApiVersionManager = require('../../utils/apiVersionManager');
const path = require('path');
const fs = require('fs');

describe('API Documentation System', () => {
  const testVersionsDir = './docs/test-versions';
  let detector;
  let versionManager;

  beforeAll(() => {
    if (!fs.existsSync(testVersionsDir)) {
      fs.mkdirSync(testVersionsDir, { recursive: true });
    }
    detector = new ApiChangeDetector(testVersionsDir);
    versionManager = new ApiVersionManager(testVersionsDir);
  });

  afterAll(() => {
    if (fs.existsSync(testVersionsDir)) {
      fs.rmSync(testVersionsDir, { recursive: true, force: true });
    }
  });

  describe('Swagger Specification Generation', () => {
    test('should generate valid OpenAPI spec', () => {
      const spec = generateSwaggerSpec();
      
      expect(spec).toBeDefined();
      expect(spec.openapi).toBe('3.0.0');
      expect(spec.info).toBeDefined();
      expect(spec.info.title).toBe('HJTPX API Documentation');
      expect(spec.info.version).toBeDefined();
    });

    test('should include required components', () => {
      const spec = generateSwaggerSpec();
      
      expect(spec.components).toBeDefined();
      expect(spec.components.schemas).toBeDefined();
      expect(spec.components.securitySchemes).toBeDefined();
      expect(spec.components.responses).toBeDefined();
    });

    test('should have valid security schemes', () => {
      const spec = generateSwaggerSpec();
      
      expect(spec.components.securitySchemes.bearerAuth).toBeDefined();
      expect(spec.components.securitySchemes.bearerAuth.type).toBe('http');
      expect(spec.components.securitySchemes.bearerAuth.scheme).toBe('bearer');
    });

    test('should have required schemas', () => {
      const spec = generateSwaggerSpec();
      
      expect(spec.components.schemas.User).toBeDefined();
      expect(spec.components.schemas.Error).toBeDefined();
      expect(spec.components.schemas.SuccessResponse).toBeDefined();
    });
  });

  describe('API Change Detection', () => {
    test('should detect initial spec as new', () => {
      const newSpec = generateSwaggerSpec();
      const changes = detector.compareSpecs(null, newSpec);
      
      expect(changes.added.length).toBeGreaterThan(0);
      expect(changes.breaking.length).toBe(0);
    });

    test('should detect added endpoints', () => {
      const oldSpec = {
        info: { version: '1.0.0' },
        paths: {},
        components: { schemas: {} }
      };
      
      const newSpec = {
        info: { version: '1.1.0' },
        paths: {
          '/api/test': {
            get: { summary: 'Test endpoint' }
          }
        },
        components: { schemas: {} }
      };
      
      const changes = detector.compareSpecs(oldSpec, newSpec);
      
      expect(changes.added.length).toBe(1);
      expect(changes.added[0].type).toBe('endpoint');
      expect(changes.added[0].path).toBe('/api/test');
    });

    test('should detect removed endpoints as breaking', () => {
      const oldSpec = {
        info: { version: '1.0.0' },
        paths: {
          '/api/old': {
            get: { summary: 'Old endpoint' }
          }
        },
        components: { schemas: {} }
      };
      
      const newSpec = {
        info: { version: '1.1.0' },
        paths: {},
        components: { schemas: {} }
      };
      
      const changes = detector.compareSpecs(oldSpec, newSpec);
      
      expect(changes.removed.length).toBe(1);
      expect(changes.breaking.length).toBe(1);
      expect(changes.breaking[0].type).toBe('endpoint');
    });

    test('should generate change report', () => {
      const oldSpec = {
        info: { version: '1.0.0' },
        paths: {},
        components: { schemas: {} }
      };
      
      const newSpec = generateSwaggerSpec();
      const changes = detector.compareSpecs(oldSpec, newSpec);
      const report = detector.generateChangeReport(changes);
      
      expect(report).toBeDefined();
      expect(report.timestamp).toBeDefined();
      expect(report.summary).toBeDefined();
      expect(report.summary.total).toBeDefined();
      expect(report.summary.added).toBeDefined();
    });
  });

  describe('API Version Management', () => {
    test('should save version spec', () => {
      const spec = generateSwaggerSpec();
      const versionInfo = versionManager.saveVersion(spec, 'Test version');
      
      expect(versionInfo).toBeDefined();
      expect(versionInfo.version).toBe(spec.info.version);
      expect(versionInfo.filepath).toBeDefined();
      expect(fs.existsSync(versionInfo.filepath)).toBe(true);
    });

    test('should list saved versions', () => {
      const versions = versionManager.getVersions();
      
      expect(Array.isArray(versions)).toBe(true);
    });

    test('should load saved version', () => {
      const spec = generateSwaggerSpec();
      const loaded = versionManager.loadVersionSpec(spec.info.version);
      
      expect(loaded).toBeDefined();
      expect(loaded.info.version).toBe(spec.info.version);
    });

    test('should compare two versions', () => {
      const changes = versionManager.compareVersions('1.0.0', '1.0.0');
      
      expect(changes).toBeDefined();
      expect(changes.added).toBeDefined();
      expect(changes.removed).toBeDefined();
    });
  });

  describe('Documentation Validation', () => {
    test('should validate required fields', () => {
      const spec = generateSwaggerSpec();
      
      expect(spec.openapi).toBeDefined();
      expect(spec.info).toBeDefined();
      expect(spec.info.title).toBeDefined();
      expect(spec.info.version).toBeDefined();
      expect(spec.servers).toBeDefined();
      expect(Array.isArray(spec.servers)).toBe(true);
    });

    test('should have valid server configuration', () => {
      const spec = generateSwaggerSpec();
      
      spec.servers.forEach(server => {
        expect(server.url).toBeDefined();
        expect(server.description).toBeDefined();
      });
    });

    test('should have valid tags', () => {
      const spec = generateSwaggerSpec();
      
      expect(spec.tags).toBeDefined();
      expect(Array.isArray(spec.tags)).toBe(true);
      
      spec.tags.forEach(tag => {
        expect(tag.name).toBeDefined();
        expect(tag.description).toBeDefined();
      });
    });
  });

  describe('Error Responses', () => {
    test('should have standard error responses', () => {
      const spec = generateSwaggerSpec();
      
      expect(spec.components.responses.Unauthorized).toBeDefined();
      expect(spec.components.responses.Forbidden).toBeDefined();
      expect(spec.components.responses.NotFound).toBeDefined();
      expect(spec.components.responses.ValidationError).toBeDefined();
      expect(spec.components.responses.InternalServerError).toBeDefined();
    });

    test('should reference error schema in responses', () => {
      const spec = generateSwaggerSpec();
      
      Object.values(spec.components.responses).forEach(response => {
        if (response.content && response.content['application/json']) {
          expect(response.content['application/json'].schema).toBeDefined();
        }
      });
    });
  });
});
