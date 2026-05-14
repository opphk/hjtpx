const { Pool } = require('pg');
const fs = require('fs');
const path = require('path');
require('dotenv').config();

const DB_HOST = process.env.DB_HOST || 'localhost';
const DB_PORT = process.env.DB_PORT || 5432;
const DB_NAME = process.env.DB_NAME || 'hjtpx';
const DB_USER = process.env.DB_USER || 'postgres';
const DB_PASSWORD = process.env.DB_PASSWORD || 'postgres';
const DB_MAX_CONNECTIONS = parseInt(process.env.DB_MAX_CONNECTIONS) || 20;
const DB_IDLE_TIMEOUT = parseInt(process.env.DB_IDLE_TIMEOUT) || 30000;
const DB_CONNECTION_TIMEOUT = parseInt(process.env.DB_CONNECTION_TIMEOUT) || 2000;

const MIGRATIONS_TABLE = 'schema_migrations';
const MIGRATIONS_DIR = path.join(__dirname, '../migrations');
const ROLLBACKS_DIR = path.join(__dirname, '../migrations/rollbacks');

class MigrationPerformance {
  constructor() {
    this.metrics = {
      totalMigrations: 0,
      successfulMigrations: 0,
      failedMigrations: 0,
      totalDuration: 0,
      avgDuration: 0,
      migrations: []
    };
  }

  startMigration(name) {
    return {
      name,
      startTime: Date.now(),
      status: 'running'
    };
  }

  completeMigration(migration) {
    migration.duration = Date.now() - migration.startTime;
    migration.status = 'completed';
    this.metrics.totalDuration += migration.duration;
    this.metrics.successfulMigrations++;
    this.metrics.migrations.push(migration);
    return migration;
  }

  failMigration(migration, error) {
    migration.duration = Date.now() - migration.startTime;
    migration.status = 'failed';
    migration.error = error.message;
    this.metrics.failedMigrations++;
    this.metrics.migrations.push(migration);
    return migration;
  }

  getMetrics() {
    this.metrics.avgDuration = this.metrics.totalMigrations > 0 
      ? (this.metrics.totalDuration / this.metrics.totalMigrations) 
      : 0;
    return {
      ...this.metrics,
      totalMigrations: this.metrics.successfulMigrations + this.metrics.failedMigrations,
      successRate: this.metrics.totalMigrations > 0 
        ? ((this.metrics.successfulMigrations / this.metrics.totalMigrations) * 100).toFixed(2) + '%'
        : '0%'
    };
  }
}

class MigrationTracker {
  constructor(pool) {
    this.pool = pool;
  }

  async createTable() {
    await this.pool.query(`
      CREATE TABLE IF NOT EXISTS ${MIGRATIONS_TABLE} (
        id SERIAL PRIMARY KEY,
        name VARCHAR(255) NOT NULL UNIQUE,
        applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        duration INTEGER,
        status VARCHAR(50) DEFAULT 'completed',
        checksum VARCHAR(64),
        batch INTEGER DEFAULT 1,
        metadata JSONB
      )
    `);
  }

  async getAppliedMigrations() {
    const result = await this.pool.query(
      `SELECT name, applied_at, checksum, batch, metadata 
       FROM ${MIGRATIONS_TABLE} 
       ORDER BY applied_at ASC`
    );
    return result.rows;
  }

  async getPendingMigrations() {
    const applied = await this.getAppliedMigrations();
    const appliedNames = new Set(applied.map(m => m.name));
    const files = fs.readdirSync(MIGRATIONS_DIR)
      .filter(f => f.endsWith('.sql'))
      .sort();

    return files.filter(file => !appliedNames.has(file));
  }

  async recordMigration(name, checksum, duration, batch, metadata = {}) {
    await this.pool.query(
      `INSERT INTO ${MIGRATIONS_TABLE} (name, checksum, duration, batch, metadata)
       VALUES ($1, $2, $3, $4, $5)`,
      [name, checksum, duration, batch, JSON.stringify(metadata)]
    );
  }

