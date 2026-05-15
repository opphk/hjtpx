require('dotenv').config();
const fs = require('fs').promises;
const path = require('path');
const { execSync } = require('child_process');
const backup = require('./backup');

const DRILL_ENV = process.env.DRILL_ENV || 'staging';
const DRILL_DB_NAME = process.env.DRILL_DB_NAME || 'hjtpx_drill';
const DRILL_DB_HOST = process.env.DRILL_DB_HOST || 'localhost';
const DRILL_DB_PORT = process.env.DRILL_DB_PORT || '5432';
const DRILL_DB_USER = process.env.DRILL_DB_USER || 'postgres';
const DRILL_DB_PASSWORD = process.env.DRILL_DB_PASSWORD || '';
const BACKUP_DIR = process.env.BACKUP_DIR || './backups';

const drillResults = [];
let drillId;

async function logDrillStep(step, status, message, details = {}) {
  const result = {
    step,
    status,
    message,
    details,
    timestamp: new Date().toISOString(),
  };
  drillResults.push(result);
  console.log(`  [${status}] ${message}`);
  if (details.error) {
    console.log(`    Error: ${details.error}`);
  }
}

async function createDrillDatabase() {
  logDrillStep('Create Drill Database', 'INFO', `Creating drill database: ${DRILL_DB_NAME}`);
  
  try {
    const createCmd = `psql -h ${DRILL_DB_HOST} -p ${DRILL_DB_PORT} -U ${DRILL_DB_USER} -c "DROP DATABASE IF EXISTS ${DRILL_DB_NAME}"`;
    execSync(createCmd, { stdio: 'pipe' });
    logDrillStep('Create Drill Database', 'SUCCESS', 'Dropped existing drill database');
  } catch (error) {
    logDrillStep('Create Drill Database', 'WARNING', 'Database may not exist yet, continuing...');
  }
  
  try {
    const createCmd = `psql -h ${DRILL_DB_HOST} -p ${DRILL_DB_PORT} -U ${DRILL_DB_USER} -c "CREATE DATABASE ${DRILL_DB_NAME}"`;
    execSync(createCmd, { stdio: 'pipe' });
    logDrillStep('Create Drill Database', 'SUCCESS', 'Created drill database');
    return true;
  } catch (error) {
    logDrillStep('Create Drill Database', 'FAILED', 'Failed to create drill database', { error: error.message });
    return false;
  }
}

async function restoreToDrillDatabase(backupPath) {
  logDrillStep('Restore to Drill DB', 'INFO', `Restoring backup to drill database...`);
  
  try {
    const restoreCmd = `pg_restore -h ${DRILL_DB_HOST} -p ${DRILL_DB_PORT} -U ${DRILL_DB_USER} -d ${DRILL_DB_NAME} "${backupPath}"`;
    execSync(restoreCmd, { stdio: 'pipe' });
    logDrillStep('Restore to Drill DB', 'SUCCESS', 'Restored backup to drill database');
    return true;
  } catch (error) {
    logDrillStep('Restore to Drill DB', 'FAILED', 'Failed to restore backup', { error: error.message });
    return false;
  }
}

async function verifyRestoredData() {
  logDrillStep('Verify Restored Data', 'INFO', 'Verifying restored data integrity...');
  
  const checks = {
    tables: false,
    rowCounts: false,
    indexes: false,
  };
  
  try {
    const tablesCmd = `psql -h ${DRILL_DB_HOST} -p ${DRILL_DB_PORT} -U ${DRILL_DB_USER} -d ${DRILL_DB_NAME} -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public'" -t`;
    const tablesResult = execSync(tablesCmd, { encoding: 'utf8' }).trim();
    const tableCount = parseInt(tablesResult, 10);
    checks.tables = tableCount > 0;
    logDrillStep('Verify Tables', checks.tables ? 'SUCCESS' : 'FAILED', `Found ${tableCount} tables`);
  } catch (error) {
    logDrillStep('Verify Tables', 'FAILED', 'Failed to verify tables', { error: error.message });
  }
  
  try {
    const indexesCmd = `psql -h ${DRILL_DB_HOST} -p ${DRILL_DB_PORT} -U ${DRILL_DB_USER} -d ${DRILL_DB_NAME} -c "SELECT COUNT(*) FROM pg_indexes WHERE schemaname = 'public'" -t`;
    const indexesResult = execSync(indexesCmd, { encoding: 'utf8' }).trim();
    const indexCount = parseInt(indexesResult, 10);
    checks.indexes = indexCount > 0;
    logDrillStep('Verify Indexes', checks.indexes ? 'SUCCESS' : 'FAILED', `Found ${indexCount} indexes`);
  } catch (error) {
    logDrillStep('Verify Indexes', 'FAILED', 'Failed to verify indexes', { error: error.message });
  }
  
  const allPassed = Object.values(checks).every(v => v);
  logDrillStep('Verify Restored Data', allPassed ? 'SUCCESS' : 'FAILED', 'Data verification complete');
  
  return allPassed;
}

