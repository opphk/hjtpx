const fs = require('fs');
const path = require('path');

class ApiVersionManager {
  constructor(versionsDir = './docs/versions') {
    this.versionsDir = versionsDir;
    this.metadataFile = path.join(this.versionsDir, 'versions.json');
    this.ensureDirExists();
  }

  ensureDirExists() {
    if (!fs.existsSync(this.versionsDir)) {
      fs.mkdirSync(this.versionsDir, { recursive: true });
    }
  }

  loadMetadata() {
    if (fs.existsSync(this.metadataFile)) {
      return JSON.parse(fs.readFileSync(this.metadataFile, 'utf-8'));
    }
    return { versions: [] };
  }

  saveMetadata(metadata) {
    fs.writeFileSync(this.metadataFile, JSON.stringify(metadata, null, 2));
  }

  saveVersion(spec, description = '') {
    const version = spec.info.version;
    const timestamp = new Date().toISOString();
    const filename = `openapi-${version}.json`;
    const filepath = path.join(this.versionsDir, filename);

    fs.writeFileSync(filepath, JSON.stringify(spec, null, 2));

    const metadata = this.loadMetadata();
    const existingIndex = metadata.versions.findIndex(v => v.version === version);

    const versionInfo = {
      version,
      createdAt: timestamp,
      description,
      filename,
      filepath,
      endpoints: Object.keys(spec.paths || {}).length
    };

    if (existingIndex >= 0) {
      metadata.versions[existingIndex] = { ...metadata.versions[existingIndex], ...versionInfo };
    } else {
      metadata.versions.unshift(versionInfo);
    }

    this.saveMetadata(metadata);
    return versionInfo;
  }

  getVersions() {
    const metadata = this.loadMetadata();
    return metadata.versions;
  }

  getVersion(version) {
    const metadata = this.loadMetadata();
    return metadata.versions.find(v => v.version === version);
  }

  loadVersionSpec(version) {
    const versionInfo = this.getVersion(version);
    if (!versionInfo) {
      return null;
    }
    const filepath = path.join(this.versionsDir, versionInfo.filename);
    if (fs.existsSync(filepath)) {
      return JSON.parse(fs.readFileSync(filepath, 'utf-8'));
    }
    return null;
  }

  deleteVersion(version) {
    const metadata = this.loadMetadata();
    const index = metadata.versions.findIndex(v => v.version === version);
    if (index >= 0) {
      const versionInfo = metadata.versions[index];
      const filepath = path.join(this.versionsDir, versionInfo.filename);
      if (fs.existsSync(filepath)) {
        fs.unlinkSync(filepath);
      }
      metadata.versions.splice(index, 1);
      this.saveMetadata(metadata);
      return true;
    }
    return false;
  }

  compareVersions(version1, version2) {
    const spec1 = this.loadVersionSpec(version1);
    const spec2 = this.loadVersionSpec(version2);

    if (!spec1 || !spec2) {
      return null;
    }

    const changes = {
      added: [],
      removed: [],
      modified: []
    };

    const paths1 = Object.keys(spec1.paths || {});
    const paths2 = Object.keys(spec2.paths || {});

    paths2.forEach(path => {
      if (!paths1.includes(path)) {
        changes.added.push(path);
      }
    });

    paths1.forEach(path => {
      if (!paths2.includes(path)) {
        changes.removed.push(path);
      }
    });

    return changes;
  }
}

module.exports = ApiVersionManager;
