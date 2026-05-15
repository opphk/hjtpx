#!/usr/bin/env node
require('dotenv').config();
const { Pool } = require('pg');
const { migrateUp, showStatus } = require('./migrate');

const DB_HOST = process.env.DB_HOST || 'localhost';
const DB_PORT = process.env.DB_PORT || 5432;
const DB_NAME = process.env.DB_NAME || 'hjtpx';
const DB_USER = process.env.DB_USER || 'postgres';
const DB_PASSWORD = process.env.DB_PASSWORD || 'postgres';

async function runMigrations() {
  const pool = new Pool({
    host: DB_HOST,
    port: DB_PORT,
    database: DB_NAME,
    user: DB_USER,
    password: DB_PASSWORD,
  });

  try {
    console.log('=== Running Database Migrations ===\n');

    console.log('Checking migration status...');
    await showStatus(pool);

    console.log('Applying migrations...');
    await migrateUp(pool);

    console.log('\n=== Migration Complete ===');
    console.log('Checking final status...');
    await showStatus(pool);

    process.exit(0);
  } catch (error) {
    console.error('Migration failed:', error.message);
    process.exit(1);
  } finally {
    await pool.end();
  }
}

if (require.main === module) {
  runMigrations();
}

module.exports = { runMigrations };
