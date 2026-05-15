const express = require('express');
const request = require('supertest');
const {
  versionNegotiator,
  deprecationWarning,
  VERSIONS,
  DEFAULT_VERSION,
  SUPPORTED_VERSIONS,
  LATEST_STABLE_VERSION,
  getVersionInfo,
  isVersionSupported,
  getDeprecationStatus,
  getMigrationInfo
} = require('../src/backend/middleware/versionControl');

const createTestApp = () => {
  const app = express();
  app.use(versionNegotiator);
  app.use(deprecationWarning);
  
  app.use('/api/v1', VERSIONS.v1.routes);
  app.use('/api/v2', VERSIONS.v2.routes);
  
  app.get('/api/health', (req, res) => {
    res.json({
      success: true,
      data: {
        message: 'API is healthy',
        negotiatedVersion: req.apiVersion,
        negotiationDetails: req.versionNegotiation
      }
    });
  });
  
  app.get('/', (req, res) => {
    res.json({
      success: true,
      data: {
        apiVersions: SUPPORTED_VERSIONS,
        defaultVersion: DEFAULT_VERSION,
        latestVersion: LATEST_STABLE_VERSION
      }
    });
  });
  
  return app;
};

describe('API Version Control Comprehensive Tests', () => {
  let app;
  
  beforeAll(() => {
    app = createTestApp();
  });
  
  describe('Version Negotiation Tests', () => {
    test('should negotiate version from URL path for v1', async () => {
      const response = await request(app).get('/api/v1');
      expect(response.status).toBe(200);
      expect(response.headers['x-api-version']).toBe('v1');
      expect(response.headers['x-api-version-status']).toBe('stable');
      expect(response.headers['x-api-latest-version']).toBe('v2');
    });
    
    test('should negotiate version from URL path for v2', async () => {
      const response = await request(app).get('/api/v2');
      expect(response.status).toBe(200);
      expect(response.headers['x-api-version']).toBe('v2');
      expect(response.headers['x-api-version-status']).toBe('stable');
    });
    
    test('should negotiate version from Accept header', async () => {
      const response = await request(app)
        .get('/api/health')
        .set('Accept', 'application/vnd.hjtpx.v1+json');
      
      expect(response.status).toBe(200);
      expect(response.headers['x-api-version']).toBe('v1');
      expect(response.headers['x-api-version-negotiated']).toBe('true');
      expect(response.headers['x-api-original-version']).toBe('v1');
    });
    
    test('should negotiate version from X-API-Version header', async () => {
      const response = await request(app)
        .get('/api/health')
        .set('X-API-Version', 'v1');
      
      expect(response.status).toBe(200);
      expect(response.headers['x-api-version']).toBe('v1');
      expect(response.headers['x-api-version-negotiated']).toBe('true');
    });
    
    test('should negotiate version from Prefer header', async () => {
      const response = await request(app)
        .get('/api/health')
        .set('Prefer', 'version=v1');
      
      expect(response.status).toBe(200);
      expect(response.headers['x-api-version']).toBe('v1');
    });
    
    test('should use default version when no version specified', async () => {
      const response = await request(app).get('/api/health');
      expect(response.status).toBe(200);
      expect(response.headers['x-api-version']).toBe('v2');
      expect(response.headers['x-api-latest-version']).toBe('v2');
    });
    
    test('should include supported versions header', async () => {
      const response = await request(app).get('/api/health');
      expect(response.headers['x-api-supported-versions']).toBe('v1, v2');
    });
    
    test('should include negotiation details in response', async () => {
      const response = await request(app)
        .get('/api/health')
        .set('Accept', 'application/vnd.hjtpx.v1+json');
      
      expect(response.status).toBe(200);
      expect(response.body.data.negotiatedVersion).toBe('v1');
      expect(response.body.data.negotiationDetails).toBeDefined();
      expect(response.body.data.negotiationDetails.resolvedVersion).toBe('v1');
      expect(response.body.data.negotiationDetails.negotiationMethod).toBe('accept-header');
    });
  });
  
  describe('Deprecation Warning Tests', () => {
    test('should include deprecation headers for v1', async () => {
      const response = await request(app).get('/api/v1/health');
      expect(response.status).toBe(200);
      expect(response.headers['warning']).toBeDefined();
      expect(response.headers['warning']).toContain('deprecated');
      expect(response.headers['x-api-deprecation-date']).toBe('2026-01-01');
      expect(response.headers['x-api-sunset-date']).toBe('2026-07-01');
    });
    
    test('should include migration guide header for v1', async () => {
      const response = await request(app).get('/api/v1/health');
      expect(response.headers['x-api-migration-guide']).toBeDefined();
      expect(response.headers['x-api-migration-guide']).toContain('v1-migration-guide');
    });
    
    test('should include breaking changes count header', async () => {
      const response = await request(app).get('/api/v1/health');
      expect(response.headers['x-api-breaking-changes']).toBe('3');
    });
    
    test('should include days until sunset header', async () => {
      const response = await request(app).get('/api/v1/health');
      expect(response.headers['x-api-days-until-sunset']).toBeDefined();
      const days = parseInt(response.headers['x-api-days-until-sunset']);
      expect(days).toBeGreaterThan(0);
      expect(days).toBeLessThanOrEqual(47);
    });
    
    test('should include urgent warning when sunset is near', async () => {
      const response = await request(app).get('/api/v1/health');
      const days = parseInt(response.headers['x-api-days-until-sunset']);
      
      if (days <= 30) {
        expect(response.headers['warning']).toContain('urgent');
      }
    });
    
    test('should NOT include deprecation headers for v2', async () => {
      const response = await request(app).get('/api/v2/health');
      expect(response.status).toBe(200);
      expect(response.headers['warning']).toBeUndefined();
      expect(response.headers['x-api-deprecation-date']).toBeUndefined();
      expect(response.headers['x-api-sunset-date']).toBeUndefined();
    });
    
    test('should include deprecation info in response body for v1', async () => {
      const response = await request(app).get('/api/v1/health');
      expect(response.status).toBe(200);
      expect(response.body.deprecation).toBeDefined();
      expect(response.body.deprecation.deprecated).toBe(true);
      expect(response.body.deprecation.currentVersion).toBe('v1');
      expect(response.body.deprecation.latestVersion).toBe('v2');
      expect(response.body.deprecation.migrationGuide).toBeDefined();
      expect(response.body.deprecation.breakingChanges).toBeInstanceOf(Array);
      expect(response.body.deprecation.breakingChanges.length).toBe(3);
    });
    
    test('should NOT include deprecation info in response body for v2', async () => {
      const response = await request(app).get('/api/v2/health');
      expect(response.status).toBe(200);
      expect(response.body.deprecation).toBeUndefined();
    });
    
    test('should include upgrade header when version not available', async () => {
      const response = await request(app)
        .get('/api/health')
        .set('X-API-Version', 'v3');
      
      expect(response.headers['x-api-version-upgrade']).toBeDefined();
      expect(response.headers['x-api-version-upgrade']).toContain('v3');
      expect(response.headers['x-api-version-upgrade']).toContain('v2');
    });
  });
  
  describe('Version Coexistence Tests', () => {
    test('v1 and v2 should coexist', async () => {
      const v1Response = await request(app).get('/api/v1/health');
      const v2Response = await request(app).get('/api/v2/health');
      
      expect(v1Response.status).toBe(200);
      expect(v2Response.status).toBe(200);
      expect(v1Response.headers['x-api-version']).toBe('v1');
      expect(v2Response.headers['x-api-version']).toBe('v2');
    });
    
    test('v1 should have different data structure than v2', async () => {
      const v1Response = await request(app).get('/api/v1/health');
      const v2Response = await request(app).get('/api/v2/health');
      
      expect(v1Response.status).toBe(200);
      expect(v2Response.status).toBe(200);
      
      expect(v1Response.body.data).toBeDefined();
      expect(v2Response.body.data).toBeDefined();
      
      expect(v1Response.body.data.version).toBe('1.0.0');
      expect(v2Response.body.data.version).toBe('v2');
    });
    
    test('both versions should be accessible simultaneously', async () => {
      const promises = [
        request(app).get('/api/v1/health'),
        request(app).get('/api/v2/health')
      ];
      
      const responses = await Promise.all(promises);
      
      responses.forEach(response => {
        expect(response.status).toBe(200);
      });
      
      expect(responses[0].headers['x-api-version']).toBe('v1');
      expect(responses[1].headers['x-api-version']).toBe('v2');
    });
    
    test('version switching should work correctly', async () => {
      const v1First = await request(app).get('/api/v1/health');
      const v2Switch = await request(app).get('/api/v2/health');
      const v1Second = await request(app).get('/api/v1/health');
      
      expect(v1First.headers['x-api-version']).toBe('v1');
      expect(v2Switch.headers['x-api-version']).toBe('v2');
      expect(v1Second.headers['x-api-version']).toBe('v1');
    });
  });
  
  describe('Version Information Utility Functions', () => {
    test('getVersionInfo should return version details', () => {
      const v1Info = getVersionInfo('v1');
      expect(v1Info).toBeDefined();
      expect(v1Info.version).toBe('v1');
      expect(v1Info.deprecated).toBe(true);
      
      const v2Info = getVersionInfo('v2');
      expect(v2Info).toBeDefined();
      expect(v2Info.version).toBe('v2');
      expect(v2Info.deprecated).toBe(false);
      
      const unknownInfo = getVersionInfo('v99');
      expect(unknownInfo).toBeNull();
    });
    
    test('isVersionSupported should check version support', () => {
      expect(isVersionSupported('v1')).toBe(true);
      expect(isVersionSupported('v2')).toBe(true);
      expect(isVersionSupported('v3')).toBe(false);
      expect(isVersionSupported('unknown')).toBe(false);
    });
    
    test('getDeprecationStatus should return deprecation info', () => {
      const v1Status = getDeprecationStatus('v1');
      expect(v1Status.supported).toBe(true);
      expect(v1Status.deprecated).toBe(true);
      expect(v1Status.sunsetDate).toBe('2026-07-01');
      expect(v1Status.daysUntilSunset).toBeGreaterThan(0);
      
      const v2Status = getDeprecationStatus('v2');
      expect(v2Status.supported).toBe(true);
      expect(v2Status.deprecated).toBe(false);
      expect(v2Status.sunsetDate).toBeNull();
      
      const unknownStatus = getDeprecationStatus('v99');
      expect(unknownStatus.supported).toBe(false);
    });
    
    test('getMigrationInfo should return migration details', () => {
      const v1Migration = getMigrationInfo('v1');
      expect(v1Migration).toBeDefined();
      expect(v1Migration.currentVersion).toBe('v1');
      expect(v1Migration.latestVersion).toBe('v2');
      expect(v1Migration.isLatest).toBe(false);
      expect(v1Migration.breakingChanges).toBeInstanceOf(Array);
      expect(v1Migration.migrationGuide).toBeDefined();
      
      const v2Migration = getMigrationInfo('v2');
      expect(v2Migration.currentVersion).toBe('v2');
      expect(v2Migration.isLatest).toBe(true);
      expect(v2Migration.breakingChanges.length).toBe(0);
      
      const unknownMigration = getMigrationInfo('v99');
      expect(unknownMigration).toBeNull();
    });
  });
  
  describe('Response Format Tests', () => {
    test('v1 should include deprecation in response body', async () => {
      const response = await request(app).get('/api/v1');
      expect(response.body.deprecation).toBeDefined();
      expect(response.body.deprecation.message).toContain('deprecated');
    });
    
    test('v2 should NOT include deprecation in response body', async () => {
      const response = await request(app).get('/api/v2');
      expect(response.body.deprecation).toBeUndefined();
    });
    
    test('all responses should include version headers', async () => {
      const v1Response = await request(app).get('/api/v1');
      const v2Response = await request(app).get('/api/v2');
      
      expect(v1Response.headers['x-api-version']).toBeDefined();
      expect(v2Response.headers['x-api-version']).toBeDefined();
      expect(v1Response.headers['x-api-version-status']).toBeDefined();
      expect(v2Response.headers['x-api-version-status']).toBeDefined();
    });
  });
  
  describe('Error Handling Tests', () => {
    test('should handle unsupported version gracefully', async () => {
      const response = await request(app)
        .get('/api/health')
        .set('X-API-Version', 'v99');
      
      expect(response.status).toBe(200);
      expect(response.headers['x-api-version']).toBe('v2');
      expect(response.headers['x-api-version-upgrade']).toBeDefined();
    });
    
    test('should handle invalid version header gracefully', async () => {
      const response = await request(app)
        .get('/api/health')
        .set('X-API-Version', 'invalid');
      
      expect(response.status).toBe(200);
      expect(response.headers['x-api-version']).toBe('v2');
    });
  });
  
  describe('Performance Tests', () => {
    test('version negotiation should complete quickly', async () => {
      const startTime = Date.now();
      
      await request(app).get('/api/v1/health');
      await request(app).get('/api/v2/health');
      
      const duration = Date.now() - startTime;
      expect(duration).toBeLessThan(1000);
    });
    
    test('should handle concurrent requests', async () => {
      const concurrentRequests = Array(10).fill(null).map(() => 
        request(app).get('/api/v1/health')
      );
      
      const responses = await Promise.all(concurrentRequests);
      
      responses.forEach(response => {
        expect(response.status).toBe(200);
        expect(response.headers['x-api-version']).toBe('v1');
      });
    });
  });
});

