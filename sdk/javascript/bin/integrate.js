#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const SDK_VERSION = '18.0.0';
const TEMPLATES_DIR = path.join(__dirname, '../templates');

const colors = {
    reset: '\x1b[0m',
    bright: '\x1b[1m',
    green: '\x1b[32m',
    blue: '\x1b[34m',
    yellow: '\x1b[33m',
    red: '\x1b[31m'
};

function log(message, color = 'reset') {
    console.log(`${colors[color]}${message}${colors.reset}`);
}

function error(message) {
    log(`Error: ${message}`, 'red');
    process.exit(1);
}

function success(message) {
    log(`✓ ${message}`, 'green');
}

function info(message) {
    log(`ℹ ${message}`, 'blue');
}

function warn(message) {
    log(`⚠ ${message}`, 'yellow');
}

class SDKIntegrator {
    constructor() {
        this.projectType = null;
        this.projectPath = process.cwd();
        this.config = {};
    }

    detectProjectType() {
        if (fs.existsSync(path.join(this.projectPath, 'package.json'))) {
            const pkg = JSON.parse(fs.readFileSync(path.join(this.projectPath, 'package.json'), 'utf8'));
            if (pkg.dependencies?.react || pkg.devDependencies?.react) {
                return 'react';
            }
            if (pkg.dependencies?.vue || pkg.devDependencies?.vue) {
                return 'vue';
            }
            if (pkg.dependencies?.angular) {
                return 'angular';
            }
            if (pkg.dependencies?.next) {
                return 'next';
            }
            return 'node';
        }
        if (fs.existsSync(path.join(this.projectPath, 'index.html'))) {
            return 'vanilla';
        }
        return 'unknown';
    }

    async integrate(options = {}) {
        log(`\n${colors.bright}HJTPX Unified SDK v${SDK_VERSION} Integration Tool${colors.reset}\n`);
        log(`Detecting project type...\n`);

        this.projectType = options.type || this.detectProjectType();
        info(`Project type: ${this.projectType}`);

        this.config = {
            apiKey: options.apiKey || process.env.HJTPX_API_KEY || '',
            baseURL: options.baseURL || 'https://api.hjtpx.com',
            enableCache: options.enableCache !== false,
            enableMetrics: options.enableMetrics !== false
        };

        await this.integrateCore();
        await this.integratePlatformSpecific();

        if (options.examples !== false) {
            await this.createExamples();
        }

        if (options.tests !== false) {
            await this.createTests();
        }

        this.printSummary();
    }

    async integrateCore() {
        info('Installing SDK package...');

        const sdkContent = fs.readFileSync(path.join(__dirname, '../javascript/unified-sdk.js'), 'utf8');

        const distDir = path.join(this.projectPath, 'node_modules/@hjtpx/sdk');
        if (!fs.existsSync(distDir)) {
            fs.mkdirSync(distDir, { recursive: true });
        }

        fs.writeFileSync(path.join(distDir, 'unified-sdk.js'), sdkContent);
        fs.writeFileSync(path.join(distDir, 'package.json'), JSON.stringify({
            name: '@hjtpx/sdk',
            version: SDK_VERSION,
            main: 'unified-sdk.js'
        }, null, 2));

        success('Core SDK integrated');
    }

    async integratePlatformSpecific() {
        info(`Integrating ${this.projectType}-specific code...`);

        const template = this.getPlatformTemplate();

        const srcDir = path.join(this.projectPath, 'src');
        if (!fs.existsSync(srcDir)) {
            fs.mkdirSync(srcDir, { recursive: true });
        }

        fs.writeFileSync(
            path.join(srcDir, 'hjtpx-init.js'),
            template.main
        );

        if (template.config) {
            fs.writeFileSync(
                path.join(this.projectPath, 'hjtpx.config.js'),
                template.config
            );
        }

        if (template.html) {
            fs.writeFileSync(
                path.join(this.projectPath, 'index-hjtpx.html'),
                template.html
            );
        }

        success(`${this.projectType} integration complete`);
    }

