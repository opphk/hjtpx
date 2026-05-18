/**
 * JavaScript SDK 完整示例
 *
 * 展示在浏览器环境中如何使用 SDK 的所有功能
 */

// 等待 DOM 加载完成
document.addEventListener('DOMContentLoaded', function() {
    console.log('JavaScript SDK Examples Loaded');

    initExamples();
});

let captchaClient = null;
let currentSessionId = null;

function initExamples() {
    captchaClient = new CaptchaClient('http://localhost:8080', {
        timeout: 30000,
        retryCount: 3,
        retryDelay: 1000
    });
}

function log(message, type = 'info') {
    const logContainer = document.getElementById('log-output');
    if (!logContainer) {
        console.log(message);
        return;
    }

    const logEntry = document.createElement('div');
    logEntry.className = `log-entry log-${type}`;
    logEntry.textContent = `[${new Date().toLocaleTimeString()}] ${message}`;
    logContainer.appendChild(logEntry);
    logContainer.scrollTop = logContainer.scrollHeight;
}

async function exampleSliderCaptcha() {
    log('开始滑块验证码示例', 'info');

    try {
        log('步骤1: 获取滑块验证码', 'info');
        const slider = await captchaClient.getSliderCaptcha({
            width: 320,
            height: 160,
            tolerance: 8
        });

        currentSessionId = slider.session_id;
        log(`✓ Session ID: ${slider.session_id}`, 'success');
        log(`✓ 图片宽度: ${slider.image_width || 'N/A'}`, 'info');
        log(`✓ 图片高度: ${slider.image_height || 'N/A'}`, 'info');

        if (slider.hint_url) {
            log(`✓ 提示URL: ${slider.hint_url}`, 'info');
        }

        displaySliderImage(slider);

        log('步骤2: 等待用户操作...', 'info');

    } catch (error) {
        log(`✗ 错误: ${error.message}`, 'error');
        handleError(error);
    }
}

function displaySliderImage(captcha) {
    const container = document.getElementById('captcha-display');
    if (!container) return;

    container.innerHTML = `
        <div class="captcha-container">
            <div class="captcha-instructions">
                <p>请拖动滑块完成验证</p>
            </div>
            <div class="captcha-images">
                <img src="${captcha.image_url}" alt="背景图" class="captcha-bg" />
                <img src="${captcha.puzzle_url}" alt="拼图" class="captcha-puzzle"
                     style="position: absolute; left: 0; top: 0; cursor: move;" />
            </div>
            <div class="captcha-slider-track">
                <div class="captcha-slider-thumb" id="slider-thumb"></div>
            </div>
            <button onclick="refreshSliderCaptcha()" class="refresh-btn">刷新</button>
        </div>
    `;

    initSliderInteraction(captcha);
}

function initSliderInteraction(captcha) {
    const thumb = document.getElementById('slider-thumb');
    if (!thumb) return;

    let isDragging = false;
    let startX = 0;
    let currentX = 0;
    const maxX = 300;

    thumb.addEventListener('mousedown', (e) => {
        isDragging = true;
        startX = e.clientX;
        thumb.classList.add('dragging');
    });

    document.addEventListener('mousemove', (e) => {
        if (!isDragging) return;

        const clientX = e.clientX;
        currentX = clientX - startX;
        currentX = Math.max(0, Math.min(currentX, maxX));

        thumb.style.left = `${currentX}px`;
    });

    document.addEventListener('mouseup', async () => {
        if (!isDragging) return;
        isDragging = false;
        thumb.classList.remove('dragging');

        log(`步骤3: 提交验证 (x=${Math.round(currentX)})`, 'info');

        try {
            const result = await captchaClient.verifySliderCaptcha({
                session_id: captcha.session_id,
                x: Math.round(currentX),
                y: captcha.secret_y
            });

            if (result.success) {
                log('✓ 验证成功!', 'success');
                showSuccessMessage('滑块验证成功');
            } else {
                log(`✗ 验证失败: ${result.message}`, 'error');
                showFailMessage(result.message);
                setTimeout(() => refreshSliderCaptcha(), 2000);
            }
        } catch (error) {
            log(`✗ 验证错误: ${error.message}`, 'error');
        }
    });
}

