require('dotenv').config();
const fs = require('fs').promises;
const path = require('path');
const { execSync, exec } = require('child_process');
const crypto = require('crypto');

const NODE_ENV = process.env.NODE_ENV || 'development';
const BACKUP_DIR = process.env.BACKUP_DIR || './backups';
const DB_TYPE = process.env.DB_TYPE || 'postgresql';
const DB_HOST = process.env.DB_HOST || 'localhost';
const DB_PORT = process.env.DB_PORT || '5432';
const DB_NAME = process.env.DB_NAME || 'hjtpx';
const DB_USER = process.env.DB_USER || 'postgres';
const DB_PASSWORD = process.env.DB_PASSWORD || '';
const RETENTION_DAYS = parseInt(process.env.BACKUP_RETENTION_DAYS || '30', 10);

const backupStateFile = path.join(BACKUP_DIR, '.backup_state.json');

async function ensureBackupDir() {
  try {
    await fs.mkdir(BACKUP_DIR, { recursive: true });
    await fs.mkdir(path.join(BACKUP_DIR, 'full'), { recursive: true });
    await fs.mkdir(path.join(BACKUP_DIR, 'incremental'), { recursive: true });
    await fs.mkdir(path.join(BACKUP_DIR, 'verify'), { recursive: true });
  } catch (error) {
    console.error('Failed to create backup directories:', error.message);
    throw error;
  }
}

async function loadBackupState() {
  try {
    const data = await fs.readFile(backupStateFile, 'utf8');
    return JSON.parse(data);
  } catch (error) {
    return {
      lastFullBackup: null,
      lastIncrementalBackup: null,
      incrementalSequence: 0,
      backups: [],
    };
  }
}

async function saveBackupState(state) {
  await ensureBackupDir();
  await fs.writeFile(backupStateFile, JSON.stringify(state, null, 2));
}

async function calculateChecksum(filePath) {
  const fileBuffer = await fs.readFile(filePath);
  const hashSum = crypto.createHash('sha256');
  hashSum.update(fileBuffer);
  return hashSum.digest('hex');
}

async function getBackupSize(filePath) {
  const stats = await fs.stat(filePath);
  return stats.size;
}

function formatBytes(bytes) {
  if (bytes === 0) return '0 Bytes';
  const k = 1024;
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
}

function getBackupFilename(type, timestamp) {
  const date = new Date(timestamp);
  const dateStr = date.toISOString().replace(/[:.]/g, '-');
  return `${type}_${dateStr}.sql`;
}

async function backupPostgres(type = 'full') {
  console.log(`\n📦 Starting PostgreSQL ${type} backup...`);
  
  const timestamp = Date.now();
  const backupType = type === 'full' ? 'full' : 'incremental';
  const filename = getBackupFilename(backupType, timestamp);
  const filepath = path.join(BACKUP_DIR, backupType, filename);
  
  try {
    let backupCmd;
    const env = { ...process.env, PGPassword: DB_PASSWORD };
    
    if (type === 'full') {
      backupCmd = `pg_dump -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d ${DB_NAME} -Fc -f "${filepath}"`;
    } else {
      const state = await loadBackupState();
      if (!state.lastFullBackup) {
        console.warn('⚠️ No full backup found. Creating full backup instead.');
        return backupPostgres('full');
      }
      backupCmd = `pg_dump -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d ${DB_NAME} -Fc -f "${filepath}"`;
    }
    
    execSync(backupCmd, { 
      env,
      stdio: 'inherit',
      cwd: BACKUP_DIR 
    });
    
    const checksum = await calculateChecksum(filepath);
    const size = await getBackupSize(filepath);
    
    const backupInfo = {
      id: crypto.randomUUID(),
      type: backupType,
      filename,
      filepath,
      checksum,
      size,
      sizeFormatted: formatBytes(size),
      timestamp,
      date: new Date(timestamp).toISOString(),
      retentionDays: RETENTION_DAYS,
      compressed: true,
      dbType: 'postgresql',
    };
    
    const state = await loadBackupState();
    
    if (type === 'full') {
      state.lastFullBackup = backupInfo;
      state.incrementalSequence = 0;
    } else {
      state.incrementalSequence++;
      backupInfo.sequenceNumber = state.incrementalSequence;
      backupInfo.parentBackup = state.lastFullBackup?.id;
    }
    
    state.lastIncrementalBackup = backupInfo;
    state.backups.push(backupInfo);
    
    await saveBackupState(state);
    
    console.log(`\n✅ ${type} backup completed successfully!`);
    console.log(`   File: ${filename}`);
    console.log(`   Size: ${formatBytes(size)}`);
    console.log(`   Checksum: ${checksum.substring(0, 16)}...`);
    
    return backupInfo;
  } catch (error) {
    console.error(`\n❌ Backup failed:`, error.message);
    throw error;
  }
}

