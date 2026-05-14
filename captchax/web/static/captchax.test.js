/**
 * CaptchaX SDK 测试套件
 * @version 1.1.0
 * @license MIT
 */
(function(global) {
    'use strict';

    const CaptchaXTest = {
        version: '1.1.0',
        passed: 0,
        failed: 0,
        results: [],
        mockServer: null,
        mockServerUrl: 'http://localhost:8765'
    };

    function assert(condition, message) {
        const result = {
            passed: condition,
            message: message,
            timestamp: new Date().toISOString()
        };
        CaptchaXTest.results.push(result);
        if (condition) {
            CaptchaXTest.passed++;
            console.log(`✓ ${message}`);
        } else {
            CaptchaXTest.failed++;
            console.error(`✗ ${message}`);
        }
        return condition;
    }

    function assertEqual(actual, expected, message) {
        const condition = actual === expected;
        const result = {
            passed: condition,
            message: `${message} (期望: ${expected}, 实际: ${actual})`,
            timestamp: new Date().toISOString()
        };
        CaptchaXTest.results.push(result);
        if (condition) {
            CaptchaXTest.passed++;
            console.log(`✓ ${message}`);
        } else {
            CaptchaXTest.failed++;
            console.error(`✗ ${message} (期望: ${expected}, 实际: ${actual})`);
        }
        return condition;
    }

    function assertDeepEqual(actual, expected, message) {
        const actualStr = JSON.stringify(actual);
        const expectedStr = JSON.stringify(expected);
        const condition = actualStr === expectedStr;
        const result = {
            passed: condition,
            message: `${message}`,
            timestamp: new Date().toISOString()
        };
        CaptchaXTest.results.push(result);
        if (condition) {
            CaptchaXTest.passed++;
            console.log(`✓ ${message}`);
        } else {
            CaptchaXTest.failed++;
            console.error(`✗ ${message}`);
            console.error(`  期望: ${expectedStr}`);
            console.error(`  实际: ${actualStr}`);
        }
        return condition;
    }

    function createMockContainer() {
        const container = document.createElement('div');
        container.className = 'test-container';
        container.style.cssText = 'position: fixed; top: -9999px; left: -9999px; width: 400px;';
        document.body.appendChild(container);
        return container;
    }

    function removeMockContainer(container) {
        if (container && container.parentNode) {
            container.parentNode.removeChild(container);
        }
    }

    function createMockServer() {
        CaptchaXTest.mockServer = {
            requests: [],
            responses: {
                slider: {
                    id: 'test-slider-captcha-id',
                    background_b64: 'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==',
                    slider_b64: 'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==',
                    target_x: 120,
                    target_y: 50
                },
                click: {
                    id: 'test-click-captcha-id',
                    image: 'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==',
                    target_chars: ['A', 'B', 'C', 'D'],
                    targets: [
                        { x: 50, y: 50, char: 'A' },
                        { x: 150, y: 50, char: 'B' },
                        { x: 50, y: 150, char: 'C' },
                        { x: 150, y: 150, char: 'D' }
                    ]
                },
                rotate: {
                    id: 'test-rotate-captcha-id',
                    image: 'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==',
                    target_angle: 45
                }
            }
        };

        const originalFetch = window.fetch;
        const mockFetch = function(url, options) {
            const urlStr = typeof url === 'string' ? url : url.url || String(url);
            const requestData = {
                url: urlStr,
                method: options?.method || 'GET',
                body: options?.body,
                timestamp: Date.now()
            };
            CaptchaXTest.mockServer.requests.push(requestData);

            let mockResponse;
            if (urlStr.includes('/api/captcha/slider') && !urlStr.includes('/verify')) {
                mockResponse = CaptchaXTest.mockServer.responses.slider;
            } else if (urlStr.includes('/api/captcha/slider/verify')) {
                mockResponse = { success: true, token: 'test-slider-token-' + Date.now() };
            } else if (urlStr.includes('/api/captcha/click') && !urlStr.includes('/verify')) {
                mockResponse = CaptchaXTest.mockServer.responses.click;
            } else if (urlStr.includes('/api/captcha/click/verify')) {
                mockResponse = { success: true, token: 'test-click-token-' + Date.now() };
            } else if (urlStr.includes('/api/captcha/rotate') && !urlStr.includes('/verify')) {
                mockResponse = CaptchaXTest.mockServer.responses.rotate;
            } else if (urlStr.includes('/api/captcha/rotate/verify')) {
                mockResponse = { success: true, token: 'test-rotate-token-' + Date.now() };
            } else {
                mockResponse = { error: 'Unknown endpoint' };
            }

            return Promise.resolve({
                ok: true,
                status: 200,
                json: function() {
                    return Promise.resolve(mockResponse);
                }
            });
        };

        window.fetch = mockFetch;
        CaptchaXTest._originalFetch = originalFetch;
    }

    function destroyMockServer() {
        if (CaptchaXTest._originalFetch) {
            window.fetch = CaptchaXTest._originalFetch;
            delete CaptchaXTest._originalFetch;
        }
        CaptchaXTest.mockServer = null;
    }

    CaptchaXTest.run = async function() {
        console.log('========================================');
        console.log('CaptchaX SDK 测试套件 v' + CaptchaXTest.version);
        console.log('========================================\n');

        createMockServer();

        await CaptchaXTest.testInit();
        await CaptchaXTest.testCreate();
        await CaptchaXTest.testSliderVerification();
        await CaptchaXTest.testClickVerification();
        await CaptchaXTest.testRotateVerification();
        await CaptchaXTest.testDarkMode();
        await CaptchaXTest.testAccessibility();
        await CaptchaXTest.testCallbacks();
        await CaptchaXTest.testDestruction();

        destroyMockServer();

        console.log('\n========================================');
        console.log('测试结果:');
        console.log(`  通过: ${CaptchaXTest.passed}`);
        console.log(`  失败: ${CaptchaXTest.failed}`);
        console.log(`  总计: ${CaptchaXTest.passed + CaptchaXTest.failed}`);
        console.log('========================================');

        return {
            passed: CaptchaXTest.passed,
            failed: CaptchaXTest.failed,
            results: CaptchaXTest.results
        };
    };

    CaptchaXTest.testInit = async function() {
        console.log('\n--- 测试初始化 ---');

        CaptchaXTest.passed = 0;
        CaptchaXTest.failed = 0;

        const originalCaptchaX = global.CaptchaX;

        try {
            await CaptchaXTest.testSDKExists();
            await CaptchaXTest.testInitWithOptions();
            await CaptchaXTest.testInitAutoServerUrl();
        } catch (e) {
            console.error('初始化测试出错:', e);
        }
    };

    CaptchaXTest.testSDKExists = async function() {
        assert(typeof global.CaptchaX !== 'undefined', 'CaptchaX 全局对象存在');
        assert(typeof global.CaptchaX.init === 'function', 'CaptchaX.init 方法存在');
        assert(typeof global.CaptchaX.create === 'function', 'CaptchaX.create 方法存在');
        assert(typeof global.CaptchaX.verify === 'function', 'CaptchaX.verify 方法存在');
        assert(typeof global.CaptchaX.onSuccess === 'function', 'CaptchaX.onSuccess 方法存在');
        assert(typeof global.CaptchaX.onError === 'function', 'CaptchaX.onError 方法存在');
        assert(typeof global.CaptchaX.destroy === 'function', 'CaptchaX.destroy 方法存在');
    };

    CaptchaXTest.testInitWithOptions = async function() {
        const config = {
            serverUrl: 'https://test.example.com',
            appId: 'test-app-id',
            timeout: 15000
        };

        const captchaX = await global.CaptchaX.init(config);

        assertEqual(captchaX.config.serverUrl, 'https://test.example.com', 'serverUrl 配置正确');
        assertEqual(captchaX.config.appId, 'test-app-id', 'appId 配置正确');
        assertEqual(captchaX.config.timeout, 15000, 'timeout 配置正确');
        assertEqual(captchaX.state.ready, true, 'SDK 状态已就绪');
    };

    CaptchaXTest.testInitAutoServerUrl = async function() {
        const scripts = document.getElementsByTagName('script');
        let captchaxScriptFound = false;
        for (let i = 0; i < scripts.length; i++) {
            if (scripts[i].src && scripts[i].src.includes('captchax')) {
                captchaxScriptFound = true;
                break;
            }
        }

        if (captchaxScriptFound) {
            console.log('  (跳过自动 URL 检测测试 - 已在初始化中测试)');
        }
    };

    CaptchaXTest.testCreate = async function() {
        console.log('\n--- 测试创建组件 ---');

        const container = createMockContainer();

        try {
            const instance = global.CaptchaX.create({
                container: container,
                type: 'slider'
            });

            assert(instance !== null, 'create 返回实例对象');
            assert(instance.id !== undefined, '实例有唯一 ID');
            assert(instance.container === container, '实例绑定到正确容器');
            assert(instance.config !== undefined, '实例有配置对象');
            assertEqual(instance.state.captchaType, 'slider', '实例类型正确');

            await CaptchaXTest.waitForElement(container, '.captchax-container');
            assert(hasClass(container, 'captchax-container'), '容器添加了 captchax-container 类');

            global.CaptchaX.destroy();

            setTimeout(() => {
                assert(container.innerHTML === '', 'destroy 后容器内容清空');
                removeMockContainer(container);
            }, 2000);
        } catch (e) {
            console.error('创建测试出错:', e);
            removeMockContainer(container);
        }
    };

    CaptchaXTest.testSliderVerification = async function() {
        console.log('\n--- 测试滑块验证 ---');

        const container = createMockContainer();

        try {
            const instance = global.CaptchaX.create({
                container: container,
                type: 'slider'
            });

            await CaptchaXTest.waitForElement(container, '.captchax-slider-thumb');

            const sliderThumb = container.querySelector('.captchax-slider-thumb');
            assert(sliderThumb !== null, '滑块元素存在');

            const sliderTrack = container.querySelector('.captchax-slider-track');
            assert(sliderTrack !== null, '滑轨元素存在');

            const sliderBg = container.querySelector('.captchax-slider-bg');
            assert(sliderBg !== null, '背景图片元素存在');
            assert(sliderBg.src.includes('data:image/png;base64,'), '背景图片使用 base64');

            const progress = container.querySelector('.captchax-slider-progress');
            assert(progress !== null, '进度条元素存在');

            const ariaLabel = sliderThumb.getAttribute('aria-label');
            assert(ariaLabel !== null, '滑块有 ARIA 标签');
            assert(ariaLabel.includes('验证'), 'ARIA 标签描述正确');

            global.CaptchaX.destroy();

            setTimeout(() => {
                removeMockContainer(container);
            }, 2000);
        } catch (e) {
            console.error('滑块验证测试出错:', e);
            removeMockContainer(container);
        }
    };

    CaptchaXTest.testClickVerification = async function() {
        console.log('\n--- 测试点选验证 ---');

        const container = createMockContainer();

        try {
            const instance = global.CaptchaX.create({
                container: container,
                type: 'click'
            });

            await CaptchaXTest.waitForElement(container, '.captchax-click-img');

            const clickImage = container.querySelector('.captchax-click-img');
            assert(clickImage !== null, '点选图片元素存在');

            const instruction = container.querySelector('.captchax-click-instruction');
            assert(instruction !== null, '指令元素存在');

            const targetChars = instruction.querySelector('.captchax-target-chars');
            assert(targetChars !== null, '目标字符元素存在');
            assert(targetChars.textContent.includes('A'), '目标字符包含 A');
            assert(targetChars.textContent.includes('B'), '目标字符包含 B');

            const refreshBtn = instruction.querySelector('.captchax-refresh-btn');
            assert(refreshBtn !== null, '刷新按钮存在');
            assert(refreshBtn.getAttribute('aria-label') !== null, '刷新按钮有 ARIA 标签');

            const progressBar = container.querySelector('.captchax-progress-bar');
            assert(progressBar !== null, '进度条元素存在');

            global.CaptchaX.destroy();

            setTimeout(() => {
                removeMockContainer(container);
            }, 2000);
        } catch (e) {
            console.error('点选验证测试出错:', e);
            removeMockContainer(container);
        }
    };

    CaptchaXTest.testRotateVerification = async function() {
        console.log('\n--- 测试旋转验证 ---');

        const container = createMockContainer();

        try {
            const instance = global.CaptchaX.create({
                container: container,
                type: 'rotate'
            });

            await CaptchaXTest.waitForElement(container, '.captchax-rotate-img');

            const rotateImage = container.querySelector('.captchax-rotate-img');
            assert(rotateImage !== null, '旋转图片元素存在');

            const rotateSlider = container.querySelector('.captchax-rotate-slider');
            assert(rotateSlider !== null, '旋转滑块存在');

            const rotateHandle = container.querySelector('.captchax-rotate-handle');
            assert(rotateHandle !== null, '旋转手柄存在');

            const verifyBtn = container.querySelector('.captchax-rotate-verify-btn');
            assert(verifyBtn !== null, '确认按钮存在');
            assertEqual(verifyBtn.textContent.trim(), '确认旋转', '确认按钮文本正确');

            const valueDisplay = container.querySelector('.captchax-rotate-value');
            assert(valueDisplay !== null, '角度值显示存在');
            assertEqual(valueDisplay.textContent, '0', '初始角度为 0');

            global.CaptchaX.destroy();

            setTimeout(() => {
                removeMockContainer(container);
            }, 2000);
        } catch (e) {
            console.error('旋转验证测试出错:', e);
            removeMockContainer(container);
        }
    };

    CaptchaXTest.testDarkMode = async function() {
        console.log('\n--- 测试深色模式 ---');

        const container = createMockContainer();

        try {
            const instance = global.CaptchaX.create({
                container: container,
                type: 'slider',
                theme: 'dark'
            });

            await CaptchaXTest.waitForElement(container, '.captchax-container');

            assert(hasClass(container, 'captchax-dark'), '容器添加了 dark 类');

            const header = container.querySelector('.captchax-header');
            assert(header !== null, '头部元素存在');

            global.CaptchaX.destroy();

            setTimeout(() => {
                removeMockContainer(container);
            }, 2000);
        } catch (e) {
            console.error('深色模式测试出错:', e);
            removeMockContainer(container);
        }
    };

    CaptchaXTest.testAccessibility = async function() {
        console.log('\n--- 测试无障碍访问 ---');

        const container = createMockContainer();

        try {
            const instance = global.CaptchaX.create({
                container: container,
                type: 'slider'
            });

            await CaptchaXTest.waitForElement(container, '.captchax-slider-thumb');

            const sliderThumb = container.querySelector('.captchax-slider-thumb');
            assert(sliderThumb.getAttribute('role') === 'slider', '滑块有 role 属性');
            assert(sliderThumb.getAttribute('tabindex') === '0', '滑块可键盘聚焦');
            assert(sliderThumb.getAttribute('aria-label') !== null, '滑块有 aria-label');
            assert(sliderThumb.getAttribute('aria-valuemin') === '0', '滑块有 aria-valuemin');
            assert(sliderThumb.getAttribute('aria-valuemax') === '100', '滑块有 aria-valuemax');

            const message = container.querySelector('.captchax-message');
            assert(message.getAttribute('role') === 'alert', '消息有 role="alert"');

            const closeBtn = container.querySelector('.captchax-close');
            assert(closeBtn.getAttribute('aria-label') !== null, '关闭按钮有 aria-label');

            global.CaptchaX.destroy();

            setTimeout(() => {
                removeMockContainer(container);
            }, 2000);
        } catch (e) {
            console.error('无障碍访问测试出错:', e);
            removeMockContainer(container);
        }
    };

    CaptchaXTest.testCallbacks = async function() {
        console.log('\n--- 测试回调函数 ---');

        const container = createMockContainer();

        try {
            let successCalled = false;
            let errorCalled = false;

            global.CaptchaX.onSuccess(function(result) {
                successCalled = true;
            });

            global.CaptchaX.onError(function(error) {
                errorCalled = true;
            });

            assert(typeof global.CaptchaX.callbacks.onSuccess === 'object', 'onSuccess 回调数组存在');
            assert(typeof global.CaptchaX.callbacks.onError === 'object', 'onError 回调数组存在');
            assert(global.CaptchaX.callbacks.onSuccess.length > 0, 'onSuccess 回调已注册');
            assert(global.CaptchaX.callbacks.onError.length > 0, 'onError 回调已注册');

            global.CaptchaX.offSuccess();
            global.CaptchaX.offError();

            assertEqual(global.CaptchaX.callbacks.onSuccess.length, 0, 'offSuccess 清除所有回调');
            assertEqual(global.CaptchaX.callbacks.onError.length, 0, 'offError 清除所有回调');

            global.CaptchaX.destroy();

            setTimeout(() => {
                removeMockContainer(container);
            }, 2000);
        } catch (e) {
            console.error('回调测试出错:', e);
            removeMockContainer(container);
        }
    };

    CaptchaXTest.testDestruction = async function() {
        console.log('\n--- 测试销毁功能 ---');

        const container = createMockContainer();

        try {
            const instance = global.CaptchaX.create({
                container: container,
                type: 'slider'
            });

            await CaptchaXTest.waitForElement(container, '.captchax-container');

            assertEqual(global.CaptchaX.instances.size, 1, '创建后有一个实例');

            global.CaptchaX.destroy();

            setTimeout(() => {
                assertEqual(global.CaptchaX.instances.size, 0, '销毁后没有实例');
                assertEqual(instance.state.destroyed, true, '实例状态标记为已销毁');

                removeMockContainer(container);
            }, 2000);
        } catch (e) {
            console.error('销毁测试出错:', e);
            removeMockContainer(container);
        }
    };

    CaptchaXTest.waitForElement = function(container, selector, timeout = 5000) {
        return new Promise((resolve, reject) => {
            const element = container.querySelector(selector);
            if (element) {
                resolve(element);
                return;
            }

            const observer = new MutationObserver(() => {
                const element = container.querySelector(selector);
                if (element) {
                    observer.disconnect();
                    resolve(element);
                }
            });

            observer.observe(container, { childList: true, subtree: true });

            setTimeout(() => {
                observer.disconnect();
                reject(new Error(`等待元素 ${selector} 超时`));
            }, timeout);
        });
    };

    function hasClass(element, className) {
        return element && element.classList && element.classList.contains(className);
    }

    function sleep(ms) {
        return new Promise(resolve => setTimeout(resolve, ms));
    }

    if (typeof module !== 'undefined' && module.exports) {
        module.exports = CaptchaXTest;
    } else {
        global.CaptchaXTest = CaptchaXTest;
    }

})(typeof window !== 'undefined' ? window : this);
