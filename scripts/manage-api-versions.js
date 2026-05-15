const fs = require('fs');
const path = require('path');
const { generateSwaggerSpec } = require('../src/backend/config/swagger-auto');
const ApiVersionManager = require('../src/backend/utils/apiVersionManager');
const ApiChangeDetector = require('../src/backend/utils/apiChangeDetector');

const args = process.argv.slice(2);
const command = args[0];

const versionsDir = process.env.API_VERSIONS_DIR || './docs/versions';
const versionManager = new ApiVersionManager(versionsDir);
const changeDetector = new ApiChangeDetector(versionsDir);

function printHelp() {
  console.log(`
API Version Management Tool
============================

Usage: node scripts/manage-api-versions.js <command> [options]

Commands:
  list                        List all API versions
  save [description]          Save current API as a new version
  show <version>              Show details of a specific version
  compare <v1> <v2>           Compare two versions
  diff <v1> <v2>              Show detailed diff between versions
  delete <version>            Delete a specific version
  current                     Generate and show current API spec
  validate                    Validate current API spec
  export [version] [format]   Export version to JSON/YAML

Examples:
  node scripts/manage-api-versions.js list
  node scripts/manage-api-versions.js save "Added new search endpoints"
  node scripts/manage-api-versions.js compare 1.0.0 1.1.0
  node scripts/manage-api-versions.js export 1.0.0 json
  `);
}

async function listVersions() {
  console.log('📚 API Versions:\n');
  const versions = versionManager.getVersions();
  
  if (versions.length === 0) {
    console.log('No versions saved yet. Run "save" to create the first version.');
    return;
  }
  
  console.log('| Version | Endpoints | Created | Description |');
  console.log('|---------|-----------|---------|-------------|');
  
  versions.forEach(v => {
    const date = new Date(v.createdAt).toLocaleDateString();
    const desc = v.description || '-';
    console.log(`| ${v.version} | ${v.endpoints} | ${date} | ${desc} |`);
  });
  console.log('');
}

async function saveVersion(description = '') {
  console.log('💾 Saving current API version...\n');
  
  try {
    const spec = generateSwaggerSpec();
    const versionInfo = versionManager.saveVersion(spec, description);
    
    console.log('✅ Version saved successfully!');
    console.log(`   Version: ${versionInfo.version}`);
    console.log(`   Endpoints: ${versionInfo.endpoints}`);
    console.log(`   File: ${versionInfo.filepath}`);
    console.log(`   Description: ${description || 'N/A'}`);
  } catch (error) {
    console.error('❌ Error saving version:', error.message);
    process.exit(1);
  }
}

async function showVersion(version) {
  console.log(`📄 API Version ${version}:\n`);
  
  const spec = versionManager.loadVersionSpec(version);
  if (!spec) {
    console.error(`❌ Version ${version} not found.`);
    process.exit(1);
  }
  
  console.log(`Title: ${spec.info.title}`);
  console.log(`Version: ${spec.info.version}`);
  console.log(`Description: ${spec.info.description || 'N/A'}`);
  console.log(`OpenAPI: ${spec.openapi}`);
  console.log(`Endpoints: ${Object.keys(spec.paths || {}).length}`);
  console.log(`Schemas: ${Object.keys(spec.components?.schemas || {}).length}`);
  console.log(`Tags: ${(spec.tags || []).map(t => t.name).join(', ')}`);
  
  console.log('\n📍 Endpoints:');
  Object.entries(spec.paths || {}).forEach(([path, methods]) => {
    const methodList = Object.keys(methods)
      .filter(m => ['get', 'post', 'put', 'delete', 'patch'].includes(m))
      .join(', ');
    console.log(`   ${methodList} ${path}`);
  });
}

async function compareVersions(v1, v2) {
  console.log(`🔄 Comparing versions ${v1} vs ${v2}:\n`);
  
  const changes = versionManager.compareVersions(v1, v2);
  if (!changes) {
    console.error(`❌ One or both versions not found.`);
    process.exit(1);
  }
  
  if (changes.added.length === 0 && changes.removed.length === 0) {
    console.log('✅ No differences found between versions.');
    return;
  }
  
  if (changes.added.length > 0) {
    console.log(`✨ Added in ${v2} (${changes.added.length}):`);
    changes.added.forEach(p => console.log(`   + ${p}`));
    console.log('');
  }
  
  if (changes.removed.length > 0) {
    console.log(`🗑️  Removed from ${v1} (${changes.removed.length}):`);
    changes.removed.forEach(p => console.log(`   - ${p}`));
    console.log('');
  }
}

async function diffVersions(v1, v2) {
  console.log(`📊 Detailed diff between ${v1} and ${v2}:\n`);
  
  const spec1 = versionManager.loadVersionSpec(v1);
  const spec2 = versionManager.loadVersionSpec(v2);
  
  if (!spec1 || !spec2) {
    console.error('❌ One or both versions not found.');
    process.exit(1);
  }
  
  const changes = changeDetector.compareSpecs(spec1, spec2);
  
  console.log('Summary:');
  console.log(`  Total added: ${changes.added.length}`);
  console.log(`  Total removed: ${changes.removed.length}`);
  console.log(`  Breaking changes: ${changes.breaking.length}`);
  console.log(`  Non-breaking changes: ${changes.nonBreaking.length}`);
  console.log('');
  
  if (changes.breaking.length > 0) {
    console.log('⚠️  Breaking Changes:');
    changes.breaking.forEach(c => console.log(`   - ${c.message}`));
    console.log('');
  }
  
  if (changes.nonBreaking.length > 0) {
    console.log('✅ Non-breaking Changes:');
    changes.nonBreaking.forEach(c => console.log(`   - ${c.message}`));
    console.log('');
  }
}