  async removeMigrationRecord(name) {
    await this.pool.query(
      `DELETE FROM ${MIGRATIONS_TABLE} WHERE name = $1`,
      [name]
    );
  }

  async getMigrationStatus() {
    const applied = await this.getAppliedMigrations();
    const pending = await this.getPendingMigrations();
    const batchResult = await this.pool.query(
      `SELECT MAX(batch) as current_batch FROM ${MIGRATIONS_TABLE}`
    );
    const currentBatch = batchResult.rows[0]?.current_batch || 0;

    return {
      applied: applied.length,
      pending: pending.length,
      currentBatch,
      migrations: applied,
      pendingMigrations: pending
    };
  }

  async getBatchNumber() {
    const result = await this.pool.query(
      `SELECT COALESCE(MAX(batch), 0) as batch FROM ${MIGRATIONS_TABLE}`
    );
    return result.rows[0].batch + 1;
  }

  async calculateChecksum(filePath) {
    const crypto = require('crypto');
    const content = fs.readFileSync(filePath, 'utf8');
    return crypto.createHash('sha256').update(content).digest('hex');
  }

  async verifyMigration(name) {
    const result = await this.pool.query(
      `SELECT checksum FROM ${MIGRATIONS_TABLE} WHERE name = $1`,
      [name]
    );
    if (result.rows.length === 0) {
      return { verified: false, reason: 'Migration not found' };
    }

    const rollbackPath = path.join(ROLLBACKS_DIR, name.replace('.sql', '_rollback.sql'));
    if (fs.existsSync(rollbackPath)) {
      return { verified: true, hasRollback: true };
    }
    return { verified: true, hasRollback: false };
  }
}

class MigrationValidator {
  static validateMigrationName(name) {
    const pattern = /^\d{14}_[a-z_]+\.sql$/i;
    if (!pattern.test(name)) {
      throw new Error(
        `Invalid migration name: ${name}. Expected format: YYYYMMDDHHMMSS_description.sql`
      );
    }
    return true;
  }

  static validateSQL(sql) {
    const dangerous = [
      'DROP DATABASE',
      'DROP SCHEMA public',
      'TRUNCATE',
      'DELETE FROM users',
      'DELETE FROM sessions',
      'ALTER ROLE',
      'DROP ROLE'
    ];

    const upperSql = sql.toUpperCase();
    for (const pattern of dangerous) {
      if (upperSql.includes(pattern)) {
        throw new Error(`Potentially dangerous SQL detected: ${pattern}`);
      }
    }
    return true;
  }

  static validateRollback(sql) {
    if (!sql.toUpperCase().includes('ROLLBACK')) {
      console.warn('Warning: Rollback script should be tested');
    }
    return true;
  }
}

async function createMigrationsTable(pool) {
  await pool.query(`
    CREATE TABLE IF NOT EXISTS ${MIGRATIONS_TABLE} (
      id SERIAL PRIMARY KEY,
      name VARCHAR(255) NOT NULL UNIQUE,
      applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
      duration INTEGER,
      status VARCHAR(50) DEFAULT 'completed',
      checksum VARCHAR(64),
      batch INTEGER DEFAULT 1,
      metadata JSONB
    )
  `);

  await pool.query(`
    CREATE INDEX IF NOT EXISTS idx_migrations_batch 
    ON ${MIGRATIONS_TABLE}(batch)
  `);

  await pool.query(`
    CREATE INDEX IF NOT EXISTS idx_migrations_applied_at 
    ON ${MIGRATIONS_TABLE}(applied_at)
  `);
}

