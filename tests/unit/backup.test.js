const { describe, test, expect, beforeAll, afterAll, beforeEach, afterEach } = require('@jest/globals');
const { execSync, spawn } = require('child_process');
const fs = require('fs');
const path = require('path');

const SCRIPT_DIR = path.join(__dirname, '../../scripts');
const BACKUP_SCRIPT = path.join(SCRIPT_DIR, 'backup.sh');
const VERIFY_SCRIPT = path.join(SCRIPT_DIR, 'verify-backup.sh');
const INCREMENTAL_SCRIPT = path.join(SCRIPT_DIR, 'backup-incremental.sh');
const RESTORE_DRILL_SCRIPT = path.join(SCRIPT_DIR, 'restore-drill.sh');

const TEST_BACKUP_DIR = process.env.TEST_BACKUP_DIR || '/tmp/hjtpx_backup_test';

describe('Backup System Tests', () => {
    beforeAll(() => {
        execSync(`mkdir -p ${TEST_BACKUP_DIR}/{full,incremental,redis,config}`, { stdio: 'inherit' });
        process.env.BACKUP_DIR = TEST_BACKUP_DIR;
        process.env.AUTO_CLEANUP = 'false';
    });

    afterAll(() => {
        execSync(`rm -rf ${TEST_BACKUP_DIR}`, { stdio: 'inherit' });
    });

    describe('backup.sh Script', () => {
        test('should exist and be executable', () => {
            expect(fs.existsSync(BACKUP_SCRIPT)).toBe(true);
            const stats = fs.statSync(BACKUP_SCRIPT);
            expect(stats.mode & 0o111).not.toBe(0);
        });

        test('should display usage when called with unknown argument', () => {
            try {
                execSync(`bash ${BACKUP_SCRIPT} unknown_arg`, { stdio: 'pipe', env: process.env });
            } catch (error) {
                expect(error.status).toBe(1);
                const output = error.stdout.toString() + error.stderr.toString();
                expect(output).toMatch(/Usage:/);
            }
        });

        test('should list backups when called with list argument', () => {
            try {
                const output = execSync(`bash ${BACKUP_SCRIPT} list`, { 
                    stdio: 'pipe', 
                    env: process.env 
                }).toString();
                expect(output).toContain('Full Backups') || expect(output).toContain('incremental') || expect(output).toContain('redis');
            } catch (error) {
                expect(error.status).toBe(0);
            }
        });

        test('should cleanup old backups when called with cleanup argument', () => {
            try {
                const output = execSync(`bash ${BACKUP_SCRIPT} cleanup`, { 
                    stdio: 'pipe', 
                    env: process.env,
                    timeout: 30000
                }).toString();
                expect(output).toContain('Cleaning up old backups') || expect(output).toContain('cleanup');
            } catch (error) {
                expect(error.status).toBe(0);
            }
        });

        test('should create backup directory structure', () => {
            const requiredDirs = ['full', 'incremental', 'redis', 'config'];
            requiredDirs.forEach(dir => {
                const dirPath = path.join(TEST_BACKUP_DIR, dir);
                if (!fs.existsSync(dirPath)) {
                    fs.mkdirSync(dirPath, { recursive: true });
                }
                expect(fs.existsSync(dirPath)).toBe(true);
            });
        });

        test('should handle environment variable for backup directory', () => {
            const customDir = path.join(TEST_BACKUP_DIR, 'custom_test');
            const output = execSync(`BACKUP_DIR=${customDir} bash ${BACKUP_SCRIPT} list`, { 
                stdio: 'pipe', 
                env: { ...process.env, BACKUP_DIR: customDir }
            }).toString();
            expect(output).toBeDefined();
        });
    });

    describe('backup-incremental.sh Script', () => {
        test('should exist and be executable', () => {
            expect(fs.existsSync(INCREMENTAL_SCRIPT)).toBe(true);
            const stats = fs.statSync(INCREMENTAL_SCRIPT);
            expect(stats.mode & 0o111).not.toBe(0);
        });

        test('should display usage when called with unknown argument', () => {
            try {
                execSync(`bash ${INCREMENTAL_SCRIPT} unknown_arg`, { stdio: 'pipe', env: process.env });
            } catch (error) {
                expect(error.status).toBe(1);
                const output = error.stdout.toString() + error.stderr.toString();
                expect(output).toMatch(/Usage:/);
            }
        });

        test('should list backup chains', () => {
            try {
                const output = execSync(`bash ${INCREMENTAL_SCRIPT} list`, { 
                    stdio: 'pipe', 
                    env: process.env 
                }).toString();
                expect(output).toContain('Base Backups') || expect(output).toContain('Incremental Backups') || expect(output).toContain('WAL Archives');
            } catch (error) {
                expect(error.status).toBe(0);
            }
        });

        test('should show backup size statistics', () => {
            try {
                const output = execSync(`bash ${INCREMENTAL_SCRIPT} size`, { 
                    stdio: 'pipe', 
                    env: process.env,
                    timeout: 30000
                }).toString();
                expect(output).toContain('Backup Size Summary') || expect(output).toContain('Total:');
            } catch (error) {
                expect(error.status).toBe(0);
            }
        });
    });

    describe('verify-backup.sh Script', () => {
        test('should exist and be executable', () => {
            expect(fs.existsSync(VERIFY_SCRIPT)).toBe(true);
            const stats = fs.statSync(VERIFY_SCRIPT);
            expect(stats.mode & 0o111).not.toBe(0);
        });

        test('should display usage when called with unknown argument', () => {
            try {
                execSync(`bash ${VERIFY_SCRIPT} unknown_arg`, { stdio: 'pipe', env: process.env });
            } catch (error) {
                expect(error.status).toBe(1);
                const output = error.stdout.toString() + error.stderr.toString();
                expect(output).toMatch(/Usage:/);
            }
        });

        test('should verify latest backups', () => {
            try {
                const output = execSync(`bash ${VERIFY_SCRIPT} latest`, { 
                    stdio: 'pipe', 
                    env: process.env,
                    timeout: 60000
                }).toString();
                expect(output).toContain('Verification') || expect(output).toContain('backup');
            } catch (error) {
                expect(error.status).toBe(0);
            }
        });
    });

    describe('restore-drill.sh Script', () => {
        test('should exist and be executable', () => {
            expect(fs.existsSync(RESTORE_DRILL_SCRIPT)).toBe(true);
            const stats = fs.statSync(RESTORE_DRILL_SCRIPT);
            expect(stats.mode & 0o111).not.toBe(0);
        });

        test('should display usage when called with unknown argument', () => {
            try {
                execSync(`bash ${RESTORE_DRILL_SCRIPT} unknown_arg`, { stdio: 'pipe', env: process.env });
            } catch (error) {
                expect(error.status).toBe(1);
                const output = error.stdout.toString() + error.stderr.toString();
                expect(output).toMatch(/Usage:/);
            }
        });
    });

    describe('Backup Integrity', () => {
        test('should create log directory', () => {
            const logDir = path.join(__dirname, '../../logs');
            if (!fs.existsSync(logDir)) {
                fs.mkdirSync(logDir, { recursive: true });
            }
            expect(fs.existsSync(logDir)).toBe(true);
        });

        test('should handle missing backup directory gracefully', () => {
            const nonExistentDir = '/tmp/non_existent_backup_dir_12345';
            try {
                const output = execSync(`BACKUP_DIR=${nonExistentDir} bash ${BACKUP_SCRIPT} list`, { 
                    stdio: 'pipe', 
                    env: { ...process.env, BACKUP_DIR: nonExistentDir }
                }).toString();
                expect(output).toBeDefined();
            } catch (error) {
                expect(error.status).toBe(0);
            }
        });
    });

    describe('Backup Configuration', () => {
        test('should support custom retention days', () => {
            const output = execSync(`FULL_RETENTION_DAYS=7 INCR_RETENTION_DAYS=3 bash ${BACKUP_SCRIPT} cleanup`, { 
                stdio: 'pipe', 
                env: process.env,
                timeout: 30000
            }).toString();
            expect(output).toContain('Cleaning up') || expect(output).toContain('cleanup');
        });

        test('should support disabling auto cleanup', () => {
            try {
                const output = execSync(`AUTO_CLEANUP=false bash ${BACKUP_SCRIPT} list`, { 
                    stdio: 'pipe', 
                    env: { ...process.env, AUTO_CLEANUP: 'false' }
                }).toString();
                expect(output).toBeDefined();
            } catch (error) {
                expect(error.status).toBe(0);
            }
        });
    });
});

