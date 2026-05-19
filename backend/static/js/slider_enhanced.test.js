/**
 * 滑块验证码增强版测试脚本 - slider_enhanced.test.js
 * 
 * 测试内容：
 * 1. 组件初始化测试
 * 2. 拖动交互测试
 * 3. 动画效果测试
 * 4. 图片加载测试
 * 5. 验证流程测试
 */

const SliderCaptchaEnhanced = window.SliderCaptchaEnhanced;

class SliderCaptchaTest {
    constructor() {
        this.results = [];
        this.container = null;
        this.captcha = null;
    }

    log(testName, passed, message = '') {
        const result = { testName, passed, message };
        this.results.push(result);
        console.log(`[${passed ? '✓' : '✗'}] ${testName}${message ? ': ' + message : ''}`);
        return passed;
    }

    async createTestContainer() {
        this.container = document.createElement('div');
        this.container.id = 'test-slider-container';
        this.container.style.cssText = 'position: fixed; top: -9999px; left: -9999px; width: 320px;';
        document.body.appendChild(this.container);
    }

    cleanup() {
        if (this.captcha) {
            this.captcha.destroy();
            this.captcha = null;
        }
        if (this.container && this.container.parentNode) {
            this.container.parentNode.removeChild(this.container);
        }
        this.container = null;
    }

    async testInitialization() {
        console.log('\n========== 测试：组件初始化 ==========\n');
        
        this.test('类是否存在', typeof SliderCaptchaEnhanced === 'function');
        
        await this.createTestContainer();
        
        try {
            this.captcha = new SliderCaptchaEnhanced('test-slider-container', {
                apiBase: '/api/v1',
                onReady: () => {
                    this.log('onReady回调执行', true);
                }
            });
            
            this.test('DOM元素创建', !!this.captcha.elements.wrapper);
            this.test('图片容器创建', !!this.captcha.elements.imageContainer);
            this.test('滑块容器创建', !!this.captcha.elements.sliderContainer);
            this.test('滑块按钮创建', !!this.captcha.elements.sliderButton);
            this.test('状态初始化', this.captcha.state.isLoading === true);
            
            await this.sleep(500);
            this.test('样式已注入', !!document.getElementById('slider-enhanced-styles'));
            
        } catch (error) {
            this.log('初始化错误', false, error.message);
        }
    }

    async testDragInteraction() {
        console.log('\n========== 测试：拖动交互 ==========\n');
        
        if (!this.captcha) {
            this.log('滑块实例不存在', false);
            return;
        }

        this.test('初始状态isDragging为false', this.captcha.state.isDragging === false);
        this.test('初始状态currentX为0', this.captcha.state.currentX === 0);

        const button = this.captcha.elements.sliderButton;
        const rect = button.getBoundingClientRect();
        
        const mockEvent = {
            type: 'mousedown',
            clientX: rect.left + rect.width / 2,
            clientY: rect.top + rect.height / 2,
            preventDefault: () => {}
        };
        
        button.dispatchEvent(new MouseEvent('mousedown', mockEvent));
        await this.sleep(50);
        
        this.test('开始拖动后isDragging为true', this.captcha.state.isDragging === true);
        this.test('touchIdentifier已设置', this.captcha.touchIdentifier !== undefined);

        const moveEvent = {
            type: 'mousemove',
            clientX: mockEvent.clientX + 100,
            clientY: mockEvent.clientY,
            preventDefault: () => {}
        };
        
        document.dispatchEvent(new MouseEvent('mousemove', moveEvent));
        await this.sleep(50);
        
        this.test('移动后currentX已更新', this.captcha.state.currentX > 0);
        this.test('目标位置targetX已更新', this.captcha.state.targetX > 0);

        const endEvent = {
            type: 'mouseup',
            clientX: moveEvent.clientX,
            clientY: moveEvent.clientY
        };
        
        document.dispatchEvent(new MouseEvent('mouseup', endEvent));
        await this.sleep(50);
        
        this.test('结束拖动后isDragging为false', this.captcha.state.isDragging === false);
    }

    async testAnimation() {
        console.log('\n========== 测试：动画效果 ==========\n');
        
        if (!this.captcha) {
            this.log('滑块实例不存在', false);
            return;
        }

        const initialX = 50;
        this.captcha.state.currentX = initialX;
        
        const startTransform = this.captcha.elements.sliderButton.style.transform;
        this.captcha.updateSliderPosition(initialX);
        await this.sleep(50);
        
        this.test('位置更新后transform已设置', 
            this.captcha.elements.sliderButton.style.transform.includes('translateX'));

        this.captcha.animateToPosition(100, 200);
        await this.sleep(300);
        
        this.test('动画完成后currentX已更新', Math.abs(this.captcha.state.currentX - 100) < 5);

        this.captcha.animateShake();
        const hasShakeClass = this.captcha.elements.sliderButton.classList.contains('shake');
        await this.sleep(600);
        const shakeRemoved = !this.captcha.elements.sliderButton.classList.contains('shake');
        
        this.test('震动动画类已添加', hasShakeClass);
        this.test('震动动画类已移除', shakeRemoved);

        this.test('粒子容器存在', !!this.captcha.elements.particlesContainer);
        this.captcha.spawnParticles('#ff0000', 5);
        await this.sleep(100);
        const particleCount = this.captcha.elements.particlesContainer.children.length;
        this.test('粒子已生成', particleCount > 0);
    }

