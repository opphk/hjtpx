document.addEventListener('DOMContentLoaded', function() {
    console.log('用户端已加载');

    const navLinks = document.querySelectorAll('nav a');
    navLinks.forEach(link => {
        link.addEventListener('click', function(e) {
            navLinks.forEach(l => l.classList.remove('active'));
            this.classList.add('active');
        });
    });

    const buttons = document.querySelectorAll('.btn');
    buttons.forEach(btn => {
        btn.addEventListener('mouseenter', function() {
            this.style.transform = 'scale(1.05)';
        });
        btn.addEventListener('mouseleave', function() {
            this.style.transform = 'scale(1)';
        });
    });

    initKeyboardNavigation();
    initAccessibilityFeatures();
    injectCaptchaStyles();
    initTouchGestures();
    initTouchFeedback();
});

function initKeyboardNavigation() {
    const focusableElements = document.querySelectorAll(
        'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])'
    );
    
    focusableElements.forEach(el => {
        el.addEventListener('keydown', function(e) {
            if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                this.click();
            }
        });
    });

    const captchaInteractive = document.querySelectorAll('.captcha-interactive');
    captchaInteractive.forEach(el => {
        el.setAttribute('tabindex', '0');
        el.setAttribute('role', 'button');
    });
}

function initAccessibilityFeatures() {
    const announcer = document.createElement('div');
    announcer.setAttribute('aria-live', 'polite');
    announcer.setAttribute('aria-atomic', 'true');
    announcer.className = 'captcha-sr-only';
    announcer.id = 'captcha-announcer';
    document.body.appendChild(announcer);

    window.captchaAnnounce = function(message) {
        const announcer = document.getElementById('captcha-announcer');
        if (announcer) {
            announcer.textContent = message;
            setTimeout(() => {
                announcer.textContent = '';
            }, 1000);
        }
    };

    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
        document.body.classList.add('reduce-motion');
    }

    window.matchMedia('(prefers-reduced-motion: reduce)').addEventListener('change', (e) => {
        document.body.classList.toggle('reduce-motion', e.matches);
    });
}

