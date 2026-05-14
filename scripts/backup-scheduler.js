require('dotenv').config();
const cron = require('node-cron');
const backup = require('./backup');

const FULL_BACKUP_CRON = process.env.FULL_BACKUP_CRON || '0 2 * * 0';
const INCREMENTAL_BACKUP_CRON = process.env.INCREMENTAL_BACKUP_CRON || '0 2 * * 1-6';
const VERIFY_BACKUP_CRON = process.env.VERIFY_BACKUP_CRON || '0 3 * * 0';
const CLEANUP_CRON = process.env.CLEANUP_CRON || '0 4 * * 0';

const jobs = {
  fullBackup: null,
  incrementalBackup: null,
  verifyBackup: null,
  cleanup: null,
};

function formatCronExpression(cronExpr) {
  const parts = cronExpr.split(' ');
  const descriptions = [
    'Minute',
    'Hour',
    'Day of Month',
    'Month',
    'Day of Week'
  ];
  
  return cronExpr + ' (' + descriptions.map((desc, i) => 
    `${desc}: ${parts[i]}`
  ).join(', ') + ')';
}

function startScheduler() {
  console.log('\n📅 Backup Scheduler Started');
  console.log('='.repeat(50));
  console.log('Scheduled Jobs:');
  console.log(`  Full Backup:     ${formatCronExpression(FULL_BACKUP_CRON)}`);
  console.log(`  Incremental:     ${formatCronExpression(INCREMENTAL_BACKUP_CRON)}`);
  console.log(`  Verification:    ${formatCronExpression(VERIFY_BACKUP_CRON)}`);
  console.log(`  Cleanup:         ${formatCronExpression(CLEANUP_CRON)}`);
  console.log('='.repeat(50));

  jobs.fullBackup = cron.schedule(FULL_BACKUP_CRON, async () => {
    console.log('\n⏰ [SCHEDULED] Starting scheduled full backup...');
    try {
      await backup.backupPostgres('full');
      await backup.backupMongo('full');
    } catch (error) {
      console.error('Full backup failed:', error.message);
    }
  }, {
    scheduled: true,
    timezone: process.env.TIMEZONE || 'UTC'
  });
  console.log('✅ Full backup job scheduled');

  jobs.incrementalBackup = cron.schedule(INCREMENTAL_BACKUP_CRON, async () => {
    console.log('\n⏰ [SCHEDULED] Starting scheduled incremental backup...');
    try {
      await backup.backupPostgres('incremental');
      await backup.backupMongo('incremental');
    } catch (error) {
      console.error('Incremental backup failed:', error.message);
    }
  }, {
    scheduled: true,
    timezone: process.env.TIMEZONE || 'UTC'
  });
  console.log('✅ Incremental backup job scheduled');

  jobs.verifyBackup = cron.schedule(VERIFY_BACKUP_CRON, async () => {
    console.log('\n⏰ [SCHEDULED] Starting scheduled backup verification...');
    try {
      await backup.verifyBackup();
    } catch (error) {
      console.error('Backup verification failed:', error.message);
    }
  }, {
    scheduled: true,
    timezone: process.env.TIMEZONE || 'UTC'
  });
  console.log('✅ Backup verification job scheduled');

  jobs.cleanup = cron.schedule(CLEANUP_CRON, async () => {
    console.log('\n⏰ [SCHEDULED] Starting scheduled backup cleanup...');
    try {
      await backup.cleanupOldBackups();
    } catch (error) {
      console.error('Backup cleanup failed:', error.message);
    }
  }, {
    scheduled: true,
    timezone: process.env.TIMEZONE || 'UTC'
  });
  console.log('✅ Cleanup job scheduled');

  console.log('\n✅ All backup jobs scheduled successfully\n');
}

function stopScheduler() {
  console.log('\n🛑 Stopping backup scheduler...');
  
  Object.values(jobs).forEach((job, index) => {
    if (job) {
      job.stop();
      console.log(`   Stopped job ${index + 1}`);
    }
  });
  
  console.log('✅ Scheduler stopped\n');
}

function getSchedulerStatus() {
  return {
    running: true,
    jobs: {
      fullBackup: jobs.fullBackup ? 'scheduled' : 'stopped',
      incrementalBackup: jobs.incrementalBackup ? 'scheduled' : 'stopped',
      verifyBackup: jobs.verifyBackup ? 'scheduled' : 'stopped',
      cleanup: jobs.cleanup ? 'scheduled' : 'stopped',
    },
    schedules: {
      fullBackup: FULL_BACKUP_CRON,
      incrementalBackup: INCREMENTAL_BACKUP_CRON,
      verifyBackup: VERIFY_BACKUP_CRON,
      cleanup: CLEANUP_CRON,
    },
  };
}

process.on('SIGTERM', () => {
  stopScheduler();
  process.exit(0);
});

process.on('SIGINT', () => {
  stopScheduler();
  process.exit(0);
});

if (require.main === module) {
  startScheduler();
}

module.exports = {
  startScheduler,
  stopScheduler,
  getSchedulerStatus,
};
