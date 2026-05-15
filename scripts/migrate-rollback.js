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

function calculateChecksum(filePath) {
  const fileContent = fs.readFileSync(filePath, 'utf8');
  return crypto.createHash('sha256').update(fileContent).digest('hex');
}

function parseMigrationFileName(fileName) {
  const match = fileName.match(/^(\d+)_([^.]+)\.(up|down)\.sql$/);
  if (!match) return null;
  return {
    version: parseInt(match[1], 10),
    name: match[2],
    type: match[3]
  };
}

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

async function getAppliedMigrations(pool) {
  const result = await pool.query(
    `SELECT version, name, type, applied_at, status, checksum FROM ${MIGRATIONS_TABLE} ORDER BY version ASC`
  );
  return result.rows;
}

async function getCurrentVersion(pool) {
  const result = await pool.query(
    `SELECT version FROM ${MIGRATIONS_TABLE} WHERE type = 'up' AND status = 'success' ORDER BY version DESC LIMIT 1`
  );
  return result.rows.length > 0 ? result.rows[0].version : 0;
}

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

async function validateRollback(pool, migration) {
  const warnings = [];

  try {
    const result = await pool.query(`
      SELECT 
        tc.table_name,
        tc.constraint_name,
        kcu.column_name
      FROM information_schema.table_constraints AS tc
      JOIN information_schema.key_column_usage AS kcu
        ON tc.constraint_name = kcu.constraint_name
      WHERE tc.constraint_type = 'FOREIGN KEY'
        AND tc.table_schema = 'public'
    `);

    for (const constraint of result.rows) {
      if (constraint.table_name.includes(migration.name.replace(/_/g, '_'))) {
        warnings.push(`Foreign key constraint found: ${constraint.constraint_name} on ${constraint.table_name}`);
      }
    }
  } catch (error) {
    log(LOG_LEVELS.WARN, 'Could not check foreign key constraints', { error: error.message });
  }

  try {
    const result = await pool.query(`
      SELECT pid, usename, application_name, state, query_start
      FROM pg_stat_activity
      WHERE datname = current_database()
        AND pid <> pg_backend_pid()
        AND state != 'idle'
    `);

    if (result.rows.length > 0) {
      warnings.push(`Found ${result.rows.length} active database connection(s)`);
    }
  } catch (error) {
    log(LOG_LEVELS.WARN, 'Could not check active connections', { error: error.message });
  }

  return warnings;
}

