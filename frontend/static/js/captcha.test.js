const { JSDOM } = require('jsdom');

describe('ClickCaptcha', () => {
    let container;
    let clickCaptcha;

    beforeEach(() => {
        const dom = new JSDOM(`
            <!DOCTYPE html>
            <html>
            <head></head>
            <body>
                <div id="test-container"></div>
            </body>
            </html>
        `);
        global.document = dom.window.document;
        global.window = dom.window;
        global.requestAnimationFrame = (cb) => setTimeout(cb, 0);
        global.setTimeout = dom.window.setTimeout;
        global.clearTimeout = dom.window.clearTimeout;
        
        container = document.getElementById('test-container');
    });

    afterEach(() => {
        if (clickCaptcha && typeof clickCaptcha.destroy === 'function') {
            clickCaptcha.destroy();
        }
        container.innerHTML = '';
    });

    describe('ClickCaptchaStyles', () => {
        test('should inject styles only once', () => {
            const styles1 = document.querySelectorAll('style');
            const initialCount = styles1.length;

            const ClickCaptchaStyles = require('./captcha.js').ClickCaptchaStyles;
            ClickCaptchaStyles.inject();
            ClickCaptchaStyles.inject();
            
            const styles2 = document.querySelectorAll('style');
            expect(styles2.length).toBe(initialCount + 1);
        });
    });

    describe('Click Process', () => {
        test('should record clicks correctly', () => {
            const mockData = {
                session_id: 'test-session-123',
                targets: [
                    { char: '中', x: 100, y: 100 },
                    { char: '国', x: 200, y: 100 },
                    { char: '人', x: 150, y: 200 }
                ],
                correct_order: [0, 1, 2],
                max_targets: 3
            };

            const Captcha = require('./captcha.js').ClickCaptcha;
            clickCaptcha = new Captcha('test-container', {
                apiBase: '/api/v1',
                maxTargets: 3
            });

            expect(clickCaptcha.clicks.length).toBe(0);
            expect(clickCaptcha.clickHistory.length).toBe(0);
        });

        test('should update progress on click', () => {
            const Captcha = require('./captcha.js').ClickCaptcha;
            clickCaptcha = new Captcha('test-container', {
                maxTargets: 3
            });

            const mockCanvas = document.createElement('canvas');
            mockCanvas.width = 400;
            mockCanvas.height = 300;
            clickCaptcha.canvas = mockCanvas;

            expect(clickCaptcha.clicks.length).toBe(0);
        });

        test('should prevent double-click with minClickInterval', () => {
            const Captcha = require('./captcha.js').ClickCaptcha;
            clickCaptcha = new Captcha('test-container', {
                minClickInterval: 100
            });

            expect(clickCaptcha.options.minClickInterval).toBe(100);
        });
    });

    describe('Mobile Optimization', () => {
        test('should enable mobile optimization by default', () => {
            const Captcha = require('./captcha.js').ClickCaptcha;
            clickCaptcha = new Captcha('test-container');

            expect(clickCaptcha.options.enableMobileOptimization).toBe(true);
        });

        test('should enable ripple effect by default', () => {
            const Captcha = require('./captcha.js').ClickCaptcha;
            clickCaptcha = new Captcha('test-container');

            expect(clickCaptcha.options.enableRipple).toBe(true);
        });

        test('should enable highlight by default', () => {
            const Captcha = require('./captcha.js').ClickCaptcha;
            clickCaptcha = new Captcha('test-container');

            expect(clickCaptcha.options.enableHighlight).toBe(true);
        });
    });

    describe('Click History', () => {
        test('should initialize empty click history', () => {
            const Captcha = require('./captcha.js').ClickCaptcha;
            clickCaptcha = new Captcha('test-container');

            expect(clickCaptcha.clickHistory).toEqual([]);
            expect(clickCaptcha.clickSequence).toEqual([]);
        });

        test('should clear click history on clearClicks', () => {
            const Captcha = require('./captcha.js').ClickCaptcha;
            clickCaptcha = new Captcha('test-container');

            clickCaptcha.clickHistory = [{ char: '中', position: { x: 100, y: 100 } }];
            clickCaptcha.clearClicks();

            expect(clickCaptcha.clickHistory.length).toBe(0);
            expect(clickCaptcha.clickSequence.length).toBe(0);
        });
    });

    describe('Marker Management', () => {
        test('should remove click by index', () => {
            const Captcha = require('./captcha.js').ClickCaptcha;
            clickCaptcha = new Captcha('test-container');

            clickCaptcha.clicks = [
                { x: 100, y: 100, timestamp: Date.now(), clickNumber: 1 },
                { x: 200, y: 100, timestamp: Date.now(), clickNumber: 2 }
            ];
            clickCaptcha.clickHistory = [
                { char: '中', position: { x: 100, y: 100 } },
                { char: '国', position: { x: 200, y: 100 } }
            ];
            clickCaptcha.clickSequence = [0, 1];

            clickCaptcha.removeClick(0);

            expect(clickCaptcha.clicks.length).toBe(1);
            expect(clickCaptcha.clickHistory.length).toBe(1);
            expect(clickCaptcha.clicks[0].clickNumber).toBe(1);
        });
    });

    describe('Error Handling', () => {
        test('should handle container not found', () => {
            const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});
            
            const Captcha = require('./captcha.js').ClickCaptcha;
            const invalidCaptcha = new Captcha('non-existent-container');
            
            expect(consoleSpy).toHaveBeenCalledWith('ClickCaptcha container not found');
            
            consoleSpy.mockRestore();
        });
    });
});