async function backupMongo(type = 'full') {
  console.log(`\n📦 Starting MongoDB ${type} backup...`);
  
  const timestamp = Date.now();
  const backupType = type === 'full' ? 'full' : 'incremental';
  const filename = getBackupFilename(backupType, timestamp).replace('.sql', '.archive');
  const filepath = path.join(BACKUP_DIR, backupType, filename);
  
  try {
    const MONGO_URI = process.env.MONGO_URI || `mongodb://${DB_HOST}:${DB_PORT}/${DB_NAME}`;
    
    if (type === 'full') {
      const cmd = `mongodump --uri="${MONGO_URI}" --archive="${filepath}" --gzip`;
      execSync(cmd, { stdio: 'inherit' });
    } else {
      const state = await loadBackupState();
      if (!state.lastFullBackup) {
        console.warn('⚠️ No full backup found. Creating full backup instead.');
        return backupMongo('full');
      }
      const cmd = `mongodump --uri="${MONGO_URI}" --archive="${filepath}" --gzip`;
      execSync(cmd, { stdio: 'inherit' });
    }
    
    const checksum = await calculateChecksum(filepath);
    const size = await getBackupSize(filepath);
    
    const backupInfo = {
      id: crypto.randomUUID(),
      type: backupType,
      filename,
      filepath,
      checksum,
      size,
      sizeFormatted: formatBytes(size),
      timestamp,
      date: new Date(timestamp).toISOString(),
      retentionDays: RETENTION_DAYS,
      compressed: true,
      dbType: 'mongodb',
    };
    
    const state = await loadBackupState();
    
    if (type === 'full') {
      state.lastFullBackup = backupInfo;
      state.incrementalSequence = 0;
    } else {
      state.incrementalSequence++;
      backupInfo.sequenceNumber = state.incrementalSequence;
      backupInfo.parentBackup = state.lastFullBackup?.id;
    }
    
    state.lastIncrementalBackup = backupInfo;
    state.backups.push(backupInfo);
    
    await saveBackupState(state);
    
    console.log(`\n✅ ${type} backup completed successfully!`);
    console.log(`   File: ${filename}`);
    console.log(`   Size: ${formatBytes(size)}`);
    
    return backupInfo;
  } catch (error) {
    console.error(`\n❌ Backup failed:`, error.message);
    throw error;
  }
}

async function verifyBackup(backupPath = null) {
  console.log('\n🔍 Verifying backup...');
  
  const state = await loadBackupState();
  const backupToVerify = backupPath 
    ? state.backups.find(b => b.filepath === backupPath)
    : state.lastFullBackup;
  
  if (!backupToVerify) {
    throw new Error('No backup found to verify');
  }
  
  try {
    const currentChecksum = await calculateChecksum(backupToVerify.filepath);
    const isValid = currentChecksum === backupToVerify.checksum;
    const size = await getBackupSize(backupToVerify.filepath);
    
    const verificationReport = {
      backupId: backupToVerify.id,
      filename: backupToVerify.filename,
      filepath: backupToVerify.filepath,
      originalChecksum: backupToVerify.checksum,
      currentChecksum,
      checksumValid: isValid,
      originalSize: backupToVerify.size,
      currentSize: size,
      sizeValid: size === backupToVerify.size,
      verifiedAt: new Date().toISOString(),
      status: isValid && size === backupToVerify.size ? 'VALID' : 'INVALID',
    };
    
    if (!isValid) {
      console.warn('⚠️ Checksum mismatch detected!');
      sentryService?.captureMessage('Backup verification failed', 'error', {
        tags: { backup_id: backupToVerify.id },
        extra: verificationReport,
      });
    }
    
    const verifyReportPath = path.join(BACKUP_DIR, 'verify', `verify_${Date.now()}.json`);
    await fs.writeFile(verifyReportPath, JSON.stringify(verificationReport, null, 2));
    
    console.log(`\n${isValid ? '✅' : '❌'} Backup verification ${isValid ? 'passed' : 'FAILED'}!`);
    console.log(`   Checksum: ${isValid ? 'Valid' : 'MISMATCH'}`);
    console.log(`   Size: ${size === backupToVerify.size ? 'Valid' : 'MISMATCH'}`);
    console.log(`   Report: ${verifyReportPath}`);
    
    return verificationReport;
  } catch (error) {
    console.error(`\n❌ Verification failed:`, error.message);
    throw error;
  }
}

