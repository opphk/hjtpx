const { generateSwaggerSpec } = require('../src/backend/config/swagger-auto');
const ApiChangeDetector = require('../src/backend/utils/apiChangeDetector');
const ApiVersionManager = require('../src/backend/utils/apiVersionManager');

const args = process.argv.slice(2);
const options = {
  autoSave: true,
  verbose: false,
  format: 'console',
  outputDir: './docs/versions'
};

for (let i = 0; i < args.length; i++) {
  if (args[i] === '--no-save') {
    options.autoSave = false;
  } else if (args[i] === '--verbose' || args[i] === '-v') {
    options.verbose = true;
  } else if (args[i] === '--format' && args[i + 1]) {
    options.format = args[i + 1];
    i++;
  } else if (args[i] === '--output' && args[i + 1]) {
    options.outputDir = args[i + 1];
    i++;
  } else if (args[i] === '--help' || args[i] === '-h') {
    printHelp();
    process.exit(0);
  }
}

function printHelp() {
  console.log(`
API Change Detection Tool
==========================

Usage: node scripts/check-api-changes.js [options]

Options:
  --no-save          Don't save the new API version automatically
  --verbose, -v      Show detailed change information
  --format <type>     Output format: console, json, markdown (default: console)
  --output <dir>     Output directory for version files (default: ./docs/versions)
  --help, -h         Show this help message

Examples:
  node scripts/check-api-changes.js
  node scripts/check-api-changes.js --verbose
  node scripts/check-api-changes.js --format json --output ./custom-docs/versions
  `);
}

console.log('🔍 Checking for API changes...');

try {
  const currentSpec = generateSwaggerSpec();
  const detector = new ApiChangeDetector(options.outputDir);
  const versionManager = new ApiVersionManager(options.outputDir);
  
  const changes = detector.checkForChanges(currentSpec, options.autoSave);
  
  if (options.format === 'json') {
    const report = detector.generateChangeReport(changes);
    console.log(JSON.stringify(report, null, 2));
  } else if (options.format === 'markdown') {
    const report = generateMarkdownReport(changes, currentSpec);
    console.log(report);
  } else {
    console.log('\n📊 API Change Summary:');
    console.log('========================================');
    console.log(`Version: ${currentSpec.info.version}`);
    console.log(`Total changes: ${changes.added.length + changes.removed.length + changes.modified.length}`);
    console.log(`  Added: ${changes.added.length}`);
    console.log(`  Removed: ${changes.removed.length}`);
    console.log(`  Modified: ${changes.modified.length}`);
    console.log(`  ⚠️ Breaking: ${changes.breaking.length}`);
    console.log(`  ✅ Non-breaking: ${changes.nonBreaking.length}`);
    console.log('========================================\n');

    if (options.verbose) {
      if (changes.added.length > 0) {
        console.log('✨ Added:');
        changes.added.forEach(change => {
          console.log(`  - ${change.message}`);
        });
        console.log('');
      }

      if (changes.removed.length > 0) {
        console.log('🗑️  Removed:');
        changes.removed.forEach(change => {
          console.log(`  - ${change.message}`);
        });
        console.log('');
      }

      if (changes.modified.length > 0) {
        console.log('📝 Modified:');
        changes.modified.forEach(change => {
          console.log(`  - ${change.message}`);
        });
        console.log('');
      }
    }

    if (changes.breaking.length > 0) {
      console.log('⚠️ BREAKING CHANGES DETECTED:');
      changes.breaking.forEach(change => {
        console.log(`  - ${change.message}`);
      });
      console.log('');
    }
  }

  if (changes.breaking.length > 0) {
    console.log('❌ Breaking changes detected!');
    process.exit(1);
  } else {
    console.log('✅ No breaking changes detected.');
    process.exit(0);
  }
} catch (error) {
  console.error('❌ Error checking API changes:', error.message);
  console.error(error.stack);
  process.exit(1);
}

function generateMarkdownReport(changes, spec) {
  let markdown = `# API Change Report\n\n`;
  markdown += `**Generated:** ${new Date().toISOString()}\n\n`;
  markdown += `**Version:** ${spec.info.version}\n\n`;
  
  markdown += `## Summary\n\n`;
  markdown += `| Metric | Count |\n`;
  markdown += `|--------|-------|\n`;
  markdown += `| Added | ${changes.added.length} |\n`;
  markdown += `| Removed | ${changes.removed.length} |\n`;
  markdown += `| Modified | ${changes.modified.length} |\n`;
  markdown += `| Breaking | ${changes.breaking.length} |\n`;
  markdown += `| Non-breaking | ${changes.nonBreaking.length} |\n\n`;

  if (changes.added.length > 0) {
    markdown += `## Added\n\n`;
    changes.added.forEach(change => {
      markdown += `- ${change.message}\n`;
    });
    markdown += '\n';
  }

  if (changes.removed.length > 0) {
    markdown += `## Removed\n\n`;
    changes.removed.forEach(change => {
      markdown += `- ${change.message}\n`;
    });
    markdown += '\n';
  }

  if (changes.modified.length > 0) {
    markdown += `## Modified\n\n`;
    changes.modified.forEach(change => {
      markdown += `- ${change.message}\n`;
    });
    markdown += '\n';
  }

  return markdown;
}
