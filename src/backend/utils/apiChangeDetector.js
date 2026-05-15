const fs = require('fs');
const path = require('path');

class ApiChangeDetector {
  constructor(specsDir = './docs/versions') {
    this.specsDir = specsDir;
    this.ensureDirExists();
  }

  ensureDirExists() {
    if (!fs.existsSync(this.specsDir)) {
      fs.mkdirSync(this.specsDir, { recursive: true });
    }
  }

  getLatestVersionPath() {
    const files = fs.readdirSync(this.specsDir)
      .filter(f => f.endsWith('.json') && f.startsWith('openapi-'))
      .sort()
      .reverse();
    return files.length > 0 ? path.join(this.specsDir, files[0]) : null;
  }

  loadSpec(filePath) {
    if (!fs.existsSync(filePath)) {
      return null;
    }
    return JSON.parse(fs.readFileSync(filePath, 'utf-8'));
  }

  saveSpec(spec, version = null) {
    const versionStr = version || spec.info.version;
    const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
    const filename = `openapi-${versionStr}-${timestamp}.json`;
    const filePath = path.join(this.specsDir, filename);
    fs.writeFileSync(filePath, JSON.stringify(spec, null, 2));
    return filePath;
  }

  compareSpecs(oldSpec, newSpec) {
    const changes = {
      added: [],
      removed: [],
      modified: [],
      breaking: [],
      nonBreaking: []
    };

    if (!oldSpec) {
      changes.added.push({ type: 'spec', message: 'Initial API specification' });
      return changes;
    }

    const oldPaths = oldSpec.paths || {};
    const newPaths = newSpec.paths || {};

    const allPaths = new Set([
      ...Object.keys(oldPaths),
      ...Object.keys(newPaths)
    ]);

    allPaths.forEach(pathKey => {
      const oldPath = oldPaths[pathKey];
      const newPath = newPaths[pathKey];

      if (!oldPath) {
        changes.added.push({ type: 'endpoint', path: pathKey, message: `Added endpoint: ${pathKey}` });
        changes.nonBreaking.push({ type: 'endpoint', path: pathKey, message: `Added endpoint: ${pathKey}` });
      } else if (!newPath) {
        changes.removed.push({ type: 'endpoint', path: pathKey, message: `Removed endpoint: ${pathKey}` });
        changes.breaking.push({ type: 'endpoint', path: pathKey, message: `Removed endpoint: ${pathKey}` });
      } else {
        this.comparePath(pathKey, oldPath, newPath, changes);
      }
    });

    this.compareComponents(oldSpec.components || {}, newSpec.components || {}, changes);

    return changes;
  }

  comparePath(pathKey, oldPath, newPath, changes) {
    const allMethods = new Set([
      ...Object.keys(oldPath),
      ...Object.keys(newPath)
    ]);

    allMethods.forEach(method => {
      const oldMethod = oldPath[method];
      const newMethod = newPath[method];

      if (!oldMethod) {
        changes.added.push({ type: 'method', path: pathKey, method, message: `Added ${method.toUpperCase()} ${pathKey}` });
        changes.nonBreaking.push({ type: 'method', path: pathKey, method, message: `Added ${method.toUpperCase()} ${pathKey}` });
      } else if (!newMethod) {
        changes.removed.push({ type: 'method', path: pathKey, method, message: `Removed ${method.toUpperCase()} ${pathKey}` });
        changes.breaking.push({ type: 'method', path: pathKey, method, message: `Removed ${method.toUpperCase()} ${pathKey}` });
      } else {
        this.compareOperation(pathKey, method, oldMethod, newMethod, changes);
      }
    });
  }