async function exampleClickCaptcha() {
    log('开始点击验证码示例', 'info');

    try {
        log('步骤1: 获取点击验证码', 'info');
        const click = await captchaClient.getClickCaptcha({
            mode: 'number',
            shuffle: true,
            points: 3
        });

        currentSessionId = click.session_id;
        log(`✓ Session ID: ${click.session_id}`, 'success');
        log(`✓ 提示: ${click.hint}`, 'info');
        log(`✓ 模式: ${click.mode}`, 'info');

        displayClickImage(click);

    } catch (error) {
        log(`✗ 错误: ${error.message}`, 'error');
    }
}

function displayClickImage(captcha) {
    const container = document.getElementById('captcha-display');
    if (!container) return;

    const clickPoints = [];

    container.innerHTML = `
        <div class="captcha-container">
            <div class="captcha-instructions">
                <p>${captcha.hint || '请按顺序点击图片'}</p>
            </div>
            <div class="captcha-images click-captcha">
                <img src="${captcha.image_url}" alt="验证码" class="captcha-img clickable" />
            </div>
            <div class="click-instructions">
                <p>已点击: <span id="click-count">0</span> / ${captcha.max_points || 3}</p>
            </div>
            <div class="captcha-controls">
                <button onclick="submitClickCaptcha()" class="verify-btn">验证</button>
                <button onclick="refreshClickCaptcha()" class="refresh-btn">刷新</button>
            </div>
        </div>
    `;

    const img = container.querySelector('.captcha-img');
    img.addEventListener('click', (e) => {
        const rect = img.getBoundingClientRect();
        const x = Math.round(e.clientX - rect.left);
        const y = Math.round(e.clientY - rect.top);

        clickPoints.push([x, y]);
        document.getElementById('click-count').textContent = clickPoints.length;

        const marker = document.createElement('div');
        marker.className = 'click-marker';
        marker.textContent = clickPoints.length;
        marker.style.left = `${e.clientX - rect.left}px`;
        marker.style.top = `${e.clientY - rect.top}px`;
        img.parentElement.appendChild(marker);
    });

    window.currentClickPoints = clickPoints;
    window.currentClickCaptcha = captcha;
}

async function submitClickCaptcha() {
    const points = window.currentClickPoints || [];
    const captcha = window.currentClickCaptcha;

    if (points.length === 0) {
        log('请先点击图片', 'warn');
        return;
    }

    log('步骤2: 提交验证', 'info');

    try {
        const result = await captchaClient.verifyClickCaptcha({
            session_id: captcha.session_id,
            points: points,
            click_sequence: captcha.hint_order
        });

        if (result.success) {
            log('✓ 验证成功!', 'success');
            showSuccessMessage('点击验证成功');
        } else {
            log(`✗ 验证失败: ${result.message}`, 'error');
            showFailMessage(result.message);
        }
    } catch (error) {
        log(`✗ 错误: ${error.message}`, 'error');
    }
}

async function exampleImageCaptcha() {
    log('开始图形验证码示例', 'info');

    try {
        log('步骤1: 获取图形验证码', 'info');
        const image = await captchaClient.getImageCaptcha({
            type: 'mixed',
            count: 4
        });

        log(`✓ Challenge ID: ${image.challenge_id}`, 'success');

        displayImageCaptcha(image);

    } catch (error) {
        log(`✗ 错误: ${error.message}`, 'error');
    }
}

function displayImageCaptcha(captcha) {
    const container = document.getElementById('captcha-display');
    if (!container) return;

    container.innerHTML = `
        <div class="captcha-container">
            <div class="captcha-instructions">
                <p>请输入图片中的字符</p>
            </div>
            <div class="captcha-images">
                <img src="data:image/png;base64,${captcha.image}" alt="验证码" class="captcha-img" />
            </div>
            <div class="captcha-input">
                <input type="text" id="captcha-answer" placeholder="请输入验证码" maxlength="6" />
            </div>
            <div class="captcha-controls">
                <button onclick="submitImageCaptcha('${captcha.challenge_id}')" class="verify-btn">验证</button>
                <button onclick="refreshImageCaptcha()" class="refresh-btn">刷新</button>
            </div>
        </div>
    `;
}

async function submitImageCaptcha(challengeId) {
    const answer = document.getElementById('captcha-answer').value;

    if (!answer) {
        log('请输入验证码', 'warn');
        return;
    }

    log('步骤2: 提交验证', 'info');

    try {
        const result = await captchaClient.verifyImageCaptcha(challengeId, answer);

        if (result.success) {
            log('✓ 验证成功!', 'success');
            showSuccessMessage('图形验证成功');
        } else {
            log(`✗ 验证失败: ${result.message}`, 'error');
            showFailMessage(result.message);
        }
    } catch (error) {
        log(`✗ 错误: ${error.message}`, 'error');
    }
}

