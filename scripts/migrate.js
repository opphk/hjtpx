const { Pool } = require('pg');
const fs = require('fs');
const path = require('path');
const crypto = require('crypto');
const os = require('os');
require('dotenv').config();

const DB_HOST = process.env.DB_HOST || 'localhost';
const DB_PORT = process.env.DB_PORT || 5432;
const DB_NAME = process.env.DB_NAME || 'hjtpx';
const DB_USER = process.env.DB_USER || 'postgres';
const DB_PASSWORD = process.env.DB_PASSWORD || 'postgres';

const MIGRATIONS_TABLE = 'migrations';
const MIGRATION_TRACKING_TABLE = 'migration_tracking';
const MIGRATIONS_DIR = path.join(__dirname, '../migrations');

const LOG_LEVELS = {
  DEBUG: 0,
  INFO: 1,
  WARN: 2,
  ERROR: 3
};

const currentLogLevel = process.env.MIGRATION_LOG_LEVEL
  ? LOG_LEVELS[process.env.MIGRATION_LOG_LEVEL.toUpperCase()] || LOG_LEVELS.INFO
  : LOG_LEVELS.INFO;

const MIGRATION_STATUS = {
  PENDING: 'pending',
  EXECUTED: 'executed',
  ROLLED_BACK: 'rolled_back',
  FAILED: 'failed'
};

function log(level, message, data = null) {
  if (level >= currentLogLevel) {
    const timestamp = new Date().toISOString();
    const levelName = Object.keys(LOG_LEVELS).find(k => LOG_LEVELS[k] === level) || 'INFO';
    const logMessage = `[${timestamp}] [${levelName}] ${message}`;
    console.log(logMessage);
    if (data) {
      console.log(JSON.stringify(data, null, 2));
    }
  }
}

function getMachineInfo() {
  return {
    hostname: os.hostname(),
    platform: os.platform(),
    arch: os.arch(),
    nodeVersion: process.version,
    pid: process.pid
  };
}

/**
 * Calculate checksum of a file for integrity verification
 */
function calculateChecksum(filePath) {
  const fileContent = fs.readFileSync(filePath, 'utf8');
  return crypto.createHash('sha256').update(fileContent).digest('hex');
}

/**
 * Parse migration file name to extract version and name
 */
function parseMigrationFileName(fileName) {
  const match = fileName.match(/^(\d+)_([^.]+)\.(up|down)\.sql$/);
  if (!match) return null;
  return {
    version: parseInt(match[1], 10),
    name: match[2],
    type: match[3]
  };
}

/**
 * Get all migration files sorted by version
 */
function getMigrationFiles() {
  const files = fs.readdirSync(MIGRATIONS_DIR)
    .filter(f => f.endsWith('.sql'))
    .map(f => ({
      fileName: f,
      parsed: parseMigrationFileName(f)
    }))
    .filter(f => f.parsed !== null)
    .sort((a, b) => a.parsed.version - b.parsed.version);
  
  return files;
}

/**
 * Group migration files by version
 */
function groupMigrationsByVersion(files) {
  const grouped = new Map();
  files.forEach(({ fileName, parsed }) => {
    if (!grouped.has(parsed.version)) {
      grouped.set(parsed.version, {
        version: parsed.version,
        name: parsed.name,
        up: null,
        down: null
      });
    }
    const migration = grouped.get(parsed.version);
    if (parsed.type === 'up') {
      migration.up = fileName;
    } else {
      migration.down = fileName;
    }
  });
  return Array.from(grouped.values()).sort((a, b) => a.version - b.version);
}

/**
 * Create migrations table if it doesn't exist
 */
async function createMigrationsTable(pool) {
  await pool.query(`
    CREATE TABLE IF NOT EXISTS ${MIGRATIONS_TABLE} (
      id SERIAL PRIMARY KEY,
      version INTEGER NOT NULL UNIQUE,
      name VARCHAR(255) NOT NULL,
      type VARCHAR(20) NOT NULL DEFAULT 'up',
      applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
      execution_time_ms INTEGER,
      status VARCHAR(20) DEFAULT 'success',
      checksum VARCHAR(64),
      error_message TEXT
    )
  `);
}