function injectCaptchaStyles() {
    if (document.getElementById('captcha-dynamic-styles')) {
        return;
    }

    const styleSheet = document.createElement('style');
    styleSheet.id = 'captcha-dynamic-styles';
    styleSheet.textContent = `
        .captcha-container {
            background: #fff;
            border-radius: 12px;
            box-shadow: 0 4px 20px rgba(0,0,0,0.08);
            overflow: hidden;
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
        }
        .captcha-header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 20px;
            text-align: center;
        }
        .captcha-header h3 {
            margin: 0 0 5px 0;
            font-size: 20px;
            font-weight: 600;
        }
        .captcha-header p {
            margin: 0;
            font-size: 14px;
            opacity: 0.9;
        }
        .captcha-body {
            padding: 20px;
        }
        .captcha-tabs {
            display: flex;
            gap: 10px;
            margin-bottom: 15px;
            border-bottom: 2px solid #f0f0f0;
            padding-bottom: 10px;
        }
        .captcha-tab {
            flex: 1;
            padding: 10px 20px;
            border: none;
            background: transparent;
            color: #666;
            font-size: 14px;
            font-weight: 500;
            cursor: pointer;
            border-radius: 6px;
            transition: all 0.3s ease;
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 8px;
        }
        .captcha-tab .tab-icon {
            font-size: 16px;
        }
        .captcha-tab:hover {
            background: #f5f5f5;
        }
        .captcha-tab.active {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
        }
        .captcha-content {
            display: none;
        }
        .captcha-content.active {
            display: block;
        }
        .captcha-loading-overlay {
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: rgba(255,255,255,0.95);
            display: flex;
            align-items: center;
            justify-content: center;
            z-index: 20;
            backdrop-filter: blur(4px);
        }
        .captcha-loading-container {
            text-align: center;
        }
        .loading-animation-pulse {
            margin-bottom: 15px;
        }
        .loading-dots {
            display: flex;
            gap: 8px;
            justify-content: center;
        }
        .loading-dots span {
            width: 12px;
            height: 12px;
            background: #667eea;
            border-radius: 50%;
            animation: loading-bounce 1.4s infinite ease-in-out both;
        }
        .loading-dots span:nth-child(1) { animation-delay: -0.32s; }
        .loading-dots span:nth-child(2) { animation-delay: -0.16s; }
        .loading-dots span:nth-child(3) { animation-delay: 0s; }
        .loading-dots span:nth-child(4) { animation-delay: 0.16s; }
        .loading-dots span:nth-child(5) { animation-delay: 0.32s; }
        @keyframes loading-bounce {
            0%, 80%, 100% { 
                transform: scale(0);
                opacity: 0.5;
            }
            40% { 
                transform: scale(1);
                opacity: 1;
            }
        }
        .loading-progress-bar {
            width: 200px;
            height: 4px;
            background: #e0e0e0;
            border-radius: 2px;
            overflow: hidden;
            margin: 15px auto;
        }
        .loading-progress-fill {
            height: 100%;
            background: linear-gradient(90deg, #667eea 0%, #764ba2 100%);
            width: 0%;
            transition: width 0.3s ease;
        }
        .loading-message {
            color: #666;
            font-size: 14px;
        }
        .captcha-image-wrapper {
            position: relative;
            width: 100%;
            max-width: 360px;
            margin: 0 auto 15px;
            border-radius: 8px;
            overflow: hidden;
            background: #f5f5f5;
        }
        .captcha-canvas {
            display: block;
            width: 100%;
            height: auto;
        }
        .captcha-background-layer {
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            pointer-events: none;
        }
        .captcha-puzzle {
            position: absolute;
            top: 0;
            left: 0;
            width: 50px;
            height: 50px;
            pointer-events: none;
            z-index: 2;
            transition: transform 0.1s ease;
        }
        .puzzle-piece-square {
            width: 50px;
            height: 50px;
            background: rgba(255,255,255,0.3);
            border: 2px solid rgba(255,255,255,0.8);
            box-shadow: 0 2px 8px rgba(0,0,0,0.2);
        }
        .puzzle-piece-circle {
            width: 50px;
            height: 50px;
            background: rgba(255,255,255,0.3);
            border: 2px solid rgba(255,255,255,0.8);
            border-radius: 50%;
            box-shadow: 0 2px 8px rgba(0,0,0,0.2);
        }
        .puzzle-piece-triangle {
            width: 0;
            height: 0;
            border-left: 25px solid transparent;
            border-right: 25px solid transparent;
            border-bottom: 43px solid rgba(255,255,255,0.4);
            background: transparent;
        }
        .puzzle-piece-diamond {
            width: 50px;
            height: 50px;
            background: rgba(255,255,255,0.3);
            border: 2px solid rgba(255,255,255,0.8);
            transform: rotate(45deg);
            margin: 5px;
        }
        .puzzle-piece-hexagon {
            width: 50px;
            height: 28.87px;
            background: rgba(255,255,255,0.3);
            border: 2px solid rgba(255,255,255,0.8);
            position: relative;
            margin-top: 10px;
        }
        .captcha-refresh {
            position: absolute;
            top: 8px;
            right: 8px;
            width: 32px;
            height: 32px;
            border: none;
            background: rgba(255,255,255,0.9);
            border-radius: 50%;
            cursor: pointer;
            font-size: 16px;
            display: flex;
            align-items: center;
            justify-content: center;
            transition: all 0.3s ease;
            z-index: 10;
            color: #666;
        }
        .captcha-refresh:hover {
            background: white;
            transform: rotate(180deg);
        }
        .captcha-refresh:focus {
            outline: 2px solid #667eea;
            outline-offset: 2px;
        }
        .captcha-slider-container {
            position: relative;
            width: 100%;
            max-width: 360px;
            height: 44px;
            margin: 0 auto;
            background: #f5f5f5;
            border-radius: 22px;
            overflow: hidden;
            border: 1px solid #e0e0e0;
            transition: all 0.3s ease;
        }
        .captcha-slider-container.is-dragging {
            border-color: #667eea;
            box-shadow: 0 0 0 3px rgba(102,126,234,0.1);
        }
        .captcha-slider-container.error-flash {
            animation: error-flash-animation 0.5s ease;
        }
        @keyframes error-flash-animation {
            0%, 100% { background: #f5f5f5; }
            50% { background: #fff2f0; }
        }
        .captcha-slider-track {
            position: absolute;
            left: 2px;
            top: 2px;
            height: 40px;
            width: 0;
            background: linear-gradient(90deg, #667eea 0%, #764ba2 100%);
            border-radius: 20px;
            transition: width 0.1s ease;
        }
        .captcha-slider-text {
            position: absolute;
            width: 100%;
            text-align: center;
            line-height: 44px;
            font-size: 14px;
            color: #666;
            pointer-events: none;
            z-index: 1;
        }
        .captcha-slider-hint {
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 6px;
            margin-top: 8px;
            font-size: 12px;
            color: #999;
        }
        .captcha-slider-hint .hint-icon {
            color: #667eea;
        }
        .captcha-slider-button {
            position: absolute;
            left: 2px;
            top: 2px;
            width: 40px;
            height: 40px;
            background: white;
            border-radius: 50%;
            cursor: grab;
            display: flex;
            align-items: center;
            justify-content: center;
            box-shadow: 0 2px 8px rgba(0,0,0,0.15);
            transition: left 0.1s ease, transform 0.2s ease, background 0.3s ease;
            z-index: 2;
            color: #667eea;
            touch-action: none;
        }
        .captcha-slider-button:hover {
            transform: scale(1.05);
        }
        .captcha-slider-button.dragging {
            cursor: grabbing;
            transform: scale(1.1);
            box-shadow: 0 4px 12px rgba(102,126,234,0.4);
        }
        .captcha-slider-button.verifying {
            animation: pulse-verifying 1s infinite;
        }
        @keyframes pulse-verifying {
            0%, 100% { box-shadow: 0 2px 8px rgba(102,126,234,0.3); }
            50% { box-shadow: 0 2px 16px rgba(102,126,234,0.6); }
        }
        .captcha-slider-button.success {
            background: #52c41a;
            color: white;
            animation: success-bounce 0.5s ease;
        }
        @keyframes success-bounce {
            0% { transform: scale(1); }
            50% { transform: scale(1.2); }
            100% { transform: scale(1); }
        }
        .captcha-slider-button.error {
            background: #ff4d4f;
            color: white;
            animation: shake 0.5s ease-in-out;
        }
        @keyframes shake {
            0%, 100% { transform: translateX(0); }
            25% { transform: translateX(-10px); }
            75% { transform: translateX(10px); }
        }
        .captcha-click-hint {
            text-align: center;
            padding: 10px;
            background: #f8f8f8;
            border-radius: 6px;
            margin-bottom: 10px;
            font-size: 14px;
            color: #333;
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 8px;
        }
        .captcha-click-hint .hint-icon {
            color: #667eea;
        }
        .captcha-click-grid {
            position: relative;
            display: inline-block;
        }
        .captcha-click-image {
            display: block;
            width: 100%;
            max-width: 360px;
            border-radius: 8px;
        }
        .captcha-click-marker {
            position: absolute;
            width: 28px;
            height: 28px;
            background: #667eea;
            border: 2px solid white;
            border-radius: 50%;
            color: white;
            font-size: 14px;
            font-weight: bold;
            display: flex;
            align-items: center;
            justify-content: center;
            transform: translate(-50%, -50%);
            cursor: pointer;
            box-shadow: 0 2px 8px rgba(0,0,0,0.2);
            animation: marker-pop 0.3s cubic-bezier(0.175, 0.885, 0.32, 1.275);
            transition: transform 0.2s ease, background 0.2s ease;
            z-index: 10;
        }
        .captcha-click-marker:hover {
            transform: translate(-50%, -50%) scale(1.1);
            background: #764ba2;
        }
        .captcha-click-marker.success-marker {
            background: #52c41a;
            animation: marker-success 0.5s ease;
        }
        @keyframes marker-success {
            0% { transform: translate(-50%, -50%) scale(1); }
            50% { transform: translate(-50%, -50%) scale(1.3); }
            100% { transform: translate(-50%, -50%) scale(1); }
        }
        @keyframes marker-pop {
            0% { transform: translate(-50%, -50%) scale(0); }
            50% { transform: translate(-50%, -50%) scale(1.2); }
            100% { transform: translate(-50%, -50%) scale(1); }
        }
        .captcha-click-progress {
            text-align: center;
            margin: 10px 0;
            font-size: 14px;
            color: #666;
        }
        .count-badge {
            display: inline-block;
            min-width: 24px;
            padding: 2px 8px;
            background: #f0f0f0;
            border-radius: 12px;
            font-weight: 600;
            transition: all 0.3s ease;
        }
        .count-badge.partial {
            background: #e6f7ff;
            color: #1890ff;
        }
        .count-badge.complete {
            background: #f6ffed;
            color: #52c41a;
        }
        .captcha-actions {
            display: flex;
            gap: 10px;
            justify-content: center;
            margin-top: 15px;
        }
        .captcha-btn {
            padding: 10px 30px;
            border: none;
            border-radius: 6px;
            font-size: 14px;
            font-weight: 500;
            cursor: pointer;
            transition: all 0.3s ease;
            display: inline-flex;
            align-items: center;
            gap: 6px;
        }
        .captcha-btn i {
            font-size: 14px;
        }
        .captcha-btn-primary {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
        }
        .captcha-btn-primary:hover {
            opacity: 0.9;
            transform: translateY(-2px);
            box-shadow: 0 4px 12px rgba(102,126,234,0.4);
        }
        .captcha-btn-primary:focus {
            outline: 2px solid #667eea;
            outline-offset: 2px;
        }
        .captcha-btn-secondary {
            background: #f5f5f5;
            color: #666;
        }
        .captcha-btn-secondary:hover {
            background: #e8e8e8;
        }
        .captcha-result {
            text-align: center;
            padding: 12px;
            margin-top: 15px;
            border-radius: 6px;
            font-size: 14px;
            font-weight: 500;
            display: none;
        }
        .captcha-result.show {
            display: block;
            animation: fadeIn 0.3s ease;
        }
        .captcha-result.success {
            background: #f6ffed;
            color: #52c41a;
            border: 1px solid #b7eb8f;
        }
        .captcha-result.error {
            background: #fff2f0;
            color: #ff4d4f;
            border: 1px solid #ffccc7;
        }
        @keyframes fadeIn {
            from { opacity: 0; transform: translateY(-10px); }
            to { opacity: 1; transform: translateY(0); }
        }
        .captcha-footer {
            padding: 12px 20px;
            background: #fafafa;
            border-top: 1px solid #f0f0f0;
        }
        .captcha-security-badge {
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 6px;
            font-size: 12px;
            color: #52c41a;
        }
        .captcha-security-badge i {
            font-size: 14px;
        }
        .captcha-image-skeleton,
        .captcha-click-skeleton {
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: #f0f0f0;
            display: none;
            overflow: hidden;
            z-index: 5;
            border-radius: 8px;
        }
        .captcha-image-skeleton.active,
        .captcha-click-skeleton.active {
            display: block;
        }
        .skeleton-shimmer {
            position: absolute;
            top: 0;
            left: -100%;
            width: 100%;
            height: 100%;
            background: linear-gradient(
                90deg,
                transparent 0%,
                rgba(255,255,255,0.4) 50%,
                transparent 100%
            );
            animation: shimmer 1.5s infinite;
        }
        @keyframes shimmer {
            0% { left: -100%; }
            100% { left: 100%; }
        }
        .success-particle {
            animation: particle-fly 0.6s ease-out forwards;
        }
        @keyframes particle-fly {
            to {
                opacity: 0;
            }
        }
        .error-shake {
            animation: error-shake-animation 0.5s ease;
        }
        @keyframes error-shake-animation {
            0%, 100% { transform: translateX(0); }
            25% { transform: translateX(-5px); }
            75% { transform: translateX(5px); }
        }
        @media (max-width: 576px) {
            .captcha-container {
                border-radius: 8px;
                margin: 0 10px;
            }
            .captcha-header {
                padding: 15px;
            }
            .captcha-header h3 {
                font-size: 18px;
            }
            .captcha-body {
                padding: 15px;
            }
            .captcha-tabs {
                flex-direction: column;
                gap: 5px;
            }
            .captcha-tab {
                padding: 8px 16px;
                font-size: 13px;
            }
            .captcha-image-wrapper {
                max-width: 100%;
                margin-bottom: 10px;
            }
            .captcha-slider-container {
                max-width: 100%;
                height: 40px;
                border-radius: 20px;
            }
            .captcha-slider-button {
                width: 36px;
                height: 36px;
            }
            .captcha-slider-track {
                height: 36px;
            }
            .captcha-slider-text {
                line-height: 40px;
                font-size: 13px;
            }
            .captcha-click-image {
                max-width: 100%;
            }
            .captcha-btn {
                padding: 8px 20px;
                font-size: 13px;
            }
            .captcha-actions {
                flex-direction: column;
            }
            .captcha-btn {
                width: 100%;
                justify-content: center;
            }
        }
        @media (max-width: 360px) {
            .captcha-tab .tab-icon {
                display: none;
            }
            .captcha-slider-hint {
                font-size: 11px;
            }
        }
        @media (prefers-reduced-motion: reduce) {
            *,
            *::before,
            *::after {
                animation-duration: 0.01ms !important;
                animation-iteration-count: 1 !important;
                transition-duration: 0.01ms !important;
            }
        }
        .visually-hidden {
            position: absolute;
            width: 1px;
            height: 1px;
            padding: 0;
            margin: -1px;
            overflow: hidden;
            clip: rect(0, 0, 0, 0);
            white-space: nowrap;
            border: 0;
        }
    `;

    document.head.appendChild(styleSheet);
}