async function deleteVersion(version) {
  const versions = versionManager.getVersions();
  const exists = versions.some(v => v.version === version);
  
  if (!exists) {
    console.error(`❌ Version ${version} not found.`);
    process.exit(1);
  }
  
  const confirmed = args[1] === '--force' || 
    await new Promise(resolve => {
      const readline = require('readline');
      const rl = readline.createInterface({ input: process.stdin, output: process.stdout });
      rl.question(`Delete version ${version}? (y/N) `, answer => {
        rl.close();
        resolve(answer.toLowerCase() === 'y');
      });
    });
  
  if (!confirmed) {
    console.log('Cancelled.');
    return;
  }
  
  versionManager.deleteVersion(version);
  console.log(`✅ Version ${version} deleted.`);
}

async function showCurrent() {
  console.log('📋 Current API Specification:\n');
  
  try {
    const spec = generateSwaggerSpec();
    console.log(JSON.stringify(spec, null, 2));
  } catch (error) {
    console.error('❌ Error generating spec:', error.message);
    process.exit(1);
  }
}

async function validateSpec() {
  console.log('🔍 Validating current API specification...\n');
  
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
    
    Object.entries(spec.paths || {}).forEach(([pathKey, methods]) => {
      Object.entries(methods || {}).forEach(([method, operation]) => {
        if (!['get', 'post', 'put', 'delete', 'patch', 'options', 'head'].includes(method)) return;
        
        if (!operation.summary && !operation.description) {
          warnings.push(`Missing summary/description for ${method.toUpperCase()} ${pathKey}`);
        }
        if (!operation.responses) {
          warnings.push(`Missing responses for ${method.toUpperCase()} ${pathKey}`);
        }
      });
    });
    
    if (errors.length > 0) {
      console.log('❌ Validation Errors:');
      errors.forEach(err => console.log(`   - ${err}`));
      console.log('');
    }
    
    if (warnings.length > 0) {
      console.log('⚠️  Validation Warnings:');
      warnings.forEach(warn => console.log(`   - ${warn}`));
      console.log('');
    }
    
    if (isValid) {
      console.log('✅ Specification is valid');
      console.log(`   Version: ${spec.info.version}`);
      console.log(`   Title: ${spec.info.title}`);
      console.log(`   Endpoints: ${Object.keys(spec.paths || {}).length}`);
    }
    
    process.exit(isValid ? 0 : 1);
  } catch (error) {
    console.error('❌ Error validating spec:', error.message);
    process.exit(1);
  }
}

async function exportVersion(version, format = 'json') {
  console.log(`📤 Exporting version ${version} as ${format.toUpperCase()}...\n`);
  
  const spec = versionManager.loadVersionSpec(version);
  if (!spec) {
    console.error(`❌ Version ${version} not found.`);
    process.exit(1);
  }
  
  const outputDir = './docs/exports';
  if (!fs.existsSync(outputDir)) {
    fs.mkdirSync(outputDir, { recursive: true });
  }
  
  const filename = `openapi-${version}.${format}`;
  const filepath = path.join(outputDir, filename);
  
  if (format === 'json') {
    fs.writeFileSync(filepath, JSON.stringify(spec, null, 2));
  } else if (format === 'yaml' || format === 'yml') {
    try {
      const yaml = require('js-yaml');
      fs.writeFileSync(filepath, yaml.dump(spec));
    } catch (e) {
      console.error('❌ YAML export failed (js-yaml not available)');
      process.exit(1);
    }
  } else {
    console.error(`❌ Unknown format: ${format}`);
    process.exit(1);
  }
  
  console.log(`✅ Exported to: ${filepath}`);
}

switch (command) {
  case 'list':
    listVersions();
    break;
  case 'save':
    saveVersion(args.slice(1).join(' ') || '');
    break;
  case 'show':
    if (!args[1]) {
      console.error('❌ Please specify a version');
      printHelp();
      process.exit(1);
    }
    showVersion(args[1]);
    break;
  case 'compare':
    if (!args[1] || !args[2]) {
      console.error('❌ Please specify two versions to compare');
      printHelp();
      process.exit(1);
    }
    compareVersions(args[1], args[2]);
    break;
  case 'diff':
    if (!args[1] || !args[2]) {
      console.error('❌ Please specify two versions to diff');
      printHelp();
      process.exit(1);
    }
    diffVersions(args[1], args[2]);
    break;
  case 'delete':
    if (!args[1]) {
      console.error('❌ Please specify a version to delete');
      printHelp();
      process.exit(1);
    }
    deleteVersion(args[1]);
    break;
  case 'current':
    showCurrent();
    break;
  case 'validate':
    validateSpec();
    break;
  case 'export':
    exportVersion(args[1] || 'current', args[2] || 'json');
    break;
  default:
    printHelp();
}

module.exports = {
  listVersions,
  saveVersion,
  showVersion,
  compareVersions,
  diffVersions,
  deleteVersion,
  showCurrent,
  validateSpec,
  exportVersion
};