    getPlatformTemplate() {
        const templates = {
            react: {
                main: `import { useEffect, useState } from 'react';
import HjtpxSDK from '@hjtpx/sdk';

const config = ${JSON.stringify(this.config, null, 2)};

export function useHjtpxSDK() {
  const [sdk, setSdk] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    const initSDK = async () => {
      try {
        const instance = HjtpxSDK(config);
        await instance.initialize();
        setSdk(instance);
        setLoading(false);
      } catch (err) {
        setError(err);
        setLoading(false);
      }
    };

    initSDK();

    return () => {
      if (sdk) {
        sdk.destroy();
      }
    };
  }, []);

  return { sdk, loading, error };
}

export function HjtpxProvider({ children }) {
  const { sdk, loading, error } = useHjtpxSDK();

  if (loading) return <div>Loading SDK...</div>;
  if (error) return <div>SDK Error: {error.message}</div>;

  return (
    <HjtpxContext.Provider value={sdk}>
      {children}
    </HjtpxContext.Provider>
  );
}

export default HjtpxSDK;
`,
                config: `module.exports = ${JSON.stringify({
                    apiKey: this.config.apiKey,
                    baseURL: this.config.baseURL,
                    enableCache: this.config.enableCache,
                    enableMetrics: this.config.enableMetrics
                }, null, 2)};
`
            },
            vue: {
                main: `import { createApp } from 'vue';
import HjtpxSDK from '@hjtpx/sdk';

const config = ${JSON.stringify(this.config, null, 2)};

export async function createHjtpxPlugin() {
  const sdk = HjtpxSDK(config);
  await sdk.initialize();

  return {
    install(app) {
      app.config.globalProperties.$hjtpx = sdk;
      app.provide('hjtpx', sdk);
    },
    sdk
  };
}

export default HjtpxSDK;
`,
                config: `module.exports = ${JSON.stringify(this.config, null, 2)};
`
            },
            node: {
                main: `const HjtpxSDK = require('@hjtpx/sdk');

const config = ${JSON.stringify(this.config, null, 2)};

const sdk = HjtpxSDK(config);

async function initialize() {
  await sdk.initialize();
  console.log('HJTPX SDK initialized');
  return sdk;
}

async function createCaptcha() {
  const result = await sdk.getCaptcha({ appId: 'your-app-id' });
  return result;
}

async function verifyCaptcha(token, params) {
  const result = await sdk.verifyCaptcha({ token, ...params });
  return result;
}

module.exports = { sdk, initialize, createCaptcha, verifyCaptcha };
`
            },
            vanilla: {
                main: `(function() {
  'use strict';

  var config = ${JSON.stringify(this.config, null, 2)};

  function initHjtpx() {
    var script = document.createElement('script');
    script.src = 'node_modules/@hjtpx/sdk/unified-sdk.js';
    script.onload = function() {
      var sdk = window.HjtpxSDK(config);
      sdk.initialize().then(function() {
        window.hjtpxSDK = sdk;
        document.dispatchEvent(new CustomEvent('hjtpx:ready'));
        console.log('HJTPX SDK initialized');
      });
    };
    document.head.appendChild(script);
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initHjtpx);
  } else {
    initHjtpx();
  }
})();
`,
                html: `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>HJTPX SDK Demo</title>
  <script src="src/hjtpx-init.js"></script>
</head>
<body>
  <h1>HJTPX SDK Demo</h1>
  <div id="captcha-container"></div>
  <div id="status"></div>

  <script>
    document.addEventListener('hjtpx:ready', function() {
      document.getElementById('status').textContent = 'SDK Ready!';

      window.hjtpxSDK.getCaptcha({ appId: 'demo-app' })
        .then(function(result) {
          console.log('Captcha created:', result);
        })
        .catch(function(err) {
          console.error('Captcha error:', err);
        });
    });
  </script>
</body>
</html>
`
            }
        };

        return templates[this.projectType] || templates.vanilla;
    }

    async createExamples() {
        info('Creating usage examples...');

        const examplesDir = path.join(this.projectPath, 'examples');
        if (!fs.existsSync(examplesDir)) {
            fs.mkdirSync(examplesDir, { recursive: true });
        }

        const exampleContent = `// HJTPX SDK v${SDK_VERSION} Usage Examples

const HjtpxSDK = require('@hjtpx/sdk');

const sdk = HjtpxSDK({
  apiKey: process.env.HJTPX_API_KEY,
  baseURL: 'https://api.hjtpx.com'
});

async function examples() {
  await sdk.initialize();
  console.log('SDK Ready!');

  // Example 1: Create Captcha
  const captcha = await sdk.getCaptcha({
    appId: 'your-app-id',
    type: 'slider'
  });
  console.log('Captcha:', captcha);

  // Example 2: Verify Captcha
  const verifyResult = await sdk.verifyCaptcha({
    sessionId: captcha.sessionId,
    x: 150,
    trajectory: [
      { x: 0, y: 0, t: Date.now() },
      { x: 50, y: 10, t: Date.now() + 100 }
    ]
  });
  console.log('Verification:', verifyResult);

  // Example 3: Report Risk
  const riskResult = await sdk.reportRisk({
    sessionId: 'session-123',
    events: [{ type: 'click', x: 100, y: 200 }]
  });
  console.log('Risk:', riskResult);

  // Example 4: Device Authentication
  const deviceAuth = await sdk.authenticateDevice({
    deviceId: 'device-123',
    fingerprint: await sdk.getDeviceFingerprint()
  });
  console.log('Device Auth:', deviceAuth);

  // Example 5: Blockchain Proof
  const proof = await sdk.recordBlockchainProof({
    recordId: 'record-123',
    eventType: 'verification_success'
  });
  console.log('Proof:', proof);

  // Example 6: Get Metrics
  const metrics = sdk.getMetrics();
  console.log('SDK Metrics:', metrics);

  sdk.destroy();
}

examples().catch(console.error);
`;

        fs.writeFileSync(path.join(examplesDir, 'basic-usage.js'), exampleContent);
        success('Examples created');
    }