function initTouchGestures() {
    if (!('ontouchstart' in window) && navigator.maxTouchPoints === 0) {
        return;
    }

    const touchConfig = {
        swipeThreshold: 50,
        tapThreshold: 200,
        longPressDelay: 500,
        doubleTapDelay: 300,
        dragThreshold: 10
    };

    let touchStartX = 0;
    let touchStartY = 0;
    let touchStartTime = 0;
    let isSwipe = false;
    let isDragging = false;
    let longPressTimer = null;
    let lastTapTime = 0;
    let lastTapPos = { x: 0, y: 0 };

    const captchaInteractive = document.querySelectorAll('.captcha-interactive, .captcha-slider-button, .captcha-click-marker, .captcha-refresh');

    captchaInteractive.forEach(el => {
        el.addEventListener('touchstart', function(e) {
            const touch = e.touches[0];
            touchStartX = touch.clientX;
            touchStartY = touch.clientY;
            touchStartTime = Date.now();
            isSwipe = true;
            isDragging = false;

            this.classList.add('touch-active');

            longPressTimer = setTimeout(() => {
                this.classList.add('long-press-active');
                triggerHapticFeedback('medium');
                this.dispatchEvent(new CustomEvent('longpress', { bubbles: true }));
            }, touchConfig.longPressDelay);
        }, { passive: true });

        el.addEventListener('touchmove', function(e) {
            if (longPressTimer) {
                const touch = e.touches[0];
                const deltaX = Math.abs(touch.clientX - touchStartX);
                const deltaY = Math.abs(touch.clientY - touchStartY);

                if (deltaX > touchConfig.dragThreshold || deltaY > touchConfig.dragThreshold) {
                    clearTimeout(longPressTimer);
                    longPressTimer = null;
                    this.classList.remove('long-press-active');
                    isDragging = true;
                }
            }

            if (isDragging) {
                this.classList.add('dragging');
            }
        }, { passive: true });

        el.addEventListener('touchend', function(e) {
            const touch = e.changedTouches[0];
            const touchDuration = Date.now() - touchStartTime;
            const deltaX = touch.clientX - touchStartX;
            const deltaY = touch.clientY - touchStartY;

            clearTimeout(longPressTimer);
            longPressTimer = null;

            this.classList.remove('touch-active', 'long-press-active', 'dragging');

            if (touchDuration < touchConfig.tapThreshold && Math.abs(deltaX) < touchConfig.dragThreshold && Math.abs(deltaY) < touchConfig.dragThreshold) {
                handleTap(this, touch);
            }

            if (isSwipe && Math.abs(deltaX) > touchConfig.swipeThreshold) {
                const direction = deltaX > 0 ? 'right' : 'left';
                this.dispatchEvent(new CustomEvent('swipe', {
                    bubbles: true,
                    detail: { direction: direction, distance: Math.abs(deltaX) }
                }));
            }

            if (isSwipe && Math.abs(deltaY) > touchConfig.swipeThreshold && Math.abs(deltaY) > Math.abs(deltaX)) {
                const direction = deltaY > 0 ? 'down' : 'up';
                this.dispatchEvent(new CustomEvent('swipe', {
                    bubbles: true,
                    detail: { direction: direction, distance: Math.abs(deltaY) }
                }));
            }

            isSwipe = false;
            isDragging = false;
        }, { passive: true });

        el.addEventListener('touchcancel', function() {
            clearTimeout(longPressTimer);
            longPressTimer = null;
            this.classList.remove('touch-active', 'long-press-active', 'dragging');
            isSwipe = false;
            isDragging = false;
        });
    });

    function handleTap(target, touch) {
        const now = Date.now();

        if (now - lastTapTime < touchConfig.doubleTapDelay) {
            const deltaX = Math.abs(touch.clientX - lastTapPos.x);
            const deltaY = Math.abs(touch.clientY - lastTapPos.y);

            if (deltaX < 50 && deltaY < 50) {
                target.dispatchEvent(new CustomEvent('doubletap', {
                    bubbles: true,
                    detail: { x: touch.clientX, y: touch.clientY }
                }));
                lastTapTime = 0;
                triggerHapticFeedback('light');
                return;
            }
        }

        lastTapTime = now;
        lastTapPos = { x: touch.clientX, y: touch.clientY };
        triggerHapticFeedback('light');

        target.dispatchEvent(new CustomEvent('tap', {
            bubbles: true,
            detail: { x: touch.clientX, y: touch.clientY }
        }));
    }
}

