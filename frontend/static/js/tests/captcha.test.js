describe('CaptchaComponent', () => {
    describe('Constructor and Initialization', () => {
        test('should create instance with valid container ID', () => {
            const container = document.createElement('div');
            container.id = 'captcha-test';
            document.body.appendChild(container);

            const captcha = new Captcha('captcha-test');

            expect(captcha.container).toBe(container);
            expect(captcha.sessionId).toBeNull();

            document.body.removeChild(container);
        });

        test('should handle invalid container ID gracefully', () => {
            const consoleSpy = jest.spyOn(console, 'error').mockImplementation();

            const captcha = new Captcha('non-existent-id');

            expect(consoleSpy).toHaveBeenCalledWith('Captcha container not found');

            consoleSpy.mockRestore();
        });

        test('should set default options correctly', () => {
            const container = document.createElement('div');
            container.id = 'captcha-test';
            document.body.appendChild(container);

            const captcha = new Captcha('captcha-test');

            expect(captcha.options.apiBase).toBe('/api/v1');
            expect(captcha.options.type).toBe('slider');
            expect(captcha.options.timeout).toBe(60);
            expect(captcha.options.imageCount).toBe(6);
            expect(captcha.options.gridColumns).toBe(3);
            expect(captcha.options.gridRows).toBe(2);

            document.body.removeChild(container);
        });

        test('should merge custom options with defaults', () => {
            const container = document.createElement('div');
            container.id = 'captcha-test';
            document.body.appendChild(container);

            const captcha = new Captcha('captcha-test', {
                apiBase: '/custom/api',
                type: 'click',
                timeout: 120
            });

            expect(captcha.options.apiBase).toBe('/custom/api');
            expect(captcha.options.type).toBe('click');
            expect(captcha.options.timeout).toBe(120);
            expect(captcha.options.gridColumns).toBe(3);

            document.body.removeChild(container);
        });

        test('should initialize slider state correctly', () => {
            const container = document.createElement('div');
            container.id = 'captcha-test';
            document.body.appendChild(container);

            const captcha = new Captcha('captcha-test');

            expect(captcha.sliderState.isDragging).toBe(false);
            expect(captcha.sliderState.startX).toBe(0);
            expect(captcha.sliderState.currentX).toBe(0);
            expect(captcha.sliderState.maxX).toBe(0);
            expect(captcha.sliderState.puzzleY).toBe(0);

            document.body.removeChild(container);
        });

        test('should initialize click state correctly', () => {
            const container = document.createElement('div');
            container.id = 'captcha-test';
            document.body.appendChild(container);

            const captcha = new Captcha('captcha-test');

            expect(captcha.clickState.selectedPoints).toEqual([]);
            expect(captcha.clickState.maxPoints).toBe(3);
            expect(captcha.clickState.clickHistory).toEqual([]);
            expect(captcha.clickState.startTime).toBeNull();

            document.body.removeChild(container);
        });
    });

    describe('Slider Validation', () => {
        test('should validate slider position within tolerance', () => {
            const container = document.createElement('div');
            container.id = 'captcha-test';
            document.body.appendChild(container);

            const captcha = new Captcha('captcha-test');
            captcha.sliderState.currentX = 195;

            const isWithinTolerance = Math.abs(captcha.sliderState.currentX - 200) <= 10;
            expect(isWithinTolerance).toBe(true);

            document.body.removeChild(container);
        });

        test('should reject slider position outside tolerance', () => {
            const container = document.createElement('div');
            container.id = 'captcha-test';
            document.body.appendChild(container);

            const captcha = new Captcha('captcha-test');
            captcha.sliderState.currentX = 180;

            const isWithinTolerance = Math.abs(captcha.sliderState.currentX - 200) <= 10;
            expect(isWithinTolerance).toBe(false);

            document.body.removeChild(container);
        });
    });

    describe('Click Point Management', () => {
        test('should add click point correctly', () => {
            const container = document.createElement('div');
            container.id = 'captcha-test';
            document.body.appendChild(container);

            const captcha = new Captcha('captcha-test');

            const point = {
                x: 100,
                y: 200,
                imageIndex: 0,
                timestamp: Date.now()
            };

            captcha.clickState.selectedPoints.push(point);
            captcha.clickState.clickHistory.push(point);

            expect(captcha.clickState.selectedPoints).toHaveLength(1);
            expect(captcha.clickState.clickHistory).toHaveLength(1);

            document.body.removeChild(container);
        });

        test('should limit click points to maxPoints', () => {
            const container = document.createElement('div');
            container.id = 'captcha-test';
            document.body.appendChild(container);

            const captcha = new Captcha('captcha-test');

            for (let i = 0; i < 5; i++) {
                captcha.clickState.selectedPoints.push({
                    x: i * 10,
                    y: i * 10,
                    imageIndex: 0,
                    timestamp: Date.now()
                });
            }

            expect(captcha.clickState.selectedPoints.length).toBeLessThanOrEqual(captcha.clickState.maxPoints);

            document.body.removeChild(container);
        });

        test('should clear click points correctly', () => {
            const container = document.createElement('div');
            container.id = 'captcha-test';
            document.body.appendChild(container);

            const captcha = new Captcha('captcha-test');

            captcha.clickState.selectedPoints = [
                { x: 100, y: 200 },
                { x: 150, y: 250 }
            ];

            captcha.clearClickPoints();

            expect(captcha.clickState.selectedPoints).toEqual([]);

            document.body.removeChild(container);
        });

        test('should undo last click correctly', () => {
            const container = document.createElement('div');
            container.id = 'captcha-test';
            document.body.appendChild(container);

            const captcha = new Captcha('captcha-test');

            captcha.clickState.selectedPoints = [
                { x: 100, y: 200 },
                { x: 150, y: 250 },
                { x: 200, y: 300 }
            ];

            captcha.undoLastClick();

            expect(captcha.clickState.selectedPoints).toHaveLength(2);

            document.body.removeChild(container);
        });
    });

    describe('Session ID Generation', () => {
        test('should handle session ID from API response', () => {
            const container = document.createElement('div');
            container.id = 'captcha-test';
            document.body.appendChild(container);

            const captcha = new Captcha('captcha-test');
            const mockResponse = { session_id: 'sess_123456', image_url: 'data:image/png;base64,...' };

            captcha.sessionId = mockResponse.session_id;

            expect(captcha.sessionId).toBe('sess_123456');

            document.body.removeChild(container);
        });

        test('should generate demo session ID when API unavailable', () => {
            const container = document.createElement('div');
            container.id = 'captcha-test';
            document.body.appendChild(container);

            const captcha = new Captcha('captcha-test');
            const demoSessionId = 'demo_' + Date.now();

            captcha.sessionId = demoSessionId;

            expect(captcha.sessionId).toMatch(/^demo_\d+$/);

            document.body.removeChild(container);
        });
    });

    describe('Progress Update', () => {
        test('should calculate progress percentage correctly', () => {
            const container = document.createElement('div');
            container.id = 'captcha-test';
            document.body.appendChild(container);

            const captcha = new Captcha('captcha-test');

            captcha.clickState.selectedPoints = [
                { x: 100, y: 200 },
                { x: 150, y: 250 },
                { x: 200, y: 300 }
            ];

            const selected = captcha.clickState.selectedPoints.length;
            const total = captcha.clickState.maxPoints;
            const percentage = (selected / total) * 100;

            expect(percentage).toBe(100);

            document.body.removeChild(container);
        });

        test('should update progress when points change', () => {
            const container = document.createElement('div');
            container.id = 'captcha-test';
            document.body.appendChild(container);

            const captcha = new Captcha('captcha-test');

            captcha.clickState.selectedPoints = [{ x: 100, y: 200 }];
            const percentage1 = (captcha.clickState.selectedPoints.length / captcha.clickState.maxPoints) * 100;
            expect(percentage1).toBeCloseTo(33.33, 1);

            captcha.clickState.selectedPoints.push({ x: 150, y: 250 });
            const percentage2 = (captcha.clickState.selectedPoints.length / captcha.clickState.maxPoints) * 100;
            expect(percentage2).toBeCloseTo(66.67, 1);

            document.body.removeChild(container);
        });
    });

    describe('Behavior Data Recording', () => {
        test('should record click behavior correctly', () => {
            const container = document.createElement('div');
            container.id = 'captcha-test';
            document.body.appendChild(container);

            const captcha = new Captcha('captcha-test');
            captcha.options.behaviorData = [];

            const point = {
                x: 100,
                y: 200,
                imageIndex: 0,
                timestamp: Date.now()
            };

            captcha.recordClickBehavior(point);

            expect(captcha.options.behaviorData.length).toBe(1);
            expect(captcha.options.behaviorData[0].event).toBe('click');
            expect(captcha.options.behaviorData[0].x).toBe(100);
            expect(captcha.options.behaviorData[0].y).toBe(200);

            document.body.removeChild(container);
        });

        test('should calculate time since start correctly', () => {
            const container = document.createElement('div');
            container.id = 'captcha-test';
            document.body.appendChild(container);

            const captcha = new Captcha('captcha-test');
            captcha.options.behaviorData = [];

            const startTime = Date.now();
            captcha.clickState.startTime = startTime;

            const point = {
                x: 100,
                y: 200,
                imageIndex: 0,
                timestamp: Date.now()
            };

            captcha.recordClickBehavior(point);

            expect(captcha.options.behaviorData[0].timeSinceStart).toBeGreaterThanOrEqual(0);

            document.body.removeChild(container);
        });
    });

    describe('Reset Functionality', () => {
        test('should reset slider state correctly', () => {
            const container = document.createElement('div');
            container.id = 'captcha-test';
            document.body.appendChild(container);

            const captcha = new Captcha('captcha-test');

            captcha.sliderState.isDragging = true;
            captcha.sliderState.currentX = 150;

            captcha.resetSlider();

            expect(captcha.sliderState.isDragging).toBe(false);
            expect(captcha.sliderState.currentX).toBe(0);

            document.body.removeChild(container);
        });
    });

    describe('Tab Switching', () => {
        test('should switch type on tab switch', () => {
            const container = document.createElement('div');
            container.id = 'captcha-test';
            document.body.appendChild(container);

            const captcha = new Captcha('captcha-test');

            captcha.switchTab('click');

            expect(captcha.options.type).toBe('click');

            document.body.removeChild(container);
        });
    });
});