async function migrate(options = {}) {
  const performance = new MigrationPerformance();
  const pool = new Pool({
    host: DB_HOST,
    port: DB_PORT,
    database: DB_NAME,
    user: DB_USER,
    password: DB_PASSWORD,
    max: DB_MAX_CONNECTIONS,
    idleTimeoutMillis: DB_IDLE_TIMEOUT,
    connectionTimeoutMillis: DB_CONNECTION_TIMEOUT
  });

  const tracker = new MigrationTracker(pool);

  try {
    await createMigrationsTable(pool);

    if (options.rollback) {
      const targetName = options.to || null;
      await rollbackMigration(pool, tracker, targetName, performance, options);
    } else if (options.redo) {
      await redoMigration(pool, tracker, performance, options);
    } else if (options.reset) {
      await resetMigrations(pool, tracker, performance, options);
    } else {
      await runMigrations(pool, tracker, performance, options);
    }

    return {
      success: true,
      performance: performance.getMetrics(),
      status: await tracker.getMigrationStatus()
    };
  } catch (error) {
    console.error('Migration error:', error.message);
    return {
      success: false,
      error: error.message,
      performance: performance.getMetrics()
    };
  } finally {
    await pool.end();
  }
}

async function runMigrations(pool, tracker, performance, options) {
  const pending = await tracker.getPendingMigrations();

  if (pending.length === 0) {
    console.log('No pending migrations');
    return;
  }

  if (options.dryRun) {
    console.log('Dry run - would apply the following migrations:');
    pending.forEach(m => console.log(`  - ${m}`));
    return;
  }

  const batch = await tracker.getBatchNumber();
  console.log(`Starting migration batch ${batch}`);
  console.log(`Found ${pending.length} pending migration(s)`);

  if (!fs.existsSync(ROLLBACKS_DIR)) {
    fs.mkdirSync(ROLLBACKS_DIR, { recursive: true });
  }

  for (const file of pending) {
    const migration = performance.startMigration(file);
    const startTime = Date.now();

    console.log(`Applying migration: ${file}`);

    const sql = fs.readFileSync(path.join(MIGRATIONS_DIR, file), 'utf8');
    const checksum = await tracker.calculateChecksum(path.join(MIGRATIONS_DIR, file));

    MigrationValidator.validateSQL(sql);

    await pool.query('BEGIN');
    try {
      await pool.query(sql);
      const duration = Date.now() - startTime;
      await tracker.recordMigration(file, checksum, duration, batch, {
        executedAt: new Date().toISOString(),
        environment: process.env.NODE_ENV || 'development'
      });
      await pool.query('COMMIT');
      performance.completeMigration(migration);
      console.log(`Migration ${file} applied successfully (${duration}ms)`);

      if (options.generateRollback) {
        await generateRollbackScript(file, sql);
      }
    } catch (error) {
      await pool.query('ROLLBACK');
      performance.failMigration(migration, error);
      console.error(`Migration ${file} failed: ${error.message}`);
      throw error;
    }
  }
}

async function rollbackMigration(pool, tracker, targetName, performance, options) {
  const applied = await tracker.getAppliedMigrations();

  if (applied.length === 0) {
    console.log('No migrations to rollback');
    return;
  }

  const toRollback = targetName 
    ? applied.filter(m => m.name === targetName || applied.indexOf(m) === applied.length - 1)
    : [applied[applied.length - 1]];

  for (const migration of toRollback.reverse()) {
    const migrationPerf = performance.startMigration(migration.name);
    const startTime = Date.now();

    console.log(`Rolling back migration: ${migration.name}`);

    const rollbackFile = path.join(
      ROLLBACKS_DIR, 
      migration.name.replace('.sql', '_rollback.sql')
    );

    if (!fs.existsSync(rollbackFile)) {
      console.error(`Rollback script not found: ${rollbackFile}`);
      if (!options.force) {
        throw new Error(`Rollback script missing for ${migration.name}. Use --force to skip.`);
      }
      console.warn(`Skipping ${migration.name} - no rollback script found`);
      continue;
    }

    const rollbackSql = fs.readFileSync(rollbackFile, 'utf8');
    MigrationValidator.validateRollback(rollbackSql);

    await pool.query('BEGIN');
    try {
      await pool.query(rollbackSql);
      await tracker.removeMigrationRecord(migration.name);
      await pool.query('COMMIT');
      const duration = Date.now() - startTime;
      performance.completeMigration(migrationPerf);
      console.log(`Rollback of ${migration.name} completed (${duration}ms)`);
    } catch (error) {
      await pool.query('ROLLBACK');
      performance.failMigration(migrationPerf, error);
      console.error(`Rollback ${migration.name} failed: ${error.message}`);
      throw error;
    }
  }
}

