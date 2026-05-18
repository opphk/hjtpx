/**
 * HJT Captcha Node.js SDK 完整示例
 * 
 * 本文件包含所有验证码类型的使用示例
 */

import { CaptchaClient } from '../src';

// 初始化客户端
const client = new CaptchaClient({
    baseUrl: 'http://localhost:8080',
    timeout: 30000,
    maxConnections: 100,
    retryConfig: {
        maxRetries: 3,
        initialDelayMs: 100,
        maxDelayMs: 5000,
    },
});

/**
 * 示例1: 滑块验证码
 */
async function sliderCaptchaExample() {
    console.log('=== 滑块验证码示例 ===');
    
    try {
        // 获取滑块验证码
        const captcha = await client.getSliderCaptcha({
            width: 320,
            height: 160,
            tolerance: 8,
        });
        console.log(`会话ID: ${captcha.session_id}`);
        console.log(`图片URL: ${captcha.image_url}`);
        
        // 模拟用户滑动轨迹
        const trajectory = [
            { x: 0, y: captcha.secret_y || 80, t: Date.now() - 1000 },
            { x: 50, y: (captcha.secret_y || 80) + 5, t: Date.now() - 800 },
            { x: 100, y: (captcha.secret_y || 80) - 3, t: Date.now() - 500 },
            { x: 150, y: (captcha.secret_y || 80) + 2, t: Date.now() - 200 },
            { x: 185, y: captcha.secret_y || 80, t: Date.now() },
        ];
        
        // 验证验证码
        const result = await client.verifySliderCaptcha(captcha.session_id, 185, {
            y: captcha.secret_y,
            trajectory,
        });
        
        console.log(`验证结果: ${result.success ? '成功' : '失败'}`);
        console.log(`消息: ${result.message}`);
        
    } catch (error) {
        console.error('滑块验证码示例失败:', error);
    }
}

/**
 * 示例2: 点击验证码
 */
async function clickCaptchaExample() {
    console.log('\n=== 点击验证码示例 ===');
    
    try {
        // 获取点击验证码
        const captcha = await client.getClickCaptcha({
            mode: 'number',
            shuffle: true,
            points: 3,
        });
        console.log(`会话ID: ${captcha.session_id}`);
        console.log(`提示: ${captcha.hint}`);
        
        // 模拟用户点击
        const clicks: [number, number][] = [
            [100, 100],
            [200, 150],
            [150, 200],
        ];
        
        // 验证验证码
        const result = await client.verifyClickCaptcha(captcha.session_id, clicks, {
            clickSequence: [0, 1, 2],
        });
        
        console.log(`验证结果: ${result.success ? '成功' : '失败'}`);
        
    } catch (error) {
        console.error('点击验证码示例失败:', error);
    }
}

/**
 * 示例3: 图形验证码
 */
async function imageCaptchaExample() {
    console.log('\n=== 图形验证码示例 ===');
    
    try {
        // 获取图形验证码
        const captcha = await client.getImageCaptcha({
            type: 'mixed',
            count: 4,
            noiseMode: 2,
            lineMode: 1,
        });
        console.log(`挑战ID: ${captcha.challenge_id}`);
        console.log(`图片数据长度: ${captcha.image.length} 字符`);
        
        // 模拟用户输入
        const answer = 'ABCD';
        
        // 验证验证码
        const result = await client.verifyImageCaptcha(captcha.challenge_id, answer);
        console.log(`验证结果: ${result.success ? '成功' : '失败'}`);
        
    } catch (error) {
        console.error('图形验证码示例失败:', error);
    }
}

/**
 * 示例4: 旋转验证码
 */
async function rotationCaptchaExample() {
    console.log('\n=== 旋转验证码示例 ===');
    
    try {
        // 获取旋转验证码
        const captcha = await client.getRotationCaptcha();
        console.log(`挑战ID: ${captcha.challenge_id}`);
        
        // 模拟用户旋转角度
        const angle = 90;
        
        // 验证验证码
        const result = await client.verifyRotationCaptcha(captcha.challenge_id, angle);
        console.log(`验证结果: ${result.success ? '成功' : '失败'}`);
        
    } catch (error) {
        console.error('旋转验证码示例失败:', error);
    }
}

/**
 * 示例5: 手势验证码
 */
