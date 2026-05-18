/**
 * HJT Captcha JavaScript SDK 浏览器端完整示例
 * 
 * 本文件包含多种浏览器端集成示例，可直接在HTML页面中使用
 */

// 初始化客户端
const client = new CaptchaClient('http://localhost:8080', {
    timeout: 30000,
    retryCount: 3,
});

/**
 * 示例1: 滑块验证码集成
 */
async function sliderCaptchaExample() {
    console.log('=== 滑块验证码示例 ===');
    
    try {
        // 获取滑块验证码
        const captcha = await client.getSliderCaptcha({ width: 320, height: 160 });
        console.log('获取验证码成功:', captcha.session_id);
        
        // 模拟用户滑动轨迹
        const trajectory = [
            { x: 0, y: captcha.secret_y || 80, t: 0 },
            { x: 50, y: (captcha.secret_y || 80) + 5, t: 200 },
            { x: 100, y: (captcha.secret_y || 80) - 3, t: 400 },
            { x: 150, y: (captcha.secret_y || 80) + 2, t: 600 },
            { x: 185, y: captcha.secret_y || 80, t: 800 },
        ];
        
        // 验证验证码
        const result = await client.verifySliderCaptcha({
            session_id: captcha.session_id,
            x: 185,
            y: captcha.secret_y,
            trajectory: trajectory,
        });
        
        console.log('验证结果:', result.success ? '成功' : '失败');
        console.log('消息:', result.message);
        
    } catch (error) {
        console.error('滑块验证码示例失败:', error.message);
    }
}

/**
 * 示例2: 点击验证码集成
 */
async function clickCaptchaExample() {
    console.log('\n=== 点击验证码示例 ===');
    
    try {
        // 获取点击验证码
        const captcha = await client.getClickCaptcha({
            mode: 'number',
            max_points: 3,
            shuffle: true,
        });
        console.log('获取验证码成功:', captcha.session_id);
        console.log('提示:', captcha.hint);
        
        // 模拟用户点击（实际应用中由用户交互获取）
        const clicks = [
            [100, 100],
            [200, 150],
            [150, 200],
        ];
        
        // 验证验证码
        const result = await client.verifyClickCaptcha({
            session_id: captcha.session_id,
            points: clicks,
            click_sequence: [0, 1, 2],
        });
        
        console.log('验证结果:', result.success ? '成功' : '失败');
        
    } catch (error) {
        console.error('点击验证码示例失败:', error.message);
    }
}

/**
 * 示例3: 图形验证码集成
 */
async function imageCaptchaExample() {
    console.log('\n=== 图形验证码示例 ===');
    
    try {
        // 获取图形验证码
        const captcha = await client.getImageCaptcha({
            type: 'mixed',
            count: 4,
        });
        console.log('获取验证码成功:', captcha.challenge_id);
        console.log('图片长度:', captcha.image.length, '字符');
        
        // 模拟用户输入（实际应用中由用户输入）
        const userAnswer = 'ABCD';
        
        // 验证验证码
        const result = await client.verifyImageCaptcha(captcha.challenge_id, userAnswer);
        console.log('验证结果:', result.success ? '成功' : '失败');
        
    } catch (error) {
        console.error('图形验证码示例失败:', error.message);
    }
}

/**
 * 示例4: 手势验证码集成
 */
async function gestureCaptchaExample() {
    console.log('\n=== 手势验证码示例 ===');
    
    try {
        // 获取手势验证码
        const captcha = await client.getGestureCaptcha();
        console.log('获取验证码成功:', captcha.session_id);
        if (captcha.hint) {
            console.log('提示:', captcha.hint);
        }
        
        // 模拟手势模式（实际应用中由用户绘制）
        const pattern = [1, 2, 3, 5, 7];
        
        // 验证验证码
        const result = await client.verifyGestureCaptcha({
            session_id: captcha.session_id,
            pattern: pattern,
        });
        
        console.log('验证结果:', result.success ? '成功' : '失败');
        
    } catch (error) {
        console.error('手势验证码示例失败:', error.message);
    }
}

/**
 * 示例5: 轨迹记录功能
 */
function trajectoryRecordingExample() {
    console.log('\n=== 轨迹记录示例 ===');
    
    // 获取滑块容器元素
    const sliderContainer = document.getElementById('slider-container');
    
    // 创建轨迹记录器
    const recorder = client.recordTrajectory((points) => {
        // 实时回调，每次轨迹更新时触发
        console.log('轨迹点数量:', points.length);
    }, sliderContainer);
    
    // 使用方式：
    // recorder.start();  // 开始记录
    // const points = recorder.stop();  // 停止记录并获取轨迹
    // recorder.reset();  // 重置轨迹
    // recorder.destroy();  // 销毁记录器
    
    console.log('轨迹记录器已创建');
    return recorder;
}

/**
 * 示例6: 用户认证功能
 */
async function userAuthExample() {
    console.log('\n=== 用户认证示例 ===');
    
    const auth = client.auth();
    
    try {
        // 登录
        const loginResult = await auth.login({
            username: 'testuser',
            password: 'password123',
        });
        console.log('登录成功!');
        console.log('访问令牌:', loginResult.access_token?.substring(0, 20) + '...');
        
        // 刷新令牌
        const refreshResult = await auth.refreshToken();
        console.log('令牌刷新成功!');
        
        // 登出
        await auth.logout();
        console.log('登出成功!');
        
    } catch (error) {
        console.error('用户认证示例失败:', error.message);
    }
}