async function redoMigration(pool, tracker, performance, options) {
  const applied = await tracker.getAppliedMigrations();

  if (applied.length === 0) {
    console.log('No migrations to redo');
    return;
  }

  const lastMigration = applied[applied.length - 1];
  console.log(`Redoing migration: ${lastMigration.name}`);

  const rollbackFile = path.join(
    ROLLBACKS_DIR, 
    lastMigration.name.replace('.sql', '_rollback.sql')
  );

  if (!fs.existsSync(rollbackFile)) {
    throw new Error(`Rollback script not found for ${lastMigration.name}`);
  }

  const rollbackSql = fs.readFileSync(rollbackFile, 'utf8');
  const migrationSql = fs.readFileSync(path.join(MIGRATIONS_DIR, lastMigration.name), 'utf8');

  await pool.query('BEGIN');
  try {
    await pool.query(rollbackSql);
    console.log(`Rolled back ${lastMigration.name}`);

    await pool.query(migrationSql);
    console.log(`Re-applied ${lastMigration.name}`);

    await pool.query('COMMIT');
  } catch (error) {
    await pool.query('ROLLBACK');
    throw error;
  }
}

async function resetMigrations(pool, tracker, performance, options) {
  const applied = await tracker.getAppliedMigrations();

  if (applied.length === 0) {
    console.log('No migrations to reset');
    return;
  }

  console.log(`Resetting ${applied.length} migrations`);

  for (const migration of applied.reverse()) {
    const rollbackFile = path.join(
      ROLLBACKS_DIR, 
      migration.name.replace('.sql', '_rollback.sql')
    );

    if (fs.existsSync(rollbackFile)) {
      const rollbackSql = fs.readFileSync(rollbackFile, 'utf8');

      await pool.query('BEGIN');
      try {
        await pool.query(rollbackSql);
        await tracker.removeMigrationRecord(migration.name);
        await pool.query('COMMIT');
        console.log(`Rolled back ${migration.name}`);
      } catch (error) {
        await pool.query('ROLLBACK');
        console.error(`Failed to rollback ${migration.name}: ${error.message}`);
        if (!options.force) {
          throw error;
        }
      }
    }
  }
}

async function generateRollbackScript(migrationName, sql) {
  const statements = sql.split(';').filter(s => s.trim());
  const rollbackStatements = [];

  for (const statement of statements.reverse()) {
    const trimmed = statement.trim();
    const rollback = generateRollbackStatement(trimmed);
    if (rollback) {
      rollbackStatements.push(rollback);
    }
  }

  const rollbackContent = `-- Rollback for: ${migrationName}
-- Generated: ${new Date().toISOString()}
-- This script reverts the changes made by ${migrationName}

BEGIN;

${rollbackStatements.join(';\n\n')};

COMMIT;
`;

  const rollbackFile = path.join(
    ROLLBACKS_DIR, 
    migrationName.replace('.sql', '_rollback.sql')
  );

  fs.writeFileSync(rollbackFile, rollbackContent);
  console.log(`Generated rollback script: ${path.basename(rollbackFile)}`);
}

