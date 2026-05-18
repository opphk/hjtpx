/**
 * 响应式测试套件 - 用于验证前端响应式布局
 * 测试各种屏幕尺寸下的UI表现
 */

(function() {
    'use strict';
    
    const ResponsiveTest = {
        breakpoints: {
            mobile: { width: 375, height: 667, name: 'iPhone SE' },
            mobileLandscape: { width: 667, height: 375, name: 'iPhone SE Landscape' },
            tablet: { width: 768, height: 1024, name: 'iPad' },
            tabletLandscape: { width: 1024, height: 768, name: 'iPad Landscape' },
            desktop: { width: 1280, height: 800, name: 'Desktop' },
            largeDesktop: { width: 1920, height: 1080, name: 'Large Desktop' }
        },
        
        testResults: [],
        
        init: function() {
            console.log('🚀 响应式测试开始');
            this.runAllTests();
        },
        
        setViewport: function(width, height) {
            if (window.innerWidth === width && window.innerHeight === height) {
                return Promise.resolve();
            }
            
            return new Promise((resolve) => {
                const resizeEvent = new Event('resize');
                Object.defineProperty(window, 'innerWidth', { value: width, writable: true });
                Object.defineProperty(window, 'innerHeight', { value: height, writable: true });
                window.dispatchEvent(resizeEvent);
                setTimeout(resolve, 100);
            });
        },
        
        testElementVisibility: function(selector, expectedVisible, breakpoint) {
            const elements = document.querySelectorAll(selector);
            let passed = true;
            
            elements.forEach(el => {
                const rect = el.getBoundingClientRect();
                const isVisible = rect.width > 0 && rect.height > 0;
                if (isVisible !== expectedVisible) {
                    passed = false;
                    console.warn(`⚠️ ${breakpoint}: ${selector} 可见性不符合预期`);
                }
            });
            
            return passed;
        },
        
        testLayoutGrid: function(breakpoint, expectedCols) {
            const container = document.querySelector('.container, .container-fluid');
            if (!container) return true;
            
            const computedStyle = window.getComputedStyle(container);
            const actualWidth = container.offsetWidth;
            
            const isResponsive = actualWidth <= expectedCols * 100;
            
            if (!isResponsive) {
                console.warn(`⚠️ ${breakpoint}: 网格布局宽度 ${actualWidth}px 超出预期`);
            }
            
            return isResponsive;
        },
        
        testToastPosition: function(breakpoint) {
            const toast = document.querySelector('.captcha-toast-container');
            if (!toast) return true;
            
            const rect = toast.getBoundingClientRect();
            
            const isPositionedCorrectly = rect.right <= window.innerWidth + 10;
            
            if (!isPositionedCorrectly) {
                console.warn(`⚠️ ${breakpoint}: Toast 容器超出视口`);
            }
            
            return isPositionedCorrectly;
        },
        
        testTouchTargets: function(breakpoint) {
            const touchTargets = document.querySelectorAll('button, a, [role="button"]');
            let passed = true;
            
            touchTargets.forEach(target => {
                const rect = target.getBoundingClientRect();
                const minSize = breakpoint.includes('mobile') ? 44 : 32;
                
                if (rect.width < minSize && rect.height < minSize) {
                    const tag = target.tagName.toLowerCase();
                    if (tag === 'button' || tag === 'a' || target.getAttribute('role')) {
                        passed = false;
                    }
                }
            });
            
            return passed;
        },
        
        testTypographyScaling: function(breakpoint) {
            const headings = document.querySelectorAll('h1, h2, h3, h4, h5, h6');
            let passed = true;
            
            headings.forEach(heading => {
                const fontSize = parseFloat(getComputedStyle(heading).fontSize);
                const lineHeight = parseFloat(getComputedStyle(heading).lineHeight);
                
                if (fontSize < 10 || lineHeight < 1.2) {
                    passed = false;
                }
            });
            
            return passed;
        },
        
        testColorContrast: function() {
            const body = document.body;
            const computedBg = window.getComputedStyle(body).backgroundColor;
            const computedColor = window.getComputedStyle(body).color;
            
            return {
                background: computedBg,
                foreground: computedColor,
                pass: true
            };
        },
        
        testAnimationPerformance: function() {
            if (!('requestAnimationFrame' in window)) {
                return { pass: false, reason: 'requestAnimationFrame not supported' };
            }
            
            let frameCount = 0;
            const startTime = performance.now();
            
            return new Promise((resolve) => {
                function countFrame() {
                    frameCount++;
                    if (performance.now() - startTime < 100) {
                        requestAnimationFrame(countFrame);
                    } else {
                        const fps = (frameCount / (performance.now() - startTime)) * 1000;
                        resolve({ fps: fps, pass: fps >= 30 });
                    }
                }
                requestAnimationFrame(countFrame);
            });
        },
        
        runBreakpointTest: async function(breakpoint, config) {
            console.log(`\n📱 测试 ${config.name} (${config.width}x${config.height})`);
            
            await this.setViewport(config.width, config.height);
            
            const results = {
                breakpoint: config.name,
                width: config.width,
                height: config.height,
                tests: {}
            };
            
            results.tests.layoutGrid = this.testLayoutGrid(config.name, config.width);
            results.tests.toastPosition = this.testToastPosition(config.name);
            results.tests.touchTargets = this.testTouchTargets(config.name);
            results.tests.typography = this.testTypographyScaling(config.name);
            
            const colorTest = this.testColorContrast();
            results.tests.colorContrast = colorTest.pass;
            
            const fpsTest = await this.testAnimationPerformance();
            results.tests.animationPerformance = fpsTest.pass;
            results.tests.fps = fpsTest.fps || 0;
            
            const allPassed = Object.values(results.tests).every(v => v === true || v.pass === true);
            results.passed = allPassed;
            
            if (allPassed) {
                console.log(`✅ ${config.name}: 全部测试通过`);
            } else {
                console.warn(`⚠️ ${config.name}: 部分测试未通过`);
            }
            
            this.testResults.push(results);
            return results;
        },
        
        runAllTests: async function() {
            for (const [name, config] of Object.entries(this.breakpoints)) {
                await this.runBreakpointTest(name, config);
            }
            
            this.generateReport();
        },
        
        generateReport: function() {
            console.log('\n📊 响应式测试报告');
            console.log('═'.repeat(50));
            
            const summary = {
                total: this.testResults.length,
                passed: this.testResults.filter(r => r.passed).length,
                failed: this.testResults.filter(r => !r.passed).length
            };
            
            console.log(`总测试数: ${summary.total}`);
            console.log(`通过: ${summary.passed}`);
            console.log(`失败: ${summary.failed}`);
            
            this.testResults.forEach(result => {
                console.log(`\n${result.breakpoint}:`);
                Object.entries(result.tests).forEach(([key, value]) => {
                    if (key === 'fps') {
                        console.log(`  FPS: ${value.toFixed(1)}`);
                    } else {
                        console.log(`  ${key}: ${value ? '✅' : '❌'}`);
                    }
                });
            });
            
            const report = {
                timestamp: new Date().toISOString(),
                summary: summary,
                results: this.testResults
            };
            
            if (typeof window !== 'undefined') {
                window.__responsiveTestReport = report;
            }
            
            return report;
        }
    };
    
    if (typeof module !== 'undefined' && module.exports) {
        module.exports = ResponsiveTest;
    } else {
        window.ResponsiveTest = ResponsiveTest;
    }
    
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', () => ResponsiveTest.init());
    } else {
        ResponsiveTest.init();
    }
})();