  compareOperation(pathKey, method, oldOp, newOp, changes) {
    const oldParams = (oldOp.parameters || []).map(p => `${p.name}:${p.in}`);
    const newParams = (newOp.parameters || []).map(p => `${p.name}:${p.in}`);

    oldParams.forEach(paramKey => {
      if (!newParams.includes(paramKey)) {
        changes.removed.push({ type: 'parameter', path: pathKey, method, param: paramKey, message: `Removed parameter ${paramKey} from ${method.toUpperCase()} ${pathKey}` });
        changes.breaking.push({ type: 'parameter', path: pathKey, method, param: paramKey, message: `Removed parameter ${paramKey} from ${method.toUpperCase()} ${pathKey}` });
      }
    });

    newParams.forEach(paramKey => {
      if (!oldParams.includes(paramKey)) {
        const param = (newOp.parameters || []).find(p => `${p.name}:${p.in}` === paramKey);
        if (param && param.required) {
          changes.added.push({ type: 'parameter', path: pathKey, method, param: paramKey, message: `Added required parameter ${paramKey} to ${method.toUpperCase()} ${pathKey}` });
          changes.breaking.push({ type: 'parameter', path: pathKey, method, param: paramKey, message: `Added required parameter ${paramKey} to ${method.toUpperCase()} ${pathKey}` });
        } else {
          changes.added.push({ type: 'parameter', path: pathKey, method, param: paramKey, message: `Added optional parameter ${paramKey} to ${method.toUpperCase()} ${pathKey}` });
          changes.nonBreaking.push({ type: 'parameter', path: pathKey, method, param: paramKey, message: `Added optional parameter ${paramKey} to ${method.toUpperCase()} ${pathKey}` });
        }
      }
    });

    const oldResponses = Object.keys(oldOp.responses || {});
    const newResponses = Object.keys(newOp.responses || {});

    oldResponses.forEach(status => {
      if (!newResponses.includes(status)) {
        changes.removed.push({ type: 'response', path: pathKey, method, status, message: `Removed response ${status} from ${method.toUpperCase()} ${pathKey}` });
        changes.breaking.push({ type: 'response', path: pathKey, method, status, message: `Removed response ${status} from ${method.toUpperCase()} ${pathKey}` });
      }
    });
  }

  compareComponents(oldComponents, newComponents, changes) {
    const oldSchemas = Object.keys(oldComponents.schemas || {});
    const newSchemas = Object.keys(newComponents.schemas || {});

    oldSchemas.forEach(schema => {
      if (!newSchemas.includes(schema)) {
        changes.removed.push({ type: 'schema', schema, message: `Removed schema: ${schema}` });
        changes.breaking.push({ type: 'schema', schema, message: `Removed schema: ${schema}` });
      }
    });

    newSchemas.forEach(schema => {
      if (!oldSchemas.includes(schema)) {
        changes.added.push({ type: 'schema', schema, message: `Added schema: ${schema}` });
        changes.nonBreaking.push({ type: 'schema', schema, message: `Added schema: ${schema}` });
      }
    });
  }

  generateChangeReport(changes, outputFile = null) {
    const report = {
      timestamp: new Date().toISOString(),
      summary: {
        total: changes.added.length + changes.removed.length + changes.modified.length,
        added: changes.added.length,
        removed: changes.removed.length,
        modified: changes.modified.length,
        breaking: changes.breaking.length,
        nonBreaking: changes.nonBreaking.length
      },
      details: changes
    };

    if (outputFile) {
      fs.writeFileSync(outputFile, JSON.stringify(report, null, 2));
      console.log(`📄 Change report saved to: ${outputFile}`);
    }

    return report;
  }

  printChangeSummary(changes) {
    console.log('\n📊 API Change Summary:');
    console.log('========================================');
    console.log(`Total changes: ${changes.added.length + changes.removed.length + changes.modified.length}`);
    console.log(`  Added: ${changes.added.length}`);
    console.log(`  Removed: ${changes.removed.length}`);
    console.log(`  Modified: ${changes.modified.length}`);
    console.log(`  ⚠️ Breaking: ${changes.breaking.length}`);
    console.log(`  ✅ Non-breaking: ${changes.nonBreaking.length}`);
    console.log('========================================\n');

    if (changes.breaking.length > 0) {
      console.log('⚠️ BREAKING CHANGES:');
      changes.breaking.forEach(change => {
        console.log(`  - ${change.message}`);
      });
      console.log('');
    }
  }

  checkForChanges(currentSpec, autoSave = true) {
    const latestPath = this.getLatestVersionPath();
    const latestSpec = latestPath ? this.loadSpec(latestPath) : null;

    const changes = this.compareSpecs(latestSpec, currentSpec);
    this.printChangeSummary(changes);

    if (autoSave && (changes.added.length > 0 || changes.removed.length > 0 || changes.modified.length > 0)) {
      const savedPath = this.saveSpec(currentSpec);
      console.log(`💾 Saved new spec version: ${savedPath}`);
      
      const reportPath = path.join(this.specsDir, `changes-${currentSpec.info.version}.json`);
      this.generateChangeReport(changes, reportPath);
    }

    return changes;
  }
}

module.exports = ApiChangeDetector;
