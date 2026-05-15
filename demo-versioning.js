const express = require('express');
const { versionNegotiator, deprecationWarning, VERSIONS, DEFAULT_VERSION, SUPPORTED_VERSIONS } = require('./src/backend/middleware/versionControl');

const app = express();
const PORT = 3001;

app.use(versionNegotiator);
app.use(deprecationWarning);

app.get('/', (req, res) => {
  res.json({
    message: 'HJTPX API Version Control Demo',
    default_version: DEFAULT_VERSION,
    supported_versions: SUPPORTED_VERSIONS,
    documentation: '/docs',
    endpoints: {
      v1: '/api/v1',
      v2: '/api/v2'
    }
  });
});

app.use('/api/v1', VERSIONS.v1.routes);
app.use('/api/v2', VERSIONS.v2.routes);

console.log('🎯 HJTPX API Version Control Demo');
console.log('=================================');
console.log('📋 Features:');
console.log('  - v1 (deprecated): Old API format');
console.log('  - v2 (current): Enhanced API with pagination');
console.log('');
console.log('🔍 Try these endpoints:');
console.log('  GET / - List versions');
console.log('  GET /api/v1/health - Check v1 health (with deprecation warning)');
console.log('  GET /api/v2/health - Check v2 health');
console.log('  GET /api/v1/users - List users (v1)');
console.log('  GET /api/v2/users - List users (v2 with pagination)');
console.log('');
console.log(`🚀 Server starting on http://localhost:${PORT}`);

app.listen(PORT, () => {
  console.log('✅ Demo server is running!');
});