async function gestureCaptchaExample() {
    console.log('\n=== 手势验证码示例 ===');
    
    try {
        // 获取手势验证码
        const captcha = await client.getGestureCaptcha();
        console.log(`会话ID: ${captcha.session_id}`);
        if (captcha.hint) {
            console.log(`提示: ${captcha.hint}`);
        }
        
        // 模拟手势模式（9宫格手势）
        const pattern = [1, 2, 3, 5, 7];
        
        // 验证验证码
        const result = await client.verifyGestureCaptcha(captcha.session_id, pattern);
        console.log(`验证结果: ${result.success ? '成功' : '失败'}`);
        
    } catch (error) {
        console.error('手势验证码示例失败:', error);
    }
}

/**
 * 示例6: 拼图验证码
 */
async function jigsawCaptchaExample() {
    console.log('\n=== 拼图验证码示例 ===');
    
    try {
        // 获取拼图验证码
        const captcha = await client.getJigsawCaptcha({
            width: 300,
            height: 300,
            gridSize: 3,
        });
        console.log(`会话ID: ${captcha.session_id}`);
        console.log(`网格大小: ${captcha.grid_size}x${captcha.grid_size}`);
        console.log(`碎片数量: ${captcha.pieces.length}`);
        
        // 模拟用户移动拼图（将碎片移回正确位置）
        const solvedPieces = captcha.pieces.map(piece => ({
            ...piece,
            current_x: piece.original_x,
            current_y: piece.original_y,
            rotation: 0,
        }));
        
        // 验证验证码
        const result = await client.verifyJigsawCaptcha(captcha.session_id, solvedPieces);
        console.log(`验证结果: ${result.success ? '成功' : '失败'}`);
        
    } catch (error) {
        console.error('拼图验证码示例失败:', error);
    }
}

/**
 * 示例7: 用户认证
 */
async function userAuthExample() {
    console.log('\n=== 用户认证示例 ===');
    
    try {
        // 登录
        const loginResult = await client.authLogin({
            username: 'testuser',
            password: 'password123',
        });
        console.log('登录成功!');
        console.log(`访问令牌: ${loginResult.access_token.substring(0, 20)}...`);
        console.log(`过期时间: ${loginResult.expires_in}秒`);
        
        // 刷新令牌
        const refreshResult = await client.authRefreshToken();
        console.log('令牌刷新成功!');
        
        // 登出
        await client.authLogout();
        console.log('登出成功!');
        
    } catch (error) {
        console.error('用户认证示例失败:', error);
    }
}

/**
 * 示例8: 环境检测
 */
async function environmentDetectionExample() {
    console.log('\n=== 环境检测示例 ===');
    
    try {
        // 获取检测脚本
        const script = await client.getDetectionScript('onDetectReady');
        console.log(`检测脚本长度: ${script.length} 字符`);
        
        // 提交检测数据
        const detectionData = {
            detection_id: 'test-' + Date.now(),
            risk_score: 0.1,
            fingerprint: 'user-fingerprint',
            timestamp: Date.now(),
        };
        const result = await client.submitDetection(detectionData);
        console.log('检测数据提交成功:', result);
        
    } catch (error) {
        console.error('环境检测示例失败:', error);
    }
}

/**
 * 示例9: 错误处理
 */
async function errorHandlingExample() {
    console.log('\n=== 错误处理示例 ===');
    
    try {
        // 使用无效的会话ID进行验证
        await client.verifySliderCaptcha('invalid-session-id', 100);
        
    } catch (error: any) {
        console.error(`错误类型: ${error.constructor.name}`);
        console.error(`错误消息: ${error.message}`);
        
        if (error.status) {
            console.error(`HTTP状态码: ${error.status}`);
        }
    }
}

/**
 * 运行所有示例
 */
async function runAllExamples() {
    console.log('='.repeat(60));
    console.log('HJT Captcha Node.js SDK 完整示例');
    console.log('='.repeat(60));
    
    await sliderCaptchaExample();
    await clickCaptchaExample();
    await imageCaptchaExample();
    await rotationCaptchaExample();
    await gestureCaptchaExample();
    await jigsawCaptchaExample();
    await userAuthExample();
    await environmentDetectionExample();
    await errorHandlingExample();
    
    // 关闭客户端
    await client.close();
    
    console.log('\n' + '='.repeat(60));
    console.log('所有示例运行完成');
    console.log('='.repeat(60));
}

// 运行示例
runAllExamples().catch(console.error);