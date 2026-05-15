const { saveSwaggerSpec, generateSwaggerSpec } = require('../src/backend/config/swagger-auto');

const args = process.argv.slice(2);
const outputDir = args[0] || './docs';

console.log('🔧 Generating Swagger documentation...');
try {
  const spec = saveSwaggerSpec(outputDir);
  console.log(`\n✅ Documentation generated successfully!`);
  console.log(`   Version: ${spec.info.version}`);
  console.log(`   Endpoints: ${Object.keys(spec.paths).length}`);
  console.log(`   Tags: ${spec.tags.length}`);
  process.exit(0);
} catch (error) {
  console.error('❌ Error generating Swagger documentation:', error.message);
  process.exit(1);
}
