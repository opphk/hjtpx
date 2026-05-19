(function() {
  'use strict';

  const MobileEnhancedTests = {
    testResults: [],
    passed: 0,
    failed: 0,

    runAllTests() {
      console.log('[MobileEnhancedTests] 开始运行测试...');

      this.testModuleInitialization();
      this.testTouchGestureRecognition();
      this.testGestureFeedback();
      this.testKeyboardInputOptimization();
      this.testMobileAnimations();
      this.testSwipeNavigation();
      this.testPinchZoom();
      this.testGestureHistory();
      this.testHapticFeedback();

      this.printResults();
      return this.testResults;
    },

    assert(condition, testName, message = '') {
      const result = {
        name: testName,
        passed: condition,
        message: message
      };

      this.testResults.push(result);

      if (condition) {
        this.passed++;
        console.log(`✓ ${testName}`);
      } else {
        this.failed++;
        console.error(`✗ ${testName}: ${message}`);
      }

      return condition;
    },

    testModuleInitialization() {
      console.log('\n=== 测试模块初始化 ===');

      this.assert(
        typeof window.MobileEnhancedAdapter !== 'undefined',
        'MobileEnhancedAdapter 全局对象存在'
      );

      this.assert(
        window.MobileEnhancedAdapter instanceof MobileEnhancedAdapter,
        'MobileEnhancedAdapter 是正确类型'
      );

      this.assert(
        typeof window.MobileEnhancedAdapter.init === 'function',
        'init 方法存在'
      );

      this.assert(
        typeof window.MobileEnhancedAdapter.getGestureHistory === 'function',
        'getGestureHistory 方法存在'
      );

      this.assert(
        typeof window.MobileEnhancedAdapter.clearGestureHistory === 'function',
        'clearGestureHistory 方法存在'
      );
    },

    testTouchGestureRecognition() {
      console.log('\n=== 测试触摸手势识别 ===');

      const adapter = window.MobileEnhancedAdapter;

      this.assert(
        typeof adapter.handleTouchStart === 'function',
        'handleTouchStart 方法存在'
      );

      this.assert(
        typeof adapter.handleTouchEnd === 'function',
        'handleTouchEnd 方法存在'
      );

      this.assert(
        typeof adapter.handleTouchMove === 'function',
        'handleTouchMove 方法存在'
      );

      this.assert(
        typeof adapter.handleTouchCancel === 'function',
        'handleTouchCancel 方法存在'
      );

      this.assert(
        typeof adapter.handleTap === 'function',
        'handleTap 方法存在'
      );

      this.assert(
        adapter.isTouchDevice !== undefined,
        'isTouchDevice 属性存在'
      );

      this.assert(
        typeof MOBILE_ENHANCED_CONFIG !== 'undefined',
        '配置常量存在'
      );

      this.assert(
        MOBILE_ENHANCED_CONFIG.longPressDelay === 500,
        'longPressDelay 配置正确'
      );

      this.assert(
        MOBILE_ENHANCED_CONFIG.swipeThreshold === 40,
        'swipeThreshold 配置正确'
      );

      this.assert(
        MOBILE_ENHANCED_CONFIG.pinchZoomMin === 1,
        'pinchZoomMin 配置正确'
      );

      this.assert(
        MOBILE_ENHANCED_CONFIG.pinchZoomMax === 3,
        'pinchZoomMax 配置正确'
      );
    },

    testGestureFeedback() {
      console.log('\n=== 测试手势反馈 ===');

      const adapter = window.MobileEnhancedAdapter;

      this.assert(
        typeof adapter.triggerHapticFeedback === 'function',
        'triggerHapticFeedback 方法存在'
      );

      this.assert(
        typeof adapter.createTouchRipple === 'function',
        'createTouchRipple 方法存在'
      );

      this.assert(
        typeof adapter.showTouchFeedback === 'function',
        'showTouchFeedback 方法存在'
      );

      const testElement = document.createElement('button');
      testElement.className = 'feedback-enabled';

      try {
        adapter.showTouchFeedback(testElement, { clientX: 100, clientY: 100 });
        this.assert(true, 'showTouchFeedback 执行成功');

        const feedback = document.querySelector('.mobile-touch-feedback');
        if (feedback) {
          feedback.remove();
          this.assert(true, '触摸反馈元素已创建');
        }
      } catch (e) {
        this.assert(false, 'showTouchFeedback 执行', e.message);
      }
    },

    testKeyboardInputOptimization() {
      console.log('\n=== 测试键盘输入优化 ===');

      const adapter = window.MobileEnhancedAdapter;

      this.assert(
        typeof adapter.setupKeyboardInputOptimization === 'function',
        'setupKeyboardInputOptimization 方法存在'
      );

      this.assert(
        typeof adapter.validateInput === 'function',
        'validateInput 方法存在'
      );

      this.assert(
        typeof adapter.handleInputFocus === 'function',
        'handleInputFocus 方法存在'
      );

      this.assert(
        typeof adapter.handleInputBlur === 'function',
        'handleInputBlur 方法存在'
      );

      this.assert(
        typeof adapter.adjustViewportForKeyboard === 'function',
        'adjustViewportForKeyboard 方法存在'
      );

      this.assert(
        typeof adapter.scrollInputIntoView === 'function',
        'scrollInputIntoView 方法存在'
      );

      const testInput = document.createElement('input');
      testInput.type = 'email';
      testInput.value = 'test@example.com';
      testInput.required = true;
      document.body.appendChild(testInput);

      const validationResult = adapter.validateInput(testInput);
      this.assert(
        validationResult === true,
        'validateInput 正确验证邮箱'
      );

      testInput.value = 'invalid-email';
      const invalidResult = adapter.validateInput(testInput);
      this.assert(
        invalidResult === false,
        'validateInput 正确拒绝无效邮箱'
      );

      testInput.remove();
    },

    testMobileAnimations() {
      console.log('\n=== 测试移动端动画 ===');

      const adapter = window.MobileEnhancedAdapter;

      this.assert(
        typeof adapter.setupMobileAnimations === 'function',
        'setupMobileAnimations 方法存在'
      );

      this.assert(
        typeof adapter.setupEnterAnimations === 'function',
        'setupEnterAnimations 方法存在'
      );

      this.assert(
        typeof adapter.setupExitAnimations === 'function',
        'setupExitAnimations 方法存在'
      );

      this.assert(
        typeof adapter.setupScrollAnimations === 'function',
        'setupScrollAnimations 方法存在'
      );

      this.assert(
        typeof adapter.playEnterAnimation === 'function',
        'playEnterAnimation 方法存在'
      );

      this.assert(
        typeof adapter.playExitAnimation === 'function',
        'playExitAnimation 方法存在'
      );

      this.assert(
        typeof adapter.animateTransform === 'function',
        'animateTransform 方法存在'
      );

      const testElement = document.createElement('div');
      testElement.style.cssText = 'position: relative; width: 100px; height: 100px;';
      document.body.appendChild(testElement);

      try {
        adapter.animateTransform(testElement, 'scale(1.5)', 100);
        this.assert(true, 'animateTransform 执行成功');

        setTimeout(() => {
          testElement.remove();
        }, 200);
      } catch (e) {
        this.assert(false, 'animateTransform 执行', e.message);
        testElement.remove();
      }
    },

    testSwipeNavigation() {
      console.log('\n=== 测试滑动导航 ===');

      const adapter = window.MobileEnhancedAdapter;

      this.assert(
        typeof adapter.setupSwipeNavigation === 'function',
        'setupSwipeNavigation 方法存在'
      );

      this.assert(
        typeof adapter.dispatchSwipeEvent === 'function',
        'dispatchSwipeEvent 方法存在'
      );

      const swipeTarget = document.createElement('div');
      swipeTarget.setAttribute('data-swipe-navigate', 'true');
      swipeTarget.setAttribute('data-swipe-directions', 'left,right');
      document.body.appendChild(swipeTarget);

      let swipeEventFired = false;
      swipeTarget.addEventListener('swipenavigate', (e) => {
        swipeEventFired = true;
      });

      adapter.dispatchSwipeEvent('left', 100, 200);

      this.assert(
        swipeEventFired,
        'swipenavigate 事件正确触发'
      );

      swipeTarget.remove();
    },

    testPinchZoom() {
      console.log('\n=== 测试双指缩放 ===');

      const adapter = window.MobileEnhancedAdapter;

      this.assert(
        typeof adapter.setupPinchZoom === 'function',
        'setupPinchZoom 方法存在'
      );

      this.assert(
        typeof adapter.handleMultiTouchStart === 'function',
        'handleMultiTouchStart 方法存在'
      );

      this.assert(
        typeof adapter.handleMultiTouchMove === 'function',
        'handleMultiTouchMove 方法存在'
      );

      this.assert(
        adapter.currentZoom !== undefined,
        'currentZoom 属性存在'
      );
    },

    testGestureHistory() {
      console.log('\n=== 测试手势历史 ===');

      const adapter = window.MobileEnhancedAdapter;

      this.assert(
        typeof adapter.getGestureHistory === 'function',
        'getGestureHistory 方法存在'
      );

      this.assert(
        typeof adapter.clearGestureHistory === 'function',
        'clearGestureHistory 方法存在'
      );

      this.assert(
        typeof adapter.recordGesture === 'function',
        'recordGesture 方法存在'
      );

      adapter.clearGestureHistory();

      const historyBefore = adapter.getGestureHistory();
      this.assert(
        Array.isArray(historyBefore) && historyBefore.length === 0,
        'clearGestureHistory 正确清空历史'
      );

      adapter.recordGesture({
        type: 'tap',
        x: 100,
        y: 200
      });

      const historyAfter = adapter.getGestureHistory();
      this.assert(
        Array.isArray(historyAfter) && historyAfter.length === 1,
        'recordGesture 正确记录手势'
      );

      this.assert(
        historyAfter[0].type === 'tap' &&
        historyAfter[0].x === 100 &&
        historyAfter[0].y === 200,
        'recordGesture 正确保存手势数据'
      );

      adapter.clearGestureHistory();
    },

    testHapticFeedback() {
      console.log('\n=== 测试触觉反馈 ===');

      const adapter = window.MobileEnhancedAdapter;

      this.assert(
        typeof adapter.triggerHapticFeedback === 'function',
        'triggerHapticFeedback 方法存在'
      );

      const originalVibrate = navigator.vibrate;
      let vibrateCalled = false;
      let vibratePattern = null;

      navigator.vibrate = (pattern) => {
        vibrateCalled = true;
        vibratePattern = pattern;
      };

      try {
        adapter.triggerHapticFeedback('light');
        this.assert(
          vibrateCalled && vibratePattern === 10,
          'triggerHapticFeedback 正确触发轻触反馈'
        );

        vibrateCalled = false;
        adapter.triggerHapticFeedback('medium');
        this.assert(
          vibrateCalled && vibratePattern === 25,
          'triggerHapticFeedback 正确触发中等反馈'
        );

        vibrateCalled = false;
        adapter.triggerHapticFeedback('heavy');
        this.assert(
          vibrateCalled && vibratePattern === 50,
          'triggerHapticFeedback 正确触发重反馈'
        );
      } catch (e) {
        this.assert(false, 'triggerHapticFeedback 执行', e.message);
      }

      navigator.vibrate = originalVibrate;
    },

    printResults() {
      console.log('\n========================================');
      console.log(`测试结果: ${this.passed} 通过, ${this.failed} 失败`);
      console.log(`总计: ${this.testResults.length} 个测试`);
      console.log('========================================\n');

      if (this.failed === 0) {
        console.log('🎉 所有测试通过!');
      } else {
        console.log('⚠️  部分测试失败，请检查上述错误信息。');
      }
    }
  };

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => {
      setTimeout(() => MobileEnhancedTests.runAllTests(), 500);
    });
  } else {
    setTimeout(() => MobileEnhancedTests.runAllTests(), 500);
  }

  window.MobileEnhancedTests = MobileEnhancedTests;
})();