function initTouchFeedback() {
    if (document.getElementById('touch-feedback-styles')) {
        return;
    }

    const style = document.createElement('style');
    style.id = 'touch-feedback-styles';
    style.textContent = `
        .touch-active {
            transform: scale(0.96);
            opacity: 0.9;
            transition: transform 0.15s cubic-bezier(0.4, 0, 0.2, 1), opacity 0.15s ease;
        }

        .long-press-active {
            background-color: rgba(102, 126, 234, 0.2) !important;
            box-shadow: inset 0 0 20px rgba(102, 126, 234, 0.15);
        }

        .dragging {
            opacity: 0.85;
            transform: scale(1.08);
            cursor: grabbing;
            box-shadow: 0 8px 16px rgba(102, 126, 234, 0.3);
        }

        @media (hover: none) and (pointer: coarse) {
            .captcha-interactive,
            .captcha-slider-button,
            .captcha-click-marker,
            .captcha-refresh,
            .btn,
            .nav-link {
                -webkit-tap-highlight-color: rgba(102, 126, 234, 0.2);
                -webkit-touch-callout: none;
                touch-action: manipulation;
                user-select: none;
                -webkit-user-select: none;
            }
        }

        @media (prefers-reduced-motion: reduce) {
            .touch-active,
            .long-press-active,
            .dragging {
                transition: none !important;
            }
        }
    `;

    document.head.appendChild(style);
}

function triggerHapticFeedback(intensity = 'light') {
    if ('vibrate' in navigator) {
        const patterns = {
            light: 10,
            medium: 25,
            heavy: 50
        };
        navigator.vibrate(patterns[intensity] || patterns.light);
    }
}