function generateRollbackStatement(statement) {
  const upper = statement.toUpperCase();

  if (upper.startsWith('CREATE TABLE')) {
    const tableName = statement.match(/CREATE TABLE\s+(?:IF NOT EXISTS\s+)?(\w+)/i);
    if (tableName) {
      return `-- Drop table ${tableName[1]}\nDROP TABLE IF EXISTS ${tableName[1]} CASCADE;`;
    }
  }

  if (upper.startsWith('CREATE INDEX') || upper.startsWith('CREATE UNIQUE INDEX')) {
    const indexMatch = statement.match(/CREATE\s+(?:UNIQUE\s+)?INDEX\s+(?:IF NOT EXISTS\s+)?(\w+)/i);
    if (indexMatch) {
      return `-- Drop index ${indexMatch[1]}\nDROP INDEX IF EXISTS ${indexMatch[1]};`;
    }
  }

  if (upper.startsWith('ALTER TABLE')) {
    const addColumnMatch = statement.match(/ALTER TABLE (\w+) ADD COLUMN (\w+)/i);
    if (addColumnMatch) {
      return `-- Drop column ${addColumnMatch[2]} from ${addColumnMatch[1]}\nALTER TABLE ${addColumnMatch[1]} DROP COLUMN ${addColumnMatch[2]};`;
    }

    const addConstraintMatch = statement.match(/ALTER TABLE (\w+) ADD CONSTRAINT (\w+)/i);
    if (addConstraintMatch) {
      return `-- Drop constraint ${addConstraintMatch[2]} from ${addConstraintMatch[1]}\nALTER TABLE ${addConstraintMatch[1]} DROP CONSTRAINT ${addConstraintMatch[2]};`;
    }
  }

  if (upper.startsWith('CREATE FUNCTION') || upper.startsWith('CREATE PROCEDURE')) {
    const funcMatch = statement.match(/CREATE\s+(?:FUNCTION|PROCEDURE)\s+(\w+)/i);
    if (funcMatch) {
      return `-- Drop function/procedure ${funcMatch[1]}\nDROP FUNCTION IF EXISTS ${funcMatch[1]} CASCADE;`;
    }
  }

  if (upper.startsWith('CREATE TRIGGER')) {
    const triggerMatch = statement.match(/CREATE TRIGGER\s+(\w+)/i);
    if (triggerMatch) {
      return `-- Drop trigger ${triggerMatch[1]}\nDROP TRIGGER IF EXISTS ${triggerMatch[1]};`;
    }
  }

  if (upper.startsWith('CREATE TYPE')) {
    const typeMatch = statement.match(/CREATE TYPE\s+(\w+)/i);
    if (typeMatch) {
      return `-- Drop type ${typeMatch[1]}\nDROP TYPE IF EXISTS ${typeMatch[1]};`;
    }
  }

  if (upper.startsWith('INSERT INTO')) {
    return `-- Note: Manual data cleanup may be required for INSERT statements\n-- ${statement.substring(0, 100)}...`;
  }

  return null;
}

async function status(options = {}) {
  const pool = new Pool({
    host: DB_HOST,
    port: DB_PORT,
    database: DB_NAME,
    user: DB_USER,
    password: DB_PASSWORD
  });

  const tracker = new MigrationTracker(pool);

  try {
    const status = await tracker.getMigrationStatus();

    console.log('\n=== Migration Status ===');
    console.log(`Environment: ${process.env.NODE_ENV || 'development'}`);
    console.log(`Current Batch: ${status.currentBatch}`);
    console.log(`Applied: ${status.applied}`);
    console.log(`Pending: ${status.pending}`);
    
    if (status.applied > 0) {
      console.log('\nApplied migrations:');
      for (const m of status.migrations) {
        const rollbackExists = fs.existsSync(
          path.join(ROLLBACKS_DIR, m.name.replace('.sql', '_rollback.sql'))
        );
        console.log(`  ✓ ${m.name} (batch: ${m.batch}, rollback: ${rollbackExists ? 'yes' : 'no'})`);
      }
    }
    
    if (status.pending > 0) {
      console.log('\nPending migrations:');
      for (const m of status.pendingMigrations) {
        console.log(`  ○ ${m}`);
      }
    }
    console.log('========================\n');

    if (options.verbose) {
      const applied = await tracker.getAppliedMigrations();
      console.log('\nDetailed Migration History:');
      for (const m of applied) {
        console.log(`  ${m.name}:`);
        console.log(`    Applied: ${m.applied_at}`);
        console.log(`    Duration: ${m.duration}ms`);
        console.log(`    Status: ${m.status}`);
        if (m.metadata) {
          console.log(`    Metadata: ${JSON.stringify(m.metadata)}`);
        }
      }
    }

    return status;
  } catch (error) {
    console.error('Error getting migration status:', error.message);
    throw error;
  } finally {
    await pool.end();
  }
}