async function testQueryPerformance() {
  logDrillStep('Query Performance Test', 'INFO', 'Testing query performance...');
  
  try {
    const queries = [
      'SELECT 1',
      'SELECT * FROM information_schema.tables LIMIT 1',
    ];
    
    for (const query of queries) {
      const start = Date.now();
      const cmd = `psql -h ${DRILL_DB_HOST} -p ${DRILL_DB_PORT} -U ${DRILL_DB_USER} -d ${DRILL_DB_NAME} -c "${query}" -t`;
      execSync(cmd, { stdio: 'pipe' });
      const duration = Date.now() - start;
      
      logDrillStep(`Query: ${query.substring(0, 50)}...`, 'SUCCESS', `Completed in ${duration}ms`);
    }
    
    return true;
  } catch (error) {
    logDrillStep('Query Performance Test', 'FAILED', 'Query performance test failed', { error: error.message });
    return false;
  }
}

async function cleanupDrillDatabase() {
  logDrillStep('Cleanup Drill DB', 'INFO', 'Cleaning up drill database...');
  
  try {
    const dropCmd = `psql -h ${DRILL_DB_HOST} -p ${DRILL_DB_PORT} -U ${DRILL_DB_USER} -c "DROP DATABASE IF EXISTS ${DRILL_DB_NAME}"`;
    execSync(dropCmd, { stdio: 'pipe' });
    logDrillStep('Cleanup Drill DB', 'SUCCESS', 'Cleaned up drill database');
    return true;
  } catch (error) {
    logDrillStep('Cleanup Drill DB', 'FAILED', 'Failed to cleanup drill database', { error: error.message });
    return false;
  }
}

async function generateDrillReport() {
  const report = {
    drillId,
    environment: DRILL_ENV,
    startTime: drillResults[0]?.timestamp,
    endTime: drillResults[drillResults.length - 1]?.timestamp,
    totalSteps: drillResults.length,
    results: drillResults,
    summary: {
      passed: drillResults.filter(r => r.status === 'SUCCESS').length,
      failed: drillResults.filter(r => r.status === 'FAILED').length,
      warnings: drillResults.filter(r => r.status === 'WARNING').length,
      info: drillResults.filter(r => r.status === 'INFO').length,
    },
    overallStatus: drillResults.some(r => r.status === 'FAILED') ? 'FAILED' : 'SUCCESS',
    recommendations: [],
  };
  
  if (report.summary.failed > 0) {
    report.recommendations.push('Review and fix failed backup/restore procedures');
  }
  
  if (report.summary.warnings > 0) {
    report.recommendations.push('Investigate warnings to prevent potential issues');
  }
  
  if (report.overallStatus === 'SUCCESS') {
    report.recommendations.push('Backup and restore procedures are working correctly');
    report.recommendations.push('Consider running drill more frequently');
  }
  
  const reportDir = path.join(BACKUP_DIR, 'drills');
  await fs.mkdir(reportDir, { recursive: true });
  
  const reportPath = path.join(reportDir, `drill_${drillId}.json`);
  await fs.writeFile(reportPath, JSON.stringify(report, null, 2));
  
  console.log('\n' + '='.repeat(80));
  console.log('DRILL REPORT SUMMARY');
  console.log('='.repeat(80));
  console.log(`Drill ID: ${drillId}`);
  console.log(`Environment: ${DRILL_ENV}`);
  console.log(`Status: ${report.overallStatus}`);
  console.log(`Steps Passed: ${report.summary.passed}/${report.totalSteps}`);
  console.log(`Steps Failed: ${report.summary.failed}`);
  console.log(`Report saved to: ${reportPath}`);
  console.log('='.repeat(80));
  
  if (report.recommendations.length > 0) {
    console.log('\nRecommendations:');
    report.recommendations.forEach((rec, i) => {
      console.log(`  ${i + 1}. ${rec}`);
    });
  }
  console.log('='.repeat(80) + '\n');
  
  return report;
}

async function runDrill() {
  drillId = `drill_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  
  console.log('\n' + '='.repeat(80));
  console.log('🔄 DATABASE BACKUP DISASTER RECOVERY DRILL');
  console.log('='.repeat(80));
  console.log(`Drill ID: ${drillId}`);
  console.log(`Environment: ${DRILL_ENV}`);
  console.log(`Target DB: ${DRILL_DB_NAME}`);
  console.log(`Start Time: ${new Date().toISOString()}`);
  console.log('='.repeat(80) + '\n');
  
  try {
    const state = await backup.loadBackupState();
    const latestBackup = state.lastFullBackup || state.backups[state.backups.length - 1];
    
    if (!latestBackup) {
      logDrillStep('Find Latest Backup', 'FAILED', 'No backup found to test');
      return await generateDrillReport();
    }
    
    logDrillStep('Find Latest Backup', 'SUCCESS', `Using backup: ${latestBackup.filename}`);
    
    await createDrillDatabase();
    
    await restoreToDrillDatabase(latestBackup.filepath);
    
    await verifyRestoredData();
    
    await testQueryPerformance();
    
    await cleanupDrillDatabase();
    
  } catch (error) {
    logDrillStep('DRILL EXECUTION', 'FAILED', 'Drill execution failed', { error: error.message });
    console.error('Drill error:', error);
  }
  
  return await generateDrillReport();
}

if (require.main === module) {
  runDrill()
    .then(report => {
      process.exit(report.overallStatus === 'SUCCESS' ? 0 : 1);
    })
    .catch(error => {
      console.error('Fatal drill error:', error);
      process.exit(1);
    });
}

module.exports = {
  runDrill,
  verifyRestoredData,
  testQueryPerformance,
  generateDrillReport,
};