/**
 * Create migration_tracking table if it doesn't exist
 */
async function createMigrationTrackingTable(pool) {
  try {
    await pool.query(`
      CREATE TABLE IF NOT EXISTS ${MIGRATION_TRACKING_TABLE} (
        migration_id SERIAL PRIMARY KEY,
        migration_name VARCHAR(255) NOT NULL,
        migration_version INTEGER NOT NULL,
        migration_hash VARCHAR(64) NOT NULL,
        executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        rollback_executed_at TIMESTAMP,
        executed_by VARCHAR(255),
        status VARCHAR(20) NOT NULL DEFAULT 'pending',
        execution_time_ms INTEGER,
        error_message TEXT,
        machine_name VARCHAR(255),
        database_name VARCHAR(100)
      )
    `);

    await pool.query(`
      CREATE INDEX IF NOT EXISTS idx_migration_tracking_version
        ON ${MIGRATION_TRACKING_TABLE}(migration_version DESC)
    `);

    await pool.query(`
      CREATE INDEX IF NOT EXISTS idx_migration_tracking_status
        ON ${MIGRATION_TRACKING_TABLE}(status)
    `);

    log(LOG_LEVELS.DEBUG, 'Migration tracking table initialized');
  } catch (error) {
    log(LOG_LEVELS.WARN, 'Could not create migration tracking table, will retry later', {
      error: error.message
    });
  }
}

/**
 * Record migration in tracking table
 */
async function recordMigrationTracking(pool, {
  migrationName,
  migrationVersion,
  migrationHash,
  executedBy,
  status,
  executionTimeMs,
  errorMessage
}) {
  const machineInfo = getMachineInfo();

  try {
    await pool.query(`
      INSERT INTO ${MIGRATION_TRACKING_TABLE} (
        migration_name,
        migration_version,
        migration_hash,
        executed_at,
        executed_by,
        status,
        execution_time_ms,
        error_message,
        machine_name,
        database_name
      ) VALUES ($1, $2, $3, CURRENT_TIMESTAMP, $4, $5, $6, $7, $8, $9)
    `, [
      migrationName,
      migrationVersion,
      migrationHash,
      executedBy || os.userInfo().username,
      status,
      executionTimeMs,
      errorMessage,
      machineInfo.hostname,
      DB_NAME
    ]);
  } catch (error) {
    log(LOG_LEVELS.WARN, 'Failed to record migration tracking', {
      error: error.message,
      migrationName,
      migrationVersion
    });
  }
}

/**
 * Update migration tracking status
 */
async function updateMigrationTracking(pool, migrationVersion, status, errorMessage = null) {
  try {
    const updates = [`status = '${status}'`];

    if (status === MIGRATION_STATUS.ROLLED_BACK) {
      updates.push('rollback_executed_at = CURRENT_TIMESTAMP');
    }

    if (errorMessage) {
      updates.push(`error_message = '${errorMessage.replace(/'/g, "''")}'`);
    }

    await pool.query(`
      UPDATE ${MIGRATION_TRACKING_TABLE}
      SET ${updates.join(', ')}
      WHERE migration_version = $1 AND status = '${MIGRATION_STATUS.EXECUTED}'
    `, [migrationVersion]);
  } catch (error) {
    log(LOG_LEVELS.WARN, 'Failed to update migration tracking', {
      error: error.message,
      migrationVersion,
      status
    });
  }
}

/**
 * Get migration tracking records
 */
async function getMigrationTracking(pool, migrationVersion = null) {
  let query = `SELECT * FROM ${MIGRATION_TRACKING_TABLE}`;
  const params = [];

  if (migrationVersion !== null) {
    query += ' WHERE migration_version = $1';
    params.push(migrationVersion);
  }

  query += ' ORDER BY migration_version ASC';

  const result = await pool.query(query, params);
  return result.rows;
}

/**
 * Check database health
 */