    async testImageLoading() {
        console.log('\n========== 测试：图片加载 ==========\n');
        
        if (!this.captcha) {
            this.log('滑块实例不存在', false);
            return;
        }

        this.test('初始状态图片未加载', !this.captcha.state.imageLoaded);
        this.test('占位符可见', !this.captcha.elements.imagePlaceholder.classList.contains('hidden'));
        
        this.captcha.hideImagePlaceholder();
        await this.sleep(50);
        this.test('隐藏占位符后不可见', this.captcha.elements.imagePlaceholder.classList.contains('hidden'));
        
        this.captcha.showImagePlaceholder();
        await this.sleep(50);
        this.test('显示占位符后可见', !this.captcha.elements.imagePlaceholder.classList.contains('hidden'));

        this.captcha.animateImageIn();
        await this.sleep(100);
        this.test('图片加载动画执行', this.captcha.elements.image.classList.contains('loaded') || 
            this.captcha.elements.puzzle.style.opacity !== '');
    }

    async testStateManagement() {
        console.log('\n========== 测试：状态管理 ==========\n');
        
        if (!this.captcha) {
            this.log('滑块实例不存在', false);
            return;
        }

        const state = this.captcha.getState();
        this.test('getState返回状态对象', typeof state === 'object');
        this.test('状态对象包含必要字段', 
            'isDragging' in state && 'currentX' in state && 'isLoading' in state);

        this.captcha.setOption('animationDuration', 500);
        this.test('setOption更新选项', this.captcha.options.animationDuration === 500);

        this.captcha.reset();
        await this.sleep(50);
        this.test('重置后currentX为0', this.captcha.state.currentX === 0);
        this.test('重置后isLoading为false', this.captcha.state.isLoading === false);

        this.captcha.disableInteraction();
        const pointerEvents = this.captcha.elements.sliderButton.style.pointerEvents;
        this.test('禁用交互后pointerEvents为none', pointerEvents === 'none');

        this.captcha.enableInteraction();
        this.test('启用交互后pointerEvents为auto', 
            this.captcha.elements.sliderButton.style.pointerEvents === 'auto');
    }

    async testFeedback() {
        console.log('\n========== 测试：反馈提示 ==========\n');
        
        if (!this.captcha) {
            this.log('滑块实例不存在', false);
            return;
        }

        this.captcha.showFeedback('测试消息', 'success');
        await this.sleep(50);
        
        const hasShowClass = this.captcha.elements.feedback.classList.contains('show');
        const hasSuccessClass = this.captcha.elements.feedback.classList.contains('success');
        const messageCorrect = this.captcha.elements.feedback.textContent === '测试消息';
        
        this.test('反馈显示后有show类', hasShowClass);
        this.test('成功反馈有success类', hasSuccessClass);
        this.test('反馈消息正确', messageCorrect);

        await this.sleep(3500);
        const isHidden = !this.captcha.elements.feedback.classList.contains('show');
        this.test('反馈自动隐藏', isHidden);
    }

    async testDestruction() {
        console.log('\n========== 测试：组件销毁 ==========\n');
        
        if (!this.captcha) {
            this.log('滑块实例不存在', false);
            return;
        }

        const wrapperExists = !!this.captcha.elements.wrapper;
        this.captcha.destroy();
        await this.sleep(50);
        
        const wrapperRemoved = !document.getElementById('test-slider-container');
        this.test('销毁后wrapper已移除', wrapperRemoved);
        
        const styleRemoved = !document.getElementById('slider-enhanced-styles');
        this.test('销毁后样式已移除', styleRemoved);
        
        this.captcha = null;
    }

    test(name, condition) {
        return this.log(name, condition === true);
    }

    sleep(ms) {
        return new Promise(resolve => setTimeout(resolve, ms));
    }

    printSummary() {
        console.log('\n========== 测试总结 ==========\n');
        const passed = this.results.filter(r => r.passed).length;
        const total = this.results.length;
        const failed = total - passed;
        
        console.log(`总计: ${total} 测试`);
        console.log(`通过: ${passed}`);
        console.log(`失败: ${failed}`);
        console.log(`通过率: ${(passed / total * 100).toFixed(1)}%`);
        
        if (failed > 0) {
            console.log('\n失败的测试:');
            this.results.filter(r => !r.passed).forEach(r => {
                console.log(`  - ${r.testName}`);
            });
        }
        
        return { passed, failed, total };
    }

    async runAll() {
        console.log('========================================');
        console.log('  滑块验证码增强版 - 自动化测试');
        console.log('========================================');
        
        await this.testInitialization();
        await this.sleep(500);
        
        await this.testDragInteraction();
        await this.sleep(300);
        
        await this.testAnimation();
        await this.sleep(300);
        
        await this.testImageLoading();
        await this.sleep(300);
        
        await this.testStateManagement();
        await this.sleep(300);
        
        await this.testFeedback();
        await this.sleep(4000);
        
        await this.testDestruction();
        
        const summary = this.printSummary();
        
        return summary;
    }
}

window.SliderCaptchaTest = SliderCaptchaTest;

if (typeof module !== 'undefined' && module.exports) {
    module.exports = SliderCaptchaTest;
}
