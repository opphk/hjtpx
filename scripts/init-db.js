const { Pool } = require('pg');
const fs = require('fs');
const path = require('path');
require('dotenv').config();

const DB_HOST = process.env.DB_HOST || 'localhost';
const DB_PORT = process.env.DB_PORT || 5432;
const DB_NAME = process.env.DB_NAME || 'hjtpx';
const DB_USER = process.env.DB_USER || 'postgres';
const DB_PASSWORD = process.env.DB_PASSWORD || 'postgres';

async function createDatabase() {
  const adminPool = new Pool({
    host: DB_HOST,
    port: DB_PORT,
    database: 'postgres',
    user: DB_USER,
    password: DB_PASSWORD,
  });

  try {
    const result = await adminPool.query(
      `SELECT 1 FROM pg_database WHERE datname = $1`,
      [DB_NAME]
    );

    if (result.rows.length === 0) {
      console.log(`Creating database: ${DB_NAME}`);
      await adminPool.query(`CREATE DATABASE ${DB_NAME}`);
      console.log(`Database ${DB_NAME} created successfully`);
    } else {
      console.log(`Database ${DB_NAME} already exists`);
    }
  } catch (error) {
    console.error('Error creating database:', error.message);
    throw error;
  } finally {
    await adminPool.end();
  }
}

async function runMigrations() {
  const pool = new Pool({
    host: DB_HOST,
    port: DB_PORT,
    database: DB_NAME,
    user: DB_USER,
    password: DB_PASSWORD,
  });

  try {
    const migrationsDir = path.join(__dirname, '../migrations');
    const files = fs.readdirSync(migrationsDir)
      .filter(f => f.endsWith('.sql'))
      .sort();

    console.log(`Found ${files.length} migration files`);

    for (const file of files) {
      console.log(`Running migration: ${file}`);
      const sql = fs.readFileSync(path.join(migrationsDir, file), 'utf8');
      await pool.query(sql);
      console.log(`Migration ${file} completed`);
    }

    console.log('All migrations completed successfully');
  } catch (error) {
    console.error('Error running migrations:', error.message);
    throw error;
  } finally {
    await pool.end();
  }
}

async function createTestData() {
  const pool = new Pool({
    host: DB_HOST,
    port: DB_PORT,
    database: DB_NAME,
    user: DB_USER,
    password: DB_PASSWORD,
  });

  try {
    const testUsers = [
      {
        email: 'admin@hjtpx.com',
        name: 'Admin User',
        password: '$2b$10$EixZaYVK1fsbw1ZfbX3OXePaWxn96p36Zy.q7W.bNNo9.3.0q.y1e',
        role: 'admin',
      },
      {
        email: 'test@hjtpx.com',
        name: 'Test User',
        password: '$2b$10$EixZaYVK1fsbw1ZfbX3OXePaWxn96p36Zy.q7W.bNNo9.3.0q.y1e',
        role: 'user',
      },
    ];

    for (const user of testUsers) {
      const exists = await pool.query(
        'SELECT 1 FROM users WHERE email = $1',
        [user.email]
      );

      if (exists.rows.length === 0) {
        await pool.query(
          'INSERT INTO users (email, name, password, role) VALUES ($1, $2, $3, $4)',
          [user.email, user.name, user.password, user.role]
        );
        console.log(`Test user created: ${user.email}`);
      } else {
        console.log(`Test user already exists: ${user.email}`);
      }
    }

    console.log('Test data created successfully');
  } catch (error) {
    console.error('Error creating test data:', error.message);
    throw error;
  } finally {
    await pool.end();
  }
}

async function verifyInitialization() {
  const pool = new Pool({
    host: DB_HOST,
    port: DB_PORT,
    database: DB_NAME,
    user: DB_USER,
    password: DB_PASSWORD,
  });

  try {
    const tablesResult = await pool.query(`
      SELECT table_name
      FROM information_schema.tables
      WHERE table_schema = 'public'
      ORDER BY table_name
    `);

    const userCount = await pool.query('SELECT COUNT(*) FROM users');
    const sessionCount = await pool.query('SELECT COUNT(*) FROM sessions');

    console.log('\n=== Database Initialization Verification ===');
    console.log('Tables:', tablesResult.rows.map(r => r.table_name).join(', '));
    console.log('User count:', userCount.rows[0].count);
    console.log('Session count:', sessionCount.rows[0].count);
    console.log('============================================\n');

    return {
      tables: tablesResult.rows.map(r => r.table_name),
      userCount: parseInt(userCount.rows[0].count),
      sessionCount: parseInt(sessionCount.rows[0].count),
    };
  } catch (error) {
    console.error('Error verifying initialization:', error.message);
    throw error;
  } finally {
    await pool.end();
  }
}

async function main() {
  console.log('Starting database initialization...\n');

  try {
    await createDatabase();
    await runMigrations();
    await createTestData();
    const verification = await verifyInitialization();

    console.log('Database initialization completed successfully!');
    console.log('Verification results:', verification);

    process.exit(0);
  } catch (error) {
    console.error('Database initialization failed:', error);
    process.exit(1);
  }
}

if (require.main === module) {
  main();
}

module.exports = {
  createDatabase,
  runMigrations,
  createTestData,
  verifyInitialization,
};