async function checkDatabaseHealth(pool) {
  try {
    const result = await pool.query(`
      SELECT
        current_setting('server_version_num') as version_num,
        current_setting('server_version') as version,
        (SELECT count(*) FROM pg_stat_activity WHERE datname = current_database()) as active_connections,
        (SELECT sum(xact_commit) FROM pg_stat_database WHERE datname = current_database()) as total_commits,
        (SELECT sum(xact_rollback) FROM pg_stat_database WHERE datname = current_database()) as total_rollbacks
    `);

    const stats = result.rows[0];
    log(LOG_LEVELS.DEBUG, 'Database health check passed', stats);

    return {
      healthy: true,
      version: stats.version,
      activeConnections: parseInt(stats.active_connections),
      totalCommits: parseInt(stats.total_commits),
      totalRollbacks: parseInt(stats.total_rollbacks)
    };
  } catch (error) {
    log(LOG_LEVELS.ERROR, 'Database health check failed', { error: error.message });
    return {
      healthy: false,
      error: error.message
    };
  }
}

/**
 * Get migration statistics
 */
async function getMigrationStats(pool) {
  const result = await pool.query(`
    SELECT
      COUNT(*) FILTER (WHERE type = 'up') as total_migrations,
      COUNT(*) FILTER (WHERE type = 'up' AND status = 'success') as successful_migrations,
      COUNT(*) FILTER (WHERE type = 'up' AND status = 'failed') as failed_migrations,
      COUNT(*) FILTER (WHERE type = 'down') as rollbacks,
      MAX(applied_at) FILTER (WHERE status = 'success') as last_successful_migration,
      MIN(applied_at) FILTER (WHERE status = 'success') as first_migration,
      AVG(execution_time_ms) FILTER (WHERE status = 'success') as avg_execution_time
    FROM ${MIGRATIONS_TABLE}
  `);

  return result.rows[0];
}

/**
 * Get all applied migrations from the database
 */
async function getAppliedMigrations(pool) {
  const result = await pool.query(
    `SELECT version, name, type, applied_at, status, checksum FROM ${MIGRATIONS_TABLE} ORDER BY version ASC`
  );
  return result.rows;
}

/**
 * Get current database version
 */
async function getCurrentVersion(pool) {
  const result = await pool.query(
    `SELECT version FROM ${MIGRATIONS_TABLE} WHERE type = 'up' AND status = 'success' ORDER BY version DESC LIMIT 1`
  );
  return result.rows.length > 0 ? result.rows[0].version : 0;
}

/**
 * Record a migration in the database
 */
async function recordMigration(pool, { version, name, type, executionTime, status, checksum, errorMessage }) {
  await pool.query(
    `INSERT INTO ${MIGRATIONS_TABLE} (version, name, type, execution_time_ms, status, checksum, error_message)
     VALUES ($1, $2, $3, $4, $5, $6, $7)
     ON CONFLICT (version) DO UPDATE SET
       type = EXCLUDED.type,
       applied_at = CURRENT_TIMESTAMP,
       execution_time_ms = EXCLUDED.execution_time_ms,
       status = EXCLUDED.status,
       checksum = EXCLUDED.checksum,
       error_message = EXCLUDED.error_message`,
    [version, name, type, executionTime, status, checksum, errorMessage]
  );
}

/**
 * Execute a SQL file
 */
async function executeSqlFile(pool, filePath) {
  const sql = fs.readFileSync(filePath, 'utf8');
  
  const client = await pool.connect();
  try {
    await client.query('BEGIN');
    const startTime = Date.now();
    await client.query(sql);
    const executionTime = Date.now() - startTime;
    await client.query('COMMIT');
    return { success: true, executionTime };
  } catch (error) {
    await client.query('ROLLBACK');
    throw error;
  } finally {
    client.release();
  }
}

/**
 * Apply pending migrations with enhanced tracking
 */