describe('Version Control Constants', () => {
  test('SUPPORTED_VERSIONS should include all versions', () => {
    expect(SUPPORTED_VERSIONS).toContain('v1');
    expect(SUPPORTED_VERSIONS).toContain('v2');
    expect(SUPPORTED_VERSIONS.length).toBeGreaterThanOrEqual(2);
  });
  
  test('DEFAULT_VERSION should be latest stable', () => {
    expect(DEFAULT_VERSION).toBe('v2');
    expect(DEFAULT_VERSION).toBe(LATEST_STABLE_VERSION);
  });
  
  test('VERSIONS should have required properties', () => {
    expect(VERSIONS.v1).toHaveProperty('version', 'v1');
    expect(VERSIONS.v1).toHaveProperty('status');
    expect(VERSIONS.v1).toHaveProperty('deprecated');
    expect(VERSIONS.v1).toHaveProperty('sunsetDate');
    expect(VERSIONS.v1).toHaveProperty('routes');
    
    expect(VERSIONS.v2).toHaveProperty('version', 'v2');
    expect(VERSIONS.v2).toHaveProperty('status');
    expect(VERSIONS.v2).toHaveProperty('deprecated');
    expect(VERSIONS.v2).toHaveProperty('routes');
  });
  
  test('v1 should have migration guide', () => {
    expect(VERSIONS.v1.migrationGuide).toBeDefined();
    expect(VERSIONS.v1.migrationGuide).toContain('migration-guide');
  });
  
  test('v1 should have breaking changes', () => {
    expect(VERSIONS.v1.breakingChanges).toBeInstanceOf(Array);
    expect(VERSIONS.v1.breakingChanges.length).toBeGreaterThan(0);
  });
});

console.log('Running comprehensive API versioning tests...');