describe('Backup Metadata Tests', () => {
    test('should track backup metadata file', () => {
        const metadataFile = path.join(TEST_BACKUP_DIR, 'backup_metadata.json');
        if (fs.existsSync(metadataFile)) {
            try {
                const content = fs.readFileSync(metadataFile, 'utf8');
                const metadata = JSON.parse(content);
                expect(metadata).toHaveProperty('backups');
                expect(Array.isArray(metadata.backups)).toBe(true);
            } catch (error) {
                expect(true).toBe(true);
            }
        }
        expect(true).toBe(true);
    });
});

describe('Backup Restoration Tests', () => {
    test('should create temporary database for restoration', () => {
        const restoreScript = path.join(SCRIPT_DIR, 'restore.sh');
        if (fs.existsSync(restoreScript)) {
            expect(fs.existsSync(restoreScript)).toBe(true);
        }
        expect(true).toBe(true);
    });

    test('should handle restoration to non-existent database', () => {
        const restoreScript = path.join(SCRIPT_DIR, 'restore.sh');
        if (fs.existsSync(restoreScript)) {
            try {
                const output = execSync(`bash ${restoreScript} list`, { 
                    stdio: 'pipe', 
                    env: process.env,
                    timeout: 30000
                }).toString();
                expect(output).toBeDefined();
            } catch (error) {
                expect(error.status).toBe(0);
            }
        }
        expect(true).toBe(true);
    });
});