async function migrateUp(pool, targetVersion = null) {
  const machineInfo = getMachineInfo();
  const startTime = Date.now();

  log(LOG_LEVELS.INFO, 'Starting migration process', {
    targetVersion,
    machine: machineInfo
  });

  const migrations = groupMigrationsByVersion(getMigrationFiles());
  const appliedMigrations = await getAppliedMigrations(pool);
  const currentVersion = await getCurrentVersion(pool);

  const appliedVersions = new Set(appliedMigrations.filter(m => m.type === 'up' && m.status === 'success').map(m => m.version));

  const pendingMigrations = migrations.filter(m =>
    !appliedVersions.has(m.version) &&
    (targetVersion === null || m.version <= targetVersion) &&
    m.version > currentVersion
  );

  if (pendingMigrations.length === 0) {
    log(LOG_LEVELS.INFO, 'No pending migrations found');
    console.log('No pending migrations');
    return;
  }

  log(LOG_LEVELS.INFO, `Found ${pendingMigrations.length} pending migration(s)`, {
    pendingVersions: pendingMigrations.map(m => m.version)
  });

  console.log(`Found ${pendingMigrations.length} pending migration(s)`);

  const results = {
    successful: [],
    failed: []
  };

  for (const migration of pendingMigrations) {
    if (!migration.up) {
      log(LOG_LEVELS.WARN, `Skipping migration ${migration.version} - no up script found`);
      console.warn(`Skipping migration ${migration.version} - no up script found`);
      continue;
    }

    const migrationStartTime = Date.now();
    log(LOG_LEVELS.INFO, `Applying migration ${migration.version}: ${migration.name}`);

    console.log(`Applying migration ${migration.version}: ${migration.name}`);
    const filePath = path.join(MIGRATIONS_DIR, migration.up);
    const checksum = calculateChecksum(filePath);

    try {
      const result = await executeSqlFile(pool, filePath);
      const executionTime = Date.now() - migrationStartTime;

      await recordMigration(pool, {
        version: migration.version,
        name: migration.name,
        type: 'up',
        executionTime: result.executionTime,
        status: 'success',
        checksum,
        errorMessage: null
      });

      await recordMigrationTracking(pool, {
        migrationName: migration.name,
        migrationVersion: migration.version,
        migrationHash: checksum,
        executedBy: os.userInfo().username,
        status: MIGRATION_STATUS.EXECUTED,
        executionTimeMs: executionTime,
        errorMessage: null
      });

      results.successful.push({
        version: migration.version,
        name: migration.name,
        executionTime
      });

      log(LOG_LEVELS.INFO, `Migration ${migration.version} applied successfully`, {
        executionTimeMs: executionTime,
        checksum
      });

      console.log(`✓ Migration ${migration.version} applied successfully (${executionTime}ms)`);
    } catch (error) {
      const executionTime = Date.now() - migrationStartTime;

      await recordMigration(pool, {
        version: migration.version,
        name: migration.name,
        type: 'up',
        executionTime: 0,
        status: 'failed',
        checksum,
        errorMessage: error.message
      });

      await recordMigrationTracking(pool, {
        migrationName: migration.name,
        migrationVersion: migration.version,
        migrationHash: checksum,
        executedBy: os.userInfo().username,
        status: MIGRATION_STATUS.FAILED,
        executionTimeMs: executionTime,
        errorMessage: error.message
      });

      results.failed.push({
        version: migration.version,
        name: migration.name,
        error: error.message
      });

      log(LOG_LEVELS.ERROR, `Migration ${migration.version} failed`, {
        error: error.message,
        executionTimeMs: executionTime
      });

      console.error(`✗ Migration ${migration.version} failed:`, error.message);

      const totalTime = Date.now() - startTime;
      log(LOG_LEVELS.ERROR, 'Migration process failed', {
        successfulCount: results.successful.length,
        failedCount: results.failed.length,
        totalTimeMs: totalTime
      });

      throw error;
    }
  }

  const totalTime = Date.now() - startTime;
  log(LOG_LEVELS.INFO, 'Migration process completed', {
    successfulCount: results.successful.length,
    failedCount: results.failed.length,
    totalTimeMs: totalTime,
    results
  });
}

/**
 * Rollback migrations with enhanced tracking
 */