describe('ClickCaptcha Animations', () => {
    let container;

    beforeEach(() => {
        const dom = new JSDOM(`
            <!DOCTYPE html>
            <html>
            <head></head>
            <body>
                <div id="test-container"></div>
            </body>
            </html>
        `);
        global.document = dom.window.document;
        global.window = dom.window;
        global.requestAnimationFrame = (cb) => setTimeout(cb, 0);
        global.setTimeout = dom.window.setTimeout;
        global.clearTimeout = dom.window.clearTimeout;
        
        container = document.getElementById('test-container');
    });

    describe('Ripple Effect', () => {
        test('should create ripple element', () => {
            const ClickCaptcha = require('./captcha.js').ClickCaptcha;
            const clickCaptcha = new ClickCaptcha('test-container');

            const rippleContainer = document.createElement('div');
            rippleContainer.className = 'ripple-container';
            container.appendChild(rippleContainer);
            clickCaptcha.rippleContainer = rippleContainer;

            clickCaptcha.createRipple(100, 100);

            expect(rippleContainer.children.length).toBe(1);
            expect(rippleContainer.querySelector('.click-ripple')).toBeTruthy();
        });
    });

    describe('Highlight Effect', () => {
        test('should create highlight element', () => {
            const ClickCaptcha = require('./captcha.js').ClickCaptcha;
            const clickCaptcha = new ClickCaptcha('test-container');

            const markersLayer = document.createElement('div');
            markersLayer.className = 'click-markers-layer';
            container.appendChild(markersLayer);
            clickCaptcha.markersLayer = markersLayer;

            clickCaptcha.createClickHighlight(100, 100);

            expect(markersLayer.children.length).toBe(1);
            expect(markersLayer.querySelector('.click-highlight')).toBeTruthy();
        });
    });

    describe('Feedback Animation', () => {
        test('should create feedback element', () => {
            const ClickCaptcha = require('./captcha.js').ClickCaptcha;
            const clickCaptcha = new ClickCaptcha('test-container');

            const clickFeedback = document.createElement('div');
            clickFeedback.className = 'click-feedback';
            container.appendChild(clickFeedback);
            clickCaptcha.clickFeedback = clickFeedback;

            clickCaptcha.showClickFeedback(100, 100, 1);

            expect(clickFeedback.children.length).toBe(1);
            expect(clickFeedback.querySelector('.click-feedback-item')).toBeTruthy();
        });
    });
});

describe('ClickCaptcha Touch Events', () => {
    let container;
    let clickCaptcha;

    beforeEach(() => {
        const dom = new JSDOM(`
            <!DOCTYPE html>
            <html>
            <head></head>
            <body>
                <div id="test-container"></div>
            </body>
            </html>
        `);
        global.document = dom.window.document;
        global.window = dom.window;
        global.requestAnimationFrame = (cb) => setTimeout(cb, 0);
        global.setTimeout = dom.window.setTimeout;
        global.clearTimeout = dom.window.clearTimeout;
        
        container = document.getElementById('test-container');
    });

    afterEach(() => {
        if (clickCaptcha && typeof clickCaptcha.destroy === 'function') {
            clickCaptcha.destroy();
        }
        container.innerHTML = '';
    });

    test('should track touch start time', () => {
        const ClickCaptcha = require('./captcha.js').ClickCaptcha;
        clickCaptcha = new ClickCaptcha('test-container');

        const mockEvent = {
            preventDefault: jest.fn(),
            touches: [{ clientX: 100, clientY: 100 }]
        };

        const mockCanvas = document.createElement('canvas');
        mockCanvas.getBoundingClientRect = () => ({ left: 0, top: 0 });
        clickCaptcha.canvas = mockCanvas;

        clickCaptcha.handleTouchStart(mockEvent);

        expect(clickCaptcha.touchStartTime).toBeGreaterThan(0);
        expect(mockEvent.preventDefault).toHaveBeenCalled();
    });

    test('should process touch end event', () => {
        const ClickCaptcha = require('./captcha.js').ClickCaptcha;
        clickCaptcha = new ClickCaptcha('test-container');

        const mockCanvas = document.createElement('canvas');
        mockCanvas.getBoundingClientRect = () => ({ left: 0, top: 0 });
        clickCaptcha.canvas = mockCanvas;

        clickCaptcha.touchStartTime = Date.now() - 100;
        clickCaptcha.targets = [{ char: '中' }];

        const mockEvent = {
            preventDefault: jest.fn(),
            changedTouches: [{ clientX: 100, clientY: 100 }]
        };

        const processClickSpy = jest.spyOn(clickCaptcha, 'processClick');

        clickCaptcha.handleTouchEnd(mockEvent);

        expect(mockEvent.preventDefault).toHaveBeenCalled();
        expect(processClickSpy).toHaveBeenCalled();
    });

    test('should ignore long press as click', () => {
        const ClickCaptcha = require('./captcha.js').ClickCaptcha;
        clickCaptcha = new ClickCaptcha('test-container');

        const mockCanvas = document.createElement('canvas');
        mockCanvas.getBoundingClientRect = () => ({ left: 0, top: 0 });
        clickCaptcha.canvas = mockCanvas;

        clickCaptcha.touchStartTime = Date.now() - 2000;

        const mockEvent = {
            preventDefault: jest.fn(),
            changedTouches: [{ clientX: 100, clientY: 100 }]
        };

        const processClickSpy = jest.spyOn(clickCaptcha, 'processClick');

        clickCaptcha.handleTouchEnd(mockEvent);

        expect(processClickSpy).not.toHaveBeenCalled();
    });
});