async function performRollback(pool, options = {}) {
  const {
    steps = 1,
    targetVersion = null,
    force = false,
    dryRun = false
  } = options;

  const machineInfo = getMachineInfo();
  const startTime = Date.now();

  log(LOG_LEVELS.INFO, 'Starting rollback process', {
    steps,
    targetVersion,
    force,
    dryRun,
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
    console.log('No migrations to rollback based on specified criteria');
    return;
  }

  log(LOG_LEVELS.INFO, `Planning to rollback ${migrationsToRollback.length} migration(s)`, {
    versions: migrationsToRollback.map(m => m.version)
  });

  console.log(`\n📋 Rollback Plan:`);
  console.log(`   Target: ${migrationsToRollback.length} migration(s)`);
  if (targetVersion !== null) {
    console.log(`   Target Version: ${targetVersion}`);
  } else {
    console.log(`   Steps: ${steps}`);
  }

  for (const migration of migrationsToRollback) {
    const migrationInfo = migrationsMap.get(migration.version);
    console.log(`   - Version ${migration.version}: ${migrationInfo ? migrationInfo.name : 'Unknown'}`);

    const warnings = await validateRollback(pool, migration);
    if (warnings.length > 0) {
      console.log(`     ⚠️  Warnings:`);
      warnings.forEach(w => console.log(`       - ${w}`));
    }
  }

  if (dryRun) {
    console.log('\n🔍 Dry run mode - no changes were made');
    log(LOG_LEVELS.INFO, 'Dry run completed, no changes made');
    return;
  }

  if (!force) {
    const readline = require('readline');
    const rl = readline.createInterface({
      input: process.stdin,
      output: process.stdout
    });

    const answer = await new Promise(resolve => {
      rl.question('\n❓ Do you want to proceed with the rollback? (yes/no): ', resolve);
    });
    rl.close();

    if (answer.toLowerCase() !== 'yes' && answer.toLowerCase() !== 'y') {
      console.log('Rollback cancelled by user');
      log(LOG_LEVELS.INFO, 'Rollback cancelled by user');
      return;
    }
  }

  console.log(`\n🔄 Rolling back ${migrationsToRollback.length} migration(s)...\n`);

  const results = {
    successful: [],
    failed: []
  };

  for (const appliedMigration of migrationsToRollback) {
    const migration = migrationsMap.get(appliedMigration.version);
    if (!migration || !migration.down) {
      log(LOG_LEVELS.WARN, `Skipping rollback of ${appliedMigration.version} - no down script found`);
      console.warn(`⚠️  Skipping rollback of ${appliedMigration.version} - no down script found`);
      continue;
    }

    const migrationStartTime = Date.now();
    log(LOG_LEVELS.INFO, `Rolling back migration ${migration.version}: ${migration.name}`);

    console.log(`\n🔄 Rolling back migration ${migration.version}: ${migration.name}`);
    
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

      try {
        await updateMigrationTracking(pool, migration.version, MIGRATION_STATUS.ROLLED_BACK);
      } catch (trackingError) {
        log(LOG_LEVELS.WARN, 'Failed to update tracking table', { error: trackingError.message });
      }

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

      console.log(`✅ Rollback of ${migration.version} completed (${executionTime}ms)`);
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

      results.failed.push({
        version: migration.version,
        name: migration.name,
        error: error.message
      });

      log(LOG_LEVELS.ERROR, `Rollback of ${migration.version} failed`, {
        error: error.message,
        executionTimeMs: executionTime
      });

      console.error(`❌ Rollback of ${migration.version} failed: ${error.message}`);

      const totalTime = Date.now() - startTime;
      log(LOG_LEVELS.ERROR, 'Rollback process failed', {
        successfulCount: results.successful.length,
        failedCount: results.failed.length,
        totalTimeMs: totalTime
      });

      console.log(`\n⚠️  Rollback partially failed`);
      console.log(`   Successful: ${results.successful.length}`);
      console.log(`   Failed: ${results.failed.length}`);
      
      return;
    }
  }

  const totalTime = Date.now() - startTime;
  log(LOG_LEVELS.INFO, 'Rollback process completed', {
    successfulCount: results.successful.length,
    failedCount: results.failed.length,
    totalTimeMs: totalTime,
    results
  });

  console.log(`\n✅ Rollback completed`);
  console.log(`   Successful: ${results.successful.length}`);
  console.log(`   Failed: ${results.failed.length}`);
  console.log(`   Total time: ${totalTime}ms`);
}

async function showRollbackPlan(pool, options = {}) {
  const { steps = 1, targetVersion = null } = options;

  const appliedMigrations = await getAppliedMigrations(pool);
  const successfulUpMigrations = appliedMigrations
    .filter(m => m.type === 'up' && m.status === 'success')
    .sort((a, b) => b.version - a.version);

  if (successfulUpMigrations.length === 0) {
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
    console.log('No migrations would be rolled back with the specified criteria');
    return;
  }

  console.log('\n📋 Rollback Plan:\n');

  for (const migration of migrationsToRollback) {
    const migrationInfo = migrationsMap.get(migration.version);
    console.log(`Version ${migration.version}: ${migrationInfo ? migrationInfo.name : 'Unknown'}`);
    console.log(`  Applied at: ${new Date(migration.applied_at).toLocaleString()}`);
    if (migrationInfo && migrationInfo.down) {
      console.log(`  Down script: ${migrationInfo.down}`);
    } else {
      console.log(`  ⚠️  No down script found`);
    }
    console.log('');
  }
}