    async createTests() {
        info('Creating test file...');

        const testsDir = path.join(this.projectPath, 'tests');
        if (!fs.existsSync(testsDir)) {
            fs.mkdirSync(testsDir, { recursive: true });
        }

        const testContent = `// HJTPX SDK Integration Tests
// Run with: npm test

const HjtpxSDK = require('@hjtpx/sdk');

describe('HJTPX SDK Integration', function() {
  this.timeout(30000);

  let sdk;

  before(async function() {
    sdk = HjtpxSDK({
      apiKey: process.env.HJTPX_API_KEY,
      baseURL: process.env.HJTPX_API_URL || 'https://api.hjtpx.com',
      enableMetrics: false
    });
    await sdk.initialize();
  });

  after(function() {
    if (sdk) sdk.destroy();
  });

  describe('Initialization', function() {
    it('should initialize successfully', function() {
      sdk.initialized.should.be.true;
    });

    it('should have correct config', function() {
      sdk.config.should.have.property('apiKey');
      sdk.config.should.have.property('baseURL');
    });
  });

  describe('Captcha Operations', function() {
    it('should create captcha', async function() {
      const result = await sdk.getCaptcha({ appId: 'test-app' });
      result.should.have.property('sessionId');
    });
  });
});
`;

        fs.writeFileSync(path.join(testsDir, 'sdk-integration.test.js'), testContent);
        success('Test file created');
    }

    printSummary() {
        console.log('\n' + colors.bright + '='.repeat(50) + colors.reset);
        console.log(colors.green + '✓ Integration Complete!' + colors.reset);
        console.log('='.repeat(50) + '\n');

        console.log('Next steps:');
        console.log('  1. Update your API key in hjtpx.config.js');
        console.log('  2. Import SDK in your code:');
        console.log(`     const HjtpxSDK = require('@hjtpx/sdk');`);
        console.log('');
        console.log('  3. Initialize SDK:');
        console.log('     const sdk = HjtpxSDK(config);');
        console.log('     await sdk.initialize();');
        console.log('');
        console.log('  4. Use SDK methods:');
        console.log('     sdk.getCaptcha({ appId: "your-app" })');
        console.log('     sdk.verifyCaptcha({ token, params })');
        console.log('');
        console.log(`For more examples, see: examples/basic-usage.js`);
        console.log('');
    }
}

function main() {
    const args = process.argv.slice(2);
    const options = {};

    for (let i = 0; i < args.length; i++) {
        const arg = args[i];
        switch (arg) {
            case '--type':
            case '-t':
                options.type = args[++i];
                break;
            case '--api-key':
            case '-k':
                options.apiKey = args[++i];
                break;
            case '--base-url':
            case '-b':
                options.baseURL = args[++i];
                break;
            case '--no-examples':
                options.examples = false;
                break;
            case '--no-tests':
                options.tests = false;
                break;
            case '--help':
            case '-h':
                printHelp();
                process.exit(0);
                break;
            case '--version':
            case '-v':
                console.log(`HJTPX SDK Integration Tool v${SDK_VERSION}`);
                process.exit(0);
                break;
        }
    }

    const integrator = new SDKIntegrator();
    integrator.integrate(options).catch(error);
}

function printHelp() {
    console.log(`
HJTPX Unified SDK v${SDK_VERSION} Integration Tool

Usage:
  npx hjtpx-integrate [options]

Options:
  -t, --type <type>       Project type (react, vue, node, vanilla)
  -k, --api-key <key>     Your API key
  -b, --base-url <url>    API base URL
  --no-examples           Skip creating example files
  --no-tests             Skip creating test files
  -h, --help              Show this help message
  -v, --version           Show version

Examples:
  npx hjtpx-integrate -t react -k your-api-key
  npx hjtpx-integrate --type vue
  npx hjtpx-integrate --no-examples
`);
}

main();