async function exampleGestureCaptcha() {
    log('开始手势验证码示例', 'info');

    try {
        const gesture = await captchaClient.getGestureCaptcha();
        log(`✓ Session ID: ${gesture.session_id}`, 'success');

        if (gesture.hint) {
            log(`✓ 提示: ${gesture.hint}`, 'info');
        }
        if (gesture.grid_size) {
            log(`✓ 网格大小: ${gesture.grid_size}`, 'info');
        }

        displayGestureCaptcha(gesture);

    } catch (error) {
        log(`✗ 错误: ${error.message}`, 'error');
    }
}

function displayGestureCaptcha(captcha) {
    const container = document.getElementById('captcha-display');
    if (!container) return;

    const points = [];
    const gridSize = captcha.grid_size || 3;
    const cellSize = 60;

    container.innerHTML = `
        <div class="captcha-container">
            <div class="captcha-instructions">
                <p>${captcha.hint || '按顺序点击网格中的点'}</p>
            </div>
            <div class="gesture-grid" id="gesture-grid"
                 style="display: grid; grid-template-columns: repeat(${gridSize}, ${cellSize}px); gap: 10px;">
            </div>
            <div class="captcha-controls">
                <button onclick="submitGestureCaptcha('${captcha.session_id}')" class="verify-btn">验证</button>
                <button onclick="refreshGestureCaptcha()" class="refresh-btn">刷新</button>
            </div>
        </div>
    `;

    const grid = document.getElementById('gesture-grid');

    for (let i = 0; i < gridSize * gridSize; i++) {
        const cell = document.createElement('div');
        cell.className = 'gesture-cell';
        cell.dataset.index = i;
        cell.textContent = i + 1;

        cell.addEventListener('click', () => {
            if (cell.classList.contains('selected')) return;

            points.push(i);
            cell.classList.add('selected');
            log(`点击: ${i + 1}`, 'info');
        });

        grid.appendChild(cell);
    }

    window.currentGesturePoints = points;
}

async function submitGestureCaptcha(sessionId) {
    const points = window.currentGesturePoints || [];

    if (points.length < 2) {
        log('请至少点击2个点', 'warn');
        return;
    }

    log('提交手势验证', 'info');

    try {
        const result = await captchaClient.verifyGestureCaptcha({
            session_id: sessionId,
            pattern: points
        });

        if (result.success) {
            log('✓ 验证成功!', 'success');
            showSuccessMessage('手势验证成功');
        } else {
            log(`✗ 验证失败: ${result.message}`, 'error');
        }
    } catch (error) {
        log(`✗ 错误: ${error.message}`, 'error');
    }
}

async function exampleUserAuth() {
    log('开始用户认证示例', 'info');

    const auth = captchaClient.auth();

    try {
        log('步骤1: 用户注册', 'info');
        const registerResult = await auth.register({
            username: 'testuser',
            email: 'test@example.com',
            password: 'password123'
        });
        log('✓ 注册成功', 'success');

        log('步骤2: 用户登录', 'info');
        const loginResult = await auth.login({
            username: 'testuser',
            password: 'password123'
        });

        if (loginResult.access_token) {
            log('✓ 登录成功', 'success');
            log(`  Token: ${loginResult.access_token.substring(0, 20)}...`, 'info');
            log(`  过期时间: ${loginResult.expires_in}秒`, 'info');

            log('步骤3: 获取检测脚本', 'info');
            const script = await captchaClient.getDetectionScript();
            log(`✓ 脚本长度: ${script.length}字符`, 'success');
        }

    } catch (error) {
        log(`✗ 错误: ${error.message}`, 'error');
    }
}

async function exampleEnvironmentDetection() {
    log('开始环境检测示例', 'info');

    const env = captchaClient.env();

    try {
        log('步骤1: 收集浏览器指纹', 'info');
        const browserData = env.collectBrowserData();

        log('浏览器数据:', 'info');
        log(`  User Agent: ${browserData.user_agent?.substring(0, 50)}...`, 'info');
        log(`  语言: ${browserData.language}`, 'info');
        log(`  平台: ${browserData.platform}`, 'info');
        log(`  屏幕: ${browserData.screen_width}x${browserData.screen_height}`, 'info');
        log(`  时区: ${browserData.timezone}`, 'info');
        log(`  WebDriver: ${browserData.is_webdriver}`, 'info');

        if (browserData.webgl_vendor) {
            log(`  WebGL厂商: ${browserData.webgl_vendor}`, 'info');
        }

        log('步骤2: 执行完整检测', 'info');
        const checkResult = await env.performFullCheck();
        log(`✓ 检测完成`, 'success');

        if (checkResult.risk_level) {
            log(`  风险等级: ${checkResult.risk_level}`, 'info');
        }

    } catch (error) {
        log(`✗ 错误: ${error.message}`, 'error');
    }
}

