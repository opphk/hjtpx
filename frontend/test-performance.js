const fs = require('fs');
const path = require('path');

function checkPerformanceOptimizations() {
    console.log('🔍 Checking Frontend Performance Optimizations...\n');

    const frontendDir = path.join(__dirname, '../frontend');
    const results = {
        passed: 0,
        failed: 0,
        warnings: []
    };

    const checks = [
        {
            name: 'Performance Optimizer Exists',
            path: path.join(frontendDir, 'static/js/performance-optimizer.js'),
            required: true
        },
        {
            name: 'Mobile Performance Optimizer Exists',
            path: path.join(frontendDir, 'static/js/mobile-performance-optimizer.js'),
            required: true
        },
        {
            name: 'CSS Files Exist',
            path: frontendDir + '/static/css',
            type: 'directory'
        },
        {
            name: 'Home Template Uses Bootstrap 5',
            template: path.join(frontendDir, 'templates/home.html'),
            search: 'cdn.bootcdn.net/ajax/libs/twitter-bootstrap/5'
        },
        {
            name: 'Async CSS Loading',
            template: path.join(frontendDir, 'templates/home.html'),
            search: 'media="print" onload="this.media=\'all\'"'
        }
    ];

    checks.forEach(check => {
        try {
            if (check.path) {
                if (check.type === 'directory') {
                    if (fs.existsSync(check.path)) {
                        console.log(`✅ ${check.name}`);
                        results.passed++;
                    } else {
                        console.log(`❌ ${check.name}`);
                        results.failed++;
                    }
                } else {
                    if (fs.existsSync(check.path)) {
                        console.log(`✅ ${check.name}`);
                        results.passed++;

                        if (check.search) {
                            const content = fs.readFileSync(check.path, 'utf8');
                            if (content.includes(check.search)) {
                                console.log(`   ✓ Feature found in file`);
                            } else {
                                console.log(`   ⚠ Search pattern not found`);
                                results.warnings.push(`${check.name}: Search pattern not found`);
                            }
                        }
                    } else {
                        console.log(`❌ ${check.name}`);
                        results.failed++;
                    }
                }
            } else if (check.template) {
                if (fs.existsSync(check.template)) {
                    const content = fs.readFileSync(check.template, 'utf8');
                    if (content.includes(check.search)) {
                        console.log(`✅ ${check.name}`);
                        results.passed++;
                    } else {
                        console.log(`⚠ ${check.name} - Pattern not found`);
                        results.warnings.push(`${check.name}: Pattern not found`);
                    }
                }
            }
        } catch (error) {
            console.log(`❌ ${check.name} - Error: ${error.message}`);
            results.failed++;
        }
    });

    console.log('\n📊 Results:');
    console.log(`✅ Passed: ${results.passed}`);
    console.log(`❌ Failed: ${results.failed}`);
    console.log(`⚠ Warnings: ${results.warnings.length}`);

    if (results.warnings.length > 0) {
        console.log('\n⚠ Warnings:');
        results.warnings.forEach(warning => console.log(`  - ${warning}`));
    }

    console.log('\n✨ Frontend Performance Optimization Check Complete!');

    return results.failed === 0;
}

if (require.main === module) {
    const success = checkPerformanceOptimizations();
    process.exit(success ? 0 : 1);
}

module.exports = { checkPerformanceOptimizations };