async function migrateDown(pool, targetVersion = null, steps = 1) {
  const machineInfo = getMachineInfo();
  const startTime = Date.now();

  log(LOG_LEVELS.INFO, 'Starting rollback process', {
    targetVersion,
    steps,
    machine: machineInfo
  });

  const appliedMigrations = await getAppliedMigrations(pool);
  const successfulUpMigrations = appliedMigrations
    .filter(m => m.type === 'up' && m.status === 'success')
    .sort((a, b) => b.version - a.version);

  if (successfulUpMigrations.length === 0) {
    log(LOG_LEVELS.INFO, 'No migrations to rollback');
    console.log('No migrations to rollback');
    return;
  }

  const migrations = groupMigrationsByVersion(getMigrationFiles());
  const migrationsMap = new Map(migrations.map(m => [m.version, m]));

  let migrationsToRollback;
  if (targetVersion !== null) {
    migrationsToRollback = successfulUpMigrations.filter(m => m.version > targetVersion);
  } else {
    migrationsToRollback = successfulUpMigrations.slice(0, steps);
  }

  if (migrationsToRollback.length === 0) {
    log(LOG_LEVELS.INFO, 'No migrations to rollback based on criteria');
    console.log('No migrations to rollback');
    return;
  }

  log(LOG_LEVELS.INFO, `Rolling back ${migrationsToRollback.length} migration(s)`, {
    versions: migrationsToRollback.map(m => m.version)
  });

  console.log(`Rolling back ${migrationsToRollback.length} migration(s)`);

  const results = {
    successful: [],
    failed: []
  };

  for (const appliedMigration of migrationsToRollback) {
    const migration = migrationsMap.get(appliedMigration.version);
    if (!migration || !migration.down) {
      log(LOG_LEVELS.WARN, `Skipping rollback of ${appliedMigration.version} - no down script found`);
      console.warn(`Skipping rollback of ${appliedMigration.version} - no down script found`);
      continue;
    }

    const migrationStartTime = Date.now();
    log(LOG_LEVELS.INFO, `Rolling back migration ${migration.version}: ${migration.name}`);

    console.log(`Rolling back migration ${migration.version}: ${migration.name}`);
    const filePath = path.join(MIGRATIONS_DIR, migration.down);
    const checksum = calculateChecksum(filePath);

    try {
      const result = await executeSqlFile(pool, filePath);
      const executionTime = Date.now() - migrationStartTime;

      await recordMigration(pool, {
        version: migration.version,
        name: migration.name,
        type: 'down',
        executionTime: result.executionTime,
        status: 'success',
        checksum,
        errorMessage: null
      });

      await updateMigrationTracking(pool, migration.version, MIGRATION_STATUS.ROLLED_BACK);

      await recordMigration(pool, {
        version: migration.version,
        name: migration.name,
        type: 'up',
        executionTime: 0,
        status: 'rolled_back',
        checksum,
        errorMessage: null
      });

      results.successful.push({
        version: migration.version,
        name: migration.name,
        executionTime
      });

      log(LOG_LEVELS.INFO, `Rollback of ${migration.version} completed`, {
        executionTimeMs: executionTime
      });

      console.log(`✓ Rollback of ${migration.version} completed (${executionTime}ms)`);
    } catch (error) {
      const executionTime = Date.now() - migrationStartTime;

      await recordMigration(pool, {
        version: migration.version,
        name: migration.name,
        type: 'down',
        executionTime: 0,
        status: 'failed',
        checksum,
        errorMessage: error.message
      });

      await recordMigrationTracking(pool, {
        migrationName: migration.name,
        migrationVersion: migration.version,
        migrationHash: checksum,
        executedBy: os.userInfo().username,
        status: MIGRATION_STATUS.FAILED,
        executionTimeMs: executionTime,
        errorMessage: `Rollback failed: ${error.message}`
      });

      results.failed.push({
        version: migration.version,
        name: migration.name,
        error: error.message
      });

      log(LOG_LEVELS.ERROR, `Rollback of ${migration.version} failed`, {
        error: error.message,
        executionTimeMs: executionTime
      });

      console.error(`✗ Rollback of ${migration.version} failed:`, error.message);

      const totalTime = Date.now() - startTime;
      log(LOG_LEVELS.ERROR, 'Rollback process failed', {
        successfulCount: results.successful.length,
        failedCount: results.failed.length,
        totalTimeMs: totalTime
      });

      throw error;
    }
  }

  const totalTime = Date.now() - startTime;
  log(LOG_LEVELS.INFO, 'Rollback process completed', {
    successfulCount: results.successful.length,
    failedCount: results.failed.length,
    totalTimeMs: totalTime,
    results
  });
}

