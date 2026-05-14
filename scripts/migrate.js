const { Pool } = require('pg');
const fs = require('fs');
const path = require('path');
require('dotenv').config();

const DB_HOST = process.env.DB_HOST || 'localhost';
const DB_PORT = process.env.DB_PORT || 5432;
const DB_NAME = process.env.DB_NAME || 'hjtpx';
const DB_USER = process.env.DB_USER || 'postgres';
const DB_PASSWORD = process.env.DB_PASSWORD || 'postgres';

const MIGRATIONS_TABLE = 'migrations';
const MIGRATIONS_DIR = path.join(__dirname, '../migrations');

async function createMigrationsTable(pool) {
  await pool.query(`
    CREATE TABLE IF NOT EXISTS ${MIGRATIONS_TABLE} (
      id SERIAL PRIMARY KEY,
      name VARCHAR(255) NOT NULL UNIQUE,
      applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    )
  `);
}

async function getAppliedMigrations(pool) {
  const result = await pool.query(
    `SELECT name FROM ${MIGRATIONS_TABLE} ORDER BY applied_at ASC`
  );
  return result.rows.map(row => row.name);
}

async function getPendingMigrations(pool) {
  const applied = await getAppliedMigrations(pool);
  const files = fs.readdirSync(MIGRATIONS_DIR)
    .filter(f => f.endsWith('.sql'))
    .sort();

  return files.filter(file => !applied.includes(file));
}

async function recordMigration(pool, name) {
  await pool.query(
    `INSERT INTO ${MIGRATIONS_TABLE} (name) VALUES ($1)`,
    [name]
  );
}

async function removeMigrationRecord(pool, name) {
  await pool.query(
    `DELETE FROM ${MIGRATIONS_TABLE} WHERE name = $1`,
    [name]
  );
}

async function migrate(options = {}) {
  const pool = new Pool({
    host: DB_HOST,
    port: DB_PORT,
    database: DB_NAME,
    user: DB_USER,
    password: DB_PASSWORD,
  });

  try {
    await createMigrationsTable(pool);

    if (options.rollback) {
      const applied = await getAppliedMigrations(pool);
      if (applied.length === 0) {
        console.log('No migrations to rollback');
        return;
      }

      const lastMigration = applied[applied.length - 1];
      console.log(`Rolling back migration: ${lastMigration}`);

      const sql = fs.readFileSync(path.join(MIGRATIONS_DIR, lastMigration), 'utf8');
      const statements = sql.split(';').filter(s => s.trim());

      await pool.query('BEGIN');
      try {
        for (const statement of statements.reverse()) {
          const rollbackStatement = generateRollbackStatement(statement.trim());
          if (rollbackStatement) {
            await pool.query(rollbackStatement);
          }
        }
        await removeMigrationRecord(pool, lastMigration);
        await pool.query('COMMIT');
        console.log(`Rollback of ${lastMigration} completed`);
      } catch (error) {
        await pool.query('ROLLBACK');
        throw error;
      }
    } else {
      const pending = await getPendingMigrations(pool);

      if (pending.length === 0) {
        console.log('No pending migrations');
        return;
      }

      console.log(`Found ${pending.length} pending migration(s)`);

      for (const file of pending) {
        console.log(`Applying migration: ${file}`);
        const sql = fs.readFileSync(path.join(MIGRATIONS_DIR, file), 'utf8');

        await pool.query('BEGIN');
        try {
          await pool.query(sql);
          await recordMigration(pool, file);
          await pool.query('COMMIT');
          console.log(`Migration ${file} applied successfully`);
        } catch (error) {
          await pool.query('ROLLBACK');
          throw error;
        }
      }
    }
  } catch (error) {
    console.error('Migration error:', error.message);
    throw error;
  } finally {
    await pool.end();
  }
}

function generateRollbackStatement(statement) {
  if (statement.toUpperCase().startsWith('CREATE TABLE')) {
    const tableName = statement.match(/CREATE TABLE (?:\s+)?(\w+)/i);
    if (tableName) {
      return `DROP TABLE IF EXISTS ${tableName[1]}`;
    }
  }

  if (statement.toUpperCase().startsWith('CREATE INDEX')) {
    const indexName = statement.match(/CREATE INDEX (?:\s+)?\w+ (?:\s+)?ON/i);
    if (indexName) {
      const match = statement.match(/CREATE INDEX (?:\s+)?(\w+)/i);
      if (match) {
        return `DROP INDEX IF EXISTS ${match[1]}`;
      }
    }
  }

  if (statement.toUpperCase().startsWith('ALTER TABLE')) {
    const alterMatch = statement.match(/ALTER TABLE (\w+) ADD COLUMN (\w+)/i);
    if (alterMatch) {
      return `ALTER TABLE ${alterMatch[1]} DROP COLUMN ${alterMatch[2]}`;
    }
  }

  return null;
}

async function status() {
  const pool = new Pool({
    host: DB_HOST,
    port: DB_PORT,
    database: DB_NAME,
    user: DB_USER,
    password: DB_PASSWORD,
  });

  try {
    await createMigrationsTable(pool);
    const applied = await getAppliedMigrations(pool);
    const pending = await getPendingMigrations(pool);

    console.log('\n=== Migration Status ===');
    console.log(`Applied: ${applied.length}`);
    console.log(`Pending: ${pending.length}`);
    console.log('\nApplied migrations:');
    applied.forEach(m => console.log(`  ✓ ${m}`));
    console.log('\nPending migrations:');
    pending.forEach(m => console.log(`  ○ ${m}`));
    console.log('========================\n');
  } catch (error) {
    console.error('Error getting migration status:', error.message);
    throw error;
  } finally {
    await pool.end();
  }
}

async function create(name) {
  const timestamp = new Date().toISOString().replace(/[-:]/g, '').split('.')[0];
  const filename = `${timestamp}_${name}.sql`;
  const filepath = path.join(MIGRATIONS_DIR, filename);

  const template = `-- Migration: ${name}
-- Created: ${new Date().toISOString()}

-- TODO: Write migration SQL here

`;

  fs.writeFileSync(filepath, template);
  console.log(`Created migration: ${filename}`);
}

function main() {
  const args = process.argv.slice(2);
  const command = args[0];

  switch (command) {
    case 'up':
      migrate({ rollback: false })
        .then(() => process.exit(0))
        .catch(() => process.exit(1));
      break;
    case 'down':
      migrate({ rollback: true })
        .then(() => process.exit(0))
        .catch(() => process.exit(1));
      break;
    case 'status':
      status()
        .then(() => process.exit(0))
        .catch(() => process.exit(1));
      break;
    case 'create':
      if (args[1]) {
        create(args[1]);
      } else {
        console.error('Please provide a migration name');
        process.exit(1);
      }
      break;
    default:
      console.log('Usage: node migrate.js [up|down|status|create <name>]');
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
};