async function create(name, options = {}) {
  const timestamp = new Date().toISOString().replace(/[-:]/g, '').split('.')[0];
  const filename = `${timestamp}_${name}.sql`;
  const filepath = path.join(MIGRATIONS_DIR, filename);

  const template = `-- Migration: ${name}
-- Created: ${new Date().toISOString()}
-- Description: TODO: Add description

BEGIN;

-- TODO: Write migration SQL here

COMMIT;
`;

  fs.writeFileSync(filepath, template);
  console.log(`Created migration: ${filename}`);

  if (options.withRollback) {
    const rollbackFilename = `${timestamp}_${name}_rollback.sql`;
    const rollbackPath = path.join(ROLLBACKS_DIR, rollbackFilename);

    if (!fs.existsSync(ROLLBACKS_DIR)) {
      fs.mkdirSync(ROLLBACKS_DIR, { recursive: true });
    }

    const rollbackTemplate = `-- Rollback for: ${filename}
-- Created: ${new Date().toISOString()}

BEGIN;

-- TODO: Write rollback SQL here

COMMIT;
`;

    fs.writeFileSync(rollbackPath, rollbackTemplate);
    console.log(`Created rollback script: ${rollbackFilename}`);
  }
}

function main() {
  const args = process.argv.slice(2);
  const command = args[0];

  const options = {
    dryRun: args.includes('--dry-run'),
    force: args.includes('--force'),
    generateRollback: args.includes('--generate-rollback'),
    verbose: args.includes('--verbose')
  };

  const targetIndex = args.indexOf('--to');
  if (targetIndex !== -1 && args[targetIndex + 1]) {
    options.to = args[targetIndex + 1];
  }

  switch (command) {
    case 'up':
      migrate({ ...options, rollback: false })
        .then((result) => {
          console.log('\nMigration Result:', result.success ? 'SUCCESS' : 'FAILED');
          process.exit(result.success ? 0 : 1);
        })
        .catch(() => process.exit(1));
      break;
    case 'down':
      migrate({ ...options, rollback: true })
        .then((result) => {
          console.log('\nRollback Result:', result.success ? 'SUCCESS' : 'FAILED');
          process.exit(result.success ? 0 : 1);
        })
        .catch(() => process.exit(1));
      break;
    case 'redo':
      migrate({ ...options, redo: true })
        .then((result) => {
          console.log('\nRedo Result:', result.success ? 'SUCCESS' : 'FAILED');
          process.exit(result.success ? 0 : 1);
        })
        .catch(() => process.exit(1));
      break;
    case 'reset':
      migrate({ ...options, reset: true })
        .then((result) => {
          console.log('\nReset Result:', result.success ? 'SUCCESS' : 'FAILED');
          process.exit(result.success ? 0 : 1);
        })
        .catch(() => process.exit(1));
      break;
    case 'status':
      status({ verbose: options.verbose })
        .then(() => process.exit(0))
        .catch(() => process.exit(1));
      break;
    case 'create':
      if (args[1]) {
        create(args[1], { withRollback: options.generateRollback });
      } else {
        console.error('Please provide a migration name');
        process.exit(1);
      }
      break;
    default:
      console.log('Usage: node migrate.js [up|down|redo|reset|status|create <name>]');
      console.log('Options:');
      console.log('  --dry-run         Preview migrations without applying');
      console.log('  --force           Force operation (skip safety checks)');
      console.log('  --generate-rollback  Generate rollback scripts');
      console.log('  --verbose         Show detailed information');
      console.log('  --to <migration>  Target migration for rollback');
      process.exit(1);
  }
}

if (require.main === module) {
  main();
}

module.exports = {
  migrate,
  status,
  create,
  MigrationTracker,
  MigrationValidator,
  MigrationPerformance
};