/**
 * Show migration status with enhanced reporting
 */
async function showStatus(pool) {
  const migrations = groupMigrationsByVersion(getMigrationFiles());
  const appliedMigrations = await getAppliedMigrations(pool);
  const appliedMap = new Map(appliedMigrations.map(m => [m.version, m]));
  const currentVersion = await getCurrentVersion(pool);
  const health = await checkDatabaseHealth(pool);
  const stats = await getMigrationStats(pool);

  let trackingStats = null;
  try {
    const trackingResult = await pool.query(`
      SELECT
        COUNT(*) FILTER (WHERE status = 'executed') as executed,
        COUNT(*) FILTER (WHERE status = 'rolled_back') as rolled_back,
        COUNT(*) FILTER (WHERE status = 'failed') as failed,
        COUNT(*) FILTER (WHERE status = 'pending') as pending
      FROM ${MIGRATION_TRACKING_TABLE}
    `);
    trackingStats = trackingResult.rows[0];
  } catch (error) {
    log(LOG_LEVELS.WARN, 'Could not fetch tracking stats', { error: error.message });
  }

  console.log('\n╔════════════════════════════════════════════════════════════════╗');
  console.log('║              Database Migration Status Report                  ║');
  console.log('╚════════════════════════════════════════════════════════════════╝\n');

  console.log('┌─────────────────────────────────────────────────────────────────┐');
  console.log('│ Database Information                                            │');
  console.log('├─────────────────────────────────────────────────────────────────┤');
  console.log(`│ Host: ${DB_HOST}:${DB_PORT}`.padEnd(64) + '│');
  console.log(`│ Database: ${DB_NAME}`.padEnd(64) + '│');
  console.log(`│ Version: ${health.healthy ? health.version : 'N/A'}`.padEnd(64) + '│');
  console.log(`│ Active Connections: ${health.healthy ? health.activeConnections : 'N/A'}`.padEnd(64) + '│');
  console.log('└─────────────────────────────────────────────────────────────────┘\n');

  console.log('┌─────────────────────────────────────────────────────────────────┐');
  console.log('│ Migration Statistics                                            │');
  console.log('├─────────────────────────────────────────────────────────────────┤');
  console.log(`│ Total Migrations: ${stats.total_migrations || 0}`.padEnd(64) + '│');
  console.log(`│ Successful: ${stats.successful_migrations || 0}`.padEnd(64) + '│');
  console.log(`│ Failed: ${stats.failed_migrations || 0}`.padEnd(64) + '│');
  console.log(`│ Rollbacks: ${stats.rollbacks || 0}`.padEnd(64) + '│');
  console.log(`│ Avg Execution Time: ${stats.avg_execution_time ? Math.round(stats.avg_execution_time) + 'ms' : 'N/A'}`.padEnd(64) + '│');
  console.log(`│ Current Version: ${currentVersion}`.padEnd(64) + '│');
  console.log('└─────────────────────────────────────────────────────────────────┘\n');

  if (trackingStats) {
    console.log('┌─────────────────────────────────────────────────────────────────┐');
    console.log('│ Migration Tracking Statistics                                   │');
    console.log('├─────────────────────────────────────────────────────────────────┤');
    console.log(`│ Executed: ${trackingStats.executed || 0}`.padEnd(64) + '│');
    console.log(`│ Rolled Back: ${trackingStats.rolled_back || 0}`.padEnd(64) + '│');
    console.log(`│ Failed: ${trackingStats.failed || 0}`.padEnd(64) + '│');
    console.log(`│ Pending: ${trackingStats.pending || 0}`.padEnd(64) + '│');
    console.log('└─────────────────────────────────────────────────────────────────┘\n');
  }

  console.log('┌─────────────────────────────────────────────────────────────────┐');
  console.log('│ Migration History                                               │');
  console.log('├─────┬────────────────────────────────────────┬────────┬──────────┤');
  console.log('│ Ver │ Name                                   │ Status │ Time     │');
  console.log('├─────┼────────────────────────────────────────┼────────┼──────────┤');

  migrations.forEach(migration => {
    const applied = appliedMap.get(migration.version);
    let status = 'pending';
    let time = '';
    let statusSymbol = '○';

    if (applied) {
      if (applied.type === 'up' && applied.status === 'success') {
        status = 'applied';
        time = applied.execution_time_ms ? `${applied.execution_time_ms}ms` : '';
        statusSymbol = '✓';
      } else if (applied.type === 'down' && applied.status === 'success') {
        status = 'rolled back';
        statusSymbol = '↺';
      } else if (applied.status === 'failed') {
        status = 'failed';
        statusSymbol = '✗';
      }
    }

    const version = migration.version.toString().padStart(3);
    const name = migration.name.substring(0, 40).padEnd(40);
    const statusDisplay = status.substring(0, 8).padEnd(8);
    const timeDisplay = time.substring(0, 10).padEnd(10);

    console.log(`│ ${version} │ ${name} │ ${statusDisplay} │ ${timeDisplay} │`);
  });

  console.log('└─────┴────────────────────────────────────────┴────────┴──────────┘\n');

  console.log('Legend: ○ = Pending | ✓ = Applied | ↺ = Rolled Back | ✗ = Failed\n');

  log(LOG_LEVELS.DEBUG, 'Status report generated', {
    currentVersion,
    totalMigrations: migrations.length,
    stats,
    trackingStats
  });
}