function exampleTrajectoryRecording() {
    log('开始轨迹记录示例', 'info');

    const recorder = captchaClient.recordTrajectory((points) => {
        log(`当前轨迹点: ${points.length}个`, 'info');
    });

    log('步骤1: 开始记录轨迹', 'info');
    recorder.start();

    log('请在页面上移动鼠标...', 'info');

    setTimeout(() => {
        log('步骤2: 停止记录', 'info');
        const finalPoints = recorder.stop();
        log(`✓ 记录完成，共 ${finalPoints.length} 个点`, 'success');

        if (finalPoints.length > 0) {
            log(`第一个点: x=${finalPoints[0].x}, y=${finalPoints[0].y}, t=${finalPoints[0].t}`, 'info');
            log(`最后一个点: x=${finalPoints[finalPoints.length-1].x}, y=${finalPoints[finalPoints.length-1].y}, t=${finalPoints[finalPoints.length-1].t}`, 'info');
        }

        recorder.destroy();
        log('✓ 轨迹记录器已销毁', 'success');

    }, 5000);
}

function handleError(error) {
    console.error('Captcha Error:', error);

    if (error.status === 401) {
        log('认证失败，请检查API密钥', 'error');
    } else if (error.status === 429) {
        log('请求过于频繁，请稍后再试', 'warn');
    } else if (error.status >= 500) {
        log('服务器错误，请稍后再试', 'error');
    }
}

function showSuccessMessage(message) {
    const container = document.getElementById('captcha-display');
    if (!container) return;

    container.innerHTML = `
        <div class="success-message">
            <div class="success-icon">✓</div>
            <p>${message}</p>
        </div>
    `;
}

function showFailMessage(message) {
    const container = document.getElementById('captcha-display');
    if (!container) return;

    container.innerHTML = `
        <div class="fail-message">
            <div class="fail-icon">✗</div>
            <p>${message || '验证失败'}</p>
        </div>
    `;
}

async function refreshSliderCaptcha() {
    await exampleSliderCaptcha();
}

async function refreshClickCaptcha() {
    await exampleClickCaptcha();
}

async function refreshImageCaptcha() {
    await exampleImageCaptcha();
}

async function refreshGestureCaptcha() {
    await exampleGestureCaptcha();
}

function runAllExamples() {
    log('开始运行所有示例...', 'info');

    exampleSliderCaptcha();
    setTimeout(() => exampleClickCaptcha(), 2000);
    setTimeout(() => exampleImageCaptcha(), 4000);
    setTimeout(() => exampleGestureCaptcha(), 6000);
    setTimeout(() => exampleUserAuth(), 8000);
    setTimeout(() => exampleEnvironmentDetection(), 10000);
}

if (typeof window !== 'undefined') {
    window.CaptchaClient = CaptchaClient;
    window.UserAuth = UserAuth;
    window.Environment = Environment;
    window.SliderCaptchaWidget = SliderCaptchaWidget;
    window.ClickCaptchaWidget = ClickCaptchaWidget;
    window.exampleSliderCaptcha = exampleSliderCaptcha;
    window.exampleClickCaptcha = exampleClickCaptcha;
    window.exampleImageCaptcha = exampleImageCaptcha;
    window.exampleGestureCaptcha = exampleGestureCaptcha;
    window.exampleUserAuth = exampleUserAuth;
    window.exampleEnvironmentDetection = exampleEnvironmentDetection;
    window.exampleTrajectoryRecording = exampleTrajectoryRecording;
    window.runAllExamples = runAllExamples;
    window.refreshSliderCaptcha = refreshSliderCaptcha;
    window.refreshClickCaptcha = refreshClickCaptcha;
    window.refreshImageCaptcha = refreshImageCaptcha;
    window.refreshGestureCaptcha = refreshGestureCaptcha;
    window.submitClickCaptcha = submitClickCaptcha;
    window.submitImageCaptcha = submitImageCaptcha;
    window.submitGestureCaptcha = submitGestureCaptcha;
}