/**
 * 示例7: 环境检测功能
 */
async function environmentDetectionExample() {
    console.log('\n=== 环境检测示例 ===');
    
    const env = client.env();
    
    try {
        // 获取检测脚本
        const script = await env.getDetectionScript('onDetectReady');
        console.log('检测脚本长度:', script.length, '字符');
        
        // 收集浏览器数据
        const browserData = env.collectBrowserData();
        console.log('浏览器数据:', {
            userAgent: browserData.user_agent?.substring(0, 50) + '...',
            screenWidth: browserData.screen_width,
            screenHeight: browserData.screen_height,
        });
        
        // 提交检测数据
        const result = await env.submitDetection({
            detection_id: 'test-' + Date.now(),
            risk_score: 0.1,
            fingerprint: browserData.canvas_hash?.substring(0, 30) + '...',
            timestamp: Date.now(),
        });
        console.log('检测数据提交成功:', result);
        
    } catch (error) {
        console.error('环境检测示例失败:', error.message);
    }
}

/**
 * 示例8: UI组件 - 滑块验证码组件
 */
function sliderWidgetExample() {
    console.log('\n=== 滑块UI组件示例 ===');
    
    const container = document.getElementById('slider-widget-container');
    if (!container) {
        console.error('请创建滑块容器元素');
        return;
    }
    
    // 创建滑块验证码组件
    const sliderWidget = new SliderCaptchaWidget(container, client, {
        width: 320,
        height: 160,
        tolerance: 8,
        onSuccess: (result) => {
            console.log('滑块验证成功:', result);
            // 验证成功后的业务逻辑
            alert('验证成功!');
        },
        onFail: (message) => {
            console.log('滑块验证失败:', message);
        },
    });
    
    return sliderWidget;
}

/**
 * 示例9: UI组件 - 点击验证码组件
 */
function clickWidgetExample() {
    console.log('\n=== 点击UI组件示例 ===');
    
    const container = document.getElementById('click-widget-container');
    if (!container) {
        console.error('请创建点击容器元素');
        return;
    }
    
    // 创建点击验证码组件
    const clickWidget = new ClickCaptchaWidget(container, client, {
        mode: 'number',
        points: 3,
        shuffle: true,
        onSuccess: (result) => {
            console.log('点击验证成功:', result);
            alert('验证成功!');
        },
        onFail: (message) => {
            console.log('点击验证失败:', message);
        },
    });
    
    return clickWidget;
}

/**
 * 示例10: 错误处理
 */
async function errorHandlingExample() {
    console.log('\n=== 错误处理示例 ===');
    
    try {
        // 尝试连接到不存在的服务
        const badClient = new CaptchaClient('http://nonexistent.example.com', {
            timeout: 5000,
        });
        
        await badClient.getSliderCaptcha();
        
    } catch (error) {
        if (error.status === 404) {
            console.error('服务不存在:', error.message);
        } else if (error.message.includes('timeout')) {
            console.error('请求超时:', error.message);
        } else {
            console.error('未知错误:', error.message);
        }
    }
}

/**
 * 示例11: 设置访问令牌
 */
function setTokenExample() {
    console.log('\n=== 设置访问令牌示例 ===');
    
    // 设置令牌（通常从登录响应获取）
    client.setToken('your-jwt-token-here');
    console.log('访问令牌已设置');
    
    // 后续请求会自动携带令牌
    // const result = await client.someProtectedApi();
}

/**
 * 运行所有示例
 */
async function runAllExamples() {
    console.log('='.repeat(50));
    console.log('HJT Captcha JavaScript SDK 浏览器端示例');
    console.log('='.repeat(50));
    
    // 运行异步示例
    await sliderCaptchaExample();
    await clickCaptchaExample();
    await imageCaptchaExample();
    await gestureCaptchaExample();
    await userAuthExample();
    await environmentDetectionExample();
    await errorHandlingExample();
    
    // 运行同步/UI示例
    setTokenExample();
    
    // 注意：UI组件示例需要页面上有对应的DOM元素
    // sliderWidgetExample();
    // clickWidgetExample();
    // trajectoryRecordingExample();
    
    console.log('\n' + '='.repeat(50));
    console.log('所有示例运行完成');
    console.log('='.repeat(50));
}

// 如果在浏览器环境中自动运行示例
if (typeof window !== 'undefined') {
    // 页面加载完成后运行示例
    document.addEventListener('DOMContentLoaded', () => {
        // 可以选择性地调用特定示例
        // sliderCaptchaExample();
        
        // 或者运行所有示例
        // runAllExamples();
    });
}

// 导出示例函数供外部使用
if (typeof module !== 'undefined' && module.exports) {
    module.exports = {
        sliderCaptchaExample,
        clickCaptchaExample,
        imageCaptchaExample,
        gestureCaptchaExample,
        trajectoryRecordingExample,
        userAuthExample,
        environmentDetectionExample,
        sliderWidgetExample,
        clickWidgetExample,
        errorHandlingExample,
        setTokenExample,
        runAllExamples,
    };
}