/**
 * Create a new migration
 */
async function createMigration(name) {
  const migrations = groupMigrationsByVersion(getMigrationFiles());
  const nextVersion = migrations.length > 0 ? migrations[migrations.length - 1].version + 1 : 1;
  const timestamp = new Date().toISOString().split('T')[0];
  
  const upFileName = `${nextVersion.toString().padStart(3, '0')}_${name}.up.sql`;
  const downFileName = `${nextVersion.toString().padStart(3, '0')}_${name}.down.sql`;
  
  const upContent = `-- Migration: ${name}
-- Created: ${timestamp}
-- Description: [Add description here]

-- Write your migration SQL here

`;
  
  const downContent = `-- Rollback: ${name}
-- Description: [Add rollback description here]

-- Write your rollback SQL here

`;
  
  fs.writeFileSync(path.join(MIGRATIONS_DIR, upFileName), upContent);
  fs.writeFileSync(path.join(MIGRATIONS_DIR, downFileName), downContent);
  
  console.log(`Created migration ${nextVersion}: ${name}`);
  console.log(`  Up: ${upFileName}`);
  console.log(`  Down: ${downFileName}`);
}

/**
 * Main function with enhanced error handling and logging
 */
async function main() {
  const startTime = Date.now();
  const machineInfo = getMachineInfo();

  const args = process.argv.slice(2);
  const command = args[0] || 'status';

  if (command === '--help' || command === '-h' || args.length === 0) {
    console.log(`
Usage: node migrate.js <command> [options]

Commands:
  up [version]          Apply pending migrations (optionally up to a specific version)
  down [steps]          Rollback last [steps] migrations (default: 1)
  down --to <version>   Rollback down to a specific version
  status                Show migration status (default)
  create <name>         Create a new migration
  health                Check database health
  stats                 Show migration statistics
  tracking              Show migration tracking history

Environment Variables:
  DB_HOST               Database host (default: localhost)
  DB_PORT               Database port (default: 5432)
  DB_NAME               Database name (default: hjtpx)
  DB_USER               Database user (default: postgres)
  DB_PASSWORD           Database password (default: postgres)
  MIGRATION_LOG_LEVEL   Log level: DEBUG, INFO, WARN, ERROR (default: INFO)

Examples:
  node migrate.js up
  node migrate.js up 5
  node migrate.js down
  node migrate.js down 3
  node migrate.js down --to 2
  node migrate.js status
  node migrate.js create add_users_table
  node migrate.js health
  node migrate.js stats
  node migrate.js tracking
  MIGRATION_LOG_LEVEL=DEBUG node migrate.js status
`);
    process.exit(0);
  }

  log(LOG_LEVELS.INFO, 'Migration process starting', {
    machine: machineInfo,
    arguments: args,
    nodeVersion: process.version,
    cwd: process.cwd()
  });

  const pool = new Pool({
    host: DB_HOST,
    port: DB_PORT,
    database: DB_NAME,
    user: DB_USER,
    password: DB_PASSWORD,
  });

  try {
    log(LOG_LEVELS.INFO, 'Connecting to database', {
      host: DB_HOST,
      port: DB_PORT,
      database: DB_NAME
    });

    await createMigrationsTable(pool);
    log(LOG_LEVELS.DEBUG, 'Migrations table initialized');

    await createMigrationTrackingTable(pool);
    log(LOG_LEVELS.DEBUG, 'Migration tracking table initialized');

    const dbHealth = await checkDatabaseHealth(pool);
    if (!dbHealth.healthy) {
      log(LOG_LEVELS.WARN, 'Database health check failed, proceeding anyway', {
        error: dbHealth.error
      });
    } else {
      log(LOG_LEVELS.DEBUG, 'Database health check passed', dbHealth);
    }

    log(LOG_LEVELS.INFO, `Executing command: ${command}`, { args });

    switch (command) {
      case 'up':
        const targetVersionUp = args[1] ? parseInt(args[1], 10) : null;
        await migrateUp(pool, targetVersionUp);
        break;

      case 'down':
        if (args[1] === '--to' && args[2]) {
          await migrateDown(pool, parseInt(args[2], 10));
        } else if (args[1]) {
          await migrateDown(pool, null, parseInt(args[1], 10));
        } else {
          await migrateDown(pool);
        }
        break;

      case 'status':
        await showStatus(pool);
        break;

      case 'create':
        if (args[1]) {
          await createMigration(args[1]);
        } else {
          console.error('Please provide a migration name');
          process.exit(1);
        }
        break;

      case 'health':
        const health = await checkDatabaseHealth(pool);
        console.log('\n=== Database Health Check ===');
        console.log(JSON.stringify(health, null, 2));
        break;

      case 'stats':
        const stats = await getMigrationStats(pool);
        console.log('\n=== Migration Statistics ===');
        console.log(JSON.stringify(stats, null, 2));
        break;

      case 'tracking':
        const tracking = await getMigrationTracking(pool);
        console.log('\n=== Migration Tracking History ===');
        if (tracking.length === 0) {
          console.log('No migration tracking records found.');
        } else {
          console.log(`Found ${tracking.length} tracking record(s):\n`);
          console.table(tracking.map(t => ({
            Version: t.migration_version,
            Name: t.migration_name,
            Status: t.status,
            Executed: t.executed_at ? new Date(t.executed_at).toLocaleString() : 'N/A',
            'Rollback At': t.rollback_executed_at ? new Date(t.rollback_executed_at).toLocaleString() : 'N/A',
            'By': t.executed_by,
            'Time (ms)': t.execution_time_ms
          })));
        }
        break;

      default:
        console.error(`Unknown command: ${command}`);
        console.log('\nRun "node migrate.js --help" for usage information.');
        process.exit(1);
    }

    const executionTime = Date.now() - startTime;
    log(LOG_LEVELS.INFO, 'Migration process completed', {
      command,
      executionTimeMs: executionTime,
      success: true
    });

  } catch (error) {
    const executionTime = Date.now() - startTime;
    log(LOG_LEVELS.ERROR, 'Migration process failed', {
      command: process.argv.slice(2),
      executionTimeMs: executionTime,
      error: error.message,
      stack: error.stack
    });

    console.error('\n❌ Migration failed:', error.message);
    if (process.env.MIGRATION_LOG_LEVEL === 'DEBUG') {
      console.error(error.stack);
    }
    process.exit(1);
  } finally {
    await pool.end();
    log(LOG_LEVELS.DEBUG, 'Database connection closed');
  }
}

if (require.main === module) {
  main().catch(error => {
    console.error('Migration error:', error);
    process.exit(1);
  });
}

module.exports = {
  migrateUp,
  migrateDown,
  showStatus,
  createMigration,
  getMigrationFiles,
  getCurrentVersion,
  checkDatabaseHealth,
  getMigrationStats,
  LOG_LEVELS
};