async function main() {
  const startTime = Date.now();
  const machineInfo = getMachineInfo();

  const args = process.argv.slice(2);
  const command = args[0] || 'status';

  if (command === '--help' || command === '-h' || args.length === 0) {
    console.log(`
Usage: node migrate-rollback.js <command> [options]

Commands:
  rollback [steps]        Rollback last [steps] migrations (default: 1)
  rollback --to <ver>    Rollback down to a specific version
  rollback --dry-run     Show what would be rolled back without making changes
  rollback --force       Skip confirmation prompt
  plan [steps]           Show rollback plan without executing
  plan --to <ver>        Show rollback plan to specific version
  status                 Show current migration status

Options:
  --force, -f            Skip confirmation prompt
  --dry-run              Preview rollback without executing
  --to <version>         Rollback to specific version

Examples:
  node migrate-rollback.js rollback
  node migrate-rollback.js rollback 3
  node migrate-rollback.js rollback --to 5
  node migrate-rollback.js rollback --dry-run
  node migrate-rollback.js rollback --force
  node migrate-rollback.js plan
  node migrate-rollback.js status

Environment Variables:
  DB_HOST               Database host (default: localhost)
  DB_PORT               Database port (default: 5432)
  DB_NAME               Database name (default: hjtpx)
  DB_USER               Database user (default: postgres)
  DB_PASSWORD           Database password (default: postgres)
  MIGRATION_LOG_LEVEL   Log level: DEBUG, INFO, WARN, ERROR (default: INFO)
`);
    process.exit(0);
  }

  log(LOG_LEVELS.INFO, 'Migration rollback script starting', {
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

    log(LOG_LEVELS.INFO, `Executing command: ${command}`, { args });

    switch (command) {
      case 'rollback':
        const steps = args.includes('--to') 
          ? null 
          : parseInt(args[1] || '1', 10);
        const targetVersion = args.includes('--to') 
          ? parseInt(args[args.indexOf('--to') + 1], 10) 
          : null;
        const force = args.includes('--force') || args.includes('-f');
        const dryRun = args.includes('--dry-run');

        await performRollback(pool, {
          steps,
          targetVersion,
          force,
          dryRun
        });
        break;

      case 'plan':
        const planSteps = args.includes('--to')
          ? null
          : parseInt(args[1] || '1', 10);
        const planTargetVersion = args.includes('--to')
          ? parseInt(args[args.indexOf('--to') + 1], 10)
          : null;

        await showRollbackPlan(pool, {
          steps: planSteps,
          targetVersion: planTargetVersion
        });
        break;

      case 'status':
        const currentVersion = await getCurrentVersion(pool);
        const appliedMigrations = await getAppliedMigrations(pool);
        const successfulMigrations = appliedMigrations.filter(
          m => m.type === 'up' && m.status === 'success'
        );

        console.log('\n📊 Rollback Status:');
        console.log(`   Current Version: ${currentVersion}`);
        console.log(`   Applied Migrations: ${successfulMigrations.length}`);
        console.log('\n   Applied Migrations:');
        
        successfulMigrations.sort((a, b) => b.version - a.version).forEach(m => {
          console.log(`   - Version ${m.version}: ${m.name}`);
        });
        break;

      default:
        console.error(`Unknown command: ${command}`);
        console.log('\nRun "node migrate-rollback.js --help" for usage information.');
        process.exit(1);
    }

    const executionTime = Date.now() - startTime;
    log(LOG_LEVELS.INFO, 'Migration rollback script completed', {
      command,
      executionTimeMs: executionTime,
      success: true
    });

  } catch (error) {
    const executionTime = Date.now() - startTime;
    log(LOG_LEVELS.ERROR, 'Migration rollback script failed', {
      command: process.argv.slice(2),
      executionTimeMs: executionTime,
      error: error.message,
      stack: error.stack
    });

    console.error('\n❌ Rollback failed:', error.message);
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
    console.error('Rollback error:', error);
    process.exit(1);
  });
}

module.exports = {
  performRollback,
  showRollbackPlan,
  validateRollback,
  getAppliedMigrations,
  getCurrentVersion,
  LOG_LEVELS
};