async function restoreBackup(backupPath = null) {
  console.log('\n🔄 Starting backup restore...');
  
  const state = await loadBackupState();
  const backupToRestore = backupPath 
    ? state.backups.find(b => b.filepath === backupPath)
    : state.lastFullBackup;
  
  if (!backupToRestore) {
    throw new Error('No backup found to restore');
  }
  
  console.log(`   Backup: ${backupToRestore.filename}`);
  console.log(`   Type: ${backupToRestore.type}`);
  console.log(`   Date: ${backupToRestore.date}`);
  
  try {
    if (DB_TYPE === 'postgresql') {
      const restoreCmd = `pg_restore -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d ${DB_NAME} --clean --if-exists "${backupToRestore.filepath}"`;
      console.log('\n⚠️ Restoring PostgreSQL backup (this will overwrite existing data)...');
      execSync(restoreCmd, { stdio: 'inherit' });
    } else if (DB_TYPE === 'mongodb') {
      const MONGO_URI = process.env.MONGO_URI || `mongodb://${DB_HOST}:${DB_PORT}/${DB_NAME}`;
      const restoreCmd = `mongorestore --uri="${MONGO_URI}" --drop --archive="${backupToRestore.filepath}" --gzip`;
      console.log('\n⚠️ Restoring MongoDB backup (this will overwrite existing data)...');
      execSync(restoreCmd, { stdio: 'inherit' });
    }
    
    const restoreReport = {
      backupId: backupToRestore.id,
      filename: backupToRestore.filename,
      restoredAt: new Date().toISOString(),
      status: 'SUCCESS',
      dbType: DB_TYPE,
    };
    
    console.log('\n✅ Restore completed successfully!');
    
    return restoreReport;
  } catch (error) {
    console.error('\n❌ Restore failed:', error.message);
    throw error;
  }
}

async function cleanupOldBackups() {
  console.log('\n🧹 Cleaning up old backups...');
  
  const state = await loadBackupState();
  const cutoffDate = new Date(Date.now() - RETENTION_DAYS * 24 * 60 * 60 * 1000);
  
  const oldBackups = state.backups.filter(
    backup => new Date(backup.date) < cutoffDate
  );
  
  let cleanedCount = 0;
  for (const backup of oldBackups) {
    try {
      await fs.unlink(backup.filepath);
      cleanedCount++;
    } catch (error) {
      console.warn(`   Failed to delete: ${backup.filename}`);
    }
  }
  
  state.backups = state.backups.filter(
    backup => new Date(backup.date) >= cutoffDate
  );
  
  await saveBackupState(state);
  
  console.log(`   Cleaned up ${cleanedCount} old backup(s)`);
  console.log(`   Kept ${state.backups.length} recent backup(s)`);
  
  return { cleanedCount, remainingCount: state.backups.length };
}

async function listBackups() {
  const state = await loadBackupState();
  
  console.log('\n📋 Backup List');
  console.log('='.repeat(80));
  console.log(`Total backups: ${state.backups.length}`);
  console.log('');
  
  if (state.lastFullBackup) {
    console.log('Last Full Backup:');
    console.log(`  File: ${state.lastFullBackup.filename}`);
    console.log(`  Date: ${state.lastFullBackup.date}`);
    console.log(`  Size: ${state.lastFullBackup.sizeFormatted}`);
    console.log('');
  }
  
  if (state.lastIncrementalBackup) {
    console.log('Last Incremental Backup:');
    console.log(`  File: ${state.lastIncrementalBackup.filename}`);
    console.log(`  Date: ${state.lastIncrementalBackup.date}`);
    console.log(`  Size: ${state.lastIncrementalBackup.sizeFormatted}`);
    console.log('');
  }
  
  console.log('All Backups:');
  state.backups
    .sort((a, b) => new Date(b.date) - new Date(a.date))
    .slice(0, 10)
    .forEach(backup => {
      console.log(`  [${backup.type.toUpperCase()}] ${backup.filename}`);
      console.log(`    Date: ${backup.date} | Size: ${backup.sizeFormatted}`);
    });
  
  console.log('='.repeat(80));
}

async function main() {
  const args = process.argv.slice(2);
  let command = 'full';
  let backupPath = null;
  
  for (let i = 0; i < args.length; i++) {
    if (args[i] === '--type' && args[i + 1]) {
      command = args[i + 1];
      i++;
    } else if (args[i] === '--verify' && args[i + 1]) {
      command = 'verify';
      backupPath = args[i + 1];
      i++;
    } else if (args[i] === '--restore' && args[i + 1]) {
      command = 'restore';
      backupPath = args[i + 1];
      i++;
    } else if (args[i] === '--list') {
      command = 'list';
    } else if (args[i] === '--cleanup') {
      command = 'cleanup';
    }
  }
  
  await ensureBackupDir();
  
  switch (command) {
    case 'full':
      if (DB_TYPE === 'mongodb') {
        await backupMongo('full');
      } else {
        await backupPostgres('full');
      }
      break;
    case 'incremental':
      if (DB_TYPE === 'mongodb') {
        await backupMongo('incremental');
      } else {
        await backupPostgres('incremental');
      }
      break;
    case 'verify':
      await verifyBackup(backupPath);
      break;
    case 'restore':
      await restoreBackup(backupPath);
      break;
    case 'list':
      await listBackups();
      break;
    case 'cleanup':
      await cleanupOldBackups();
      break;
    default:
      console.error(`Unknown command: ${command}`);
      process.exit(1);
  }
}

if (require.main === module) {
  main().catch(error => {
    console.error('Fatal error:', error);
    process.exit(1);
  });
}

module.exports = {
  backupPostgres,
  backupMongo,
  verifyBackup,
  restoreBackup,
  cleanupOldBackups,
  listBackups,
  ensureBackupDir,
  loadBackupState,
};
