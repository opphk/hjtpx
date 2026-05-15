const { generateSwaggerSpec } = require('../src/backend/config/swagger-auto');

console.log('🔍 Validating Swagger specification...');

try {
  const spec = generateSwaggerSpec();
  
  let isValid = true;
  const errors = [];
  const warnings = [];

  if (!spec.openapi) {
    errors.push('Missing openapi version');
    isValid = false;
  }

  if (!spec.info || !spec.info.title) {
    errors.push('Missing info.title');
    isValid = false;
  }

  if (!spec.info || !spec.info.version) {
    errors.push('Missing info.version');
    isValid = false;
  }

  if (!spec.paths || Object.keys(spec.paths).length === 0) {
    warnings.push('No API paths defined');
  }

  Object.entries(spec.paths || {}).forEach(([path, methods]) => {
    Object.entries(methods || {}).forEach(([method, operation]) => {
      if (!operation.summary && !operation.description) {
        warnings.push(`Missing summary/description for ${method.toUpperCase()} ${path}`);
      }
      if (!operation.responses) {
        warnings.push(`Missing responses for ${method.toUpperCase()} ${path}`);
      }
    });
  });

  if (errors.length > 0) {
    console.error('❌ Validation Errors:');
    errors.forEach(err => console.error(`  - ${err}`));
  }

  if (warnings.length > 0) {
    console.warn('⚠️  Validation Warnings:');
    warnings.forEach(warn => console.warn(`  - ${warn}`));
  }

  if (isValid) {
    console.log('✅ Swagger specification is valid');
    console.log(`   Version: ${spec.info.version}`);
    console.log(`   Title: ${spec.info.title}`);
    console.log(`   Endpoints: ${Object.keys(spec.paths || {}).length}`);
    process.exit(0);
  } else {
    process.exit(1);
  }

} catch (error) {
  console.error('❌ Error validating Swagger:', error.message);
  process.exit(1);
}
