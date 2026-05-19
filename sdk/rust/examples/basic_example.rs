//! 基础示例
//!
//! 展示Rust SDK的基本用法。

use hjtpx_captcha::{CaptchaClient, CaptchaConfig, LoginRequest, TrajectoryPoint};
use std::time::{SystemTime, UNIX_EPOCH};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    println!("=" .repeat(50));
    println!("HJTPX Rust SDK 基础示例");
    println!("=" .repeat(50));

    let config = CaptchaConfig::new()
        .with_timeout(std::time::Duration::from_secs(30))
        .with_max_retries(3);

    let client = CaptchaClient::new("http://localhost:8080", config);

    // 示例1: 滑块验证码
    println!("\n[1] 滑块验证码示例");
    match client.get_slider_captcha(320, 160, 8).await {
        Ok(slider) => {
            println!("  会话ID: {}", slider.session_id);
            println!("  图片宽度: {:?}", slider.image_width);
            println!("  图片高度: {:?}", slider.image_height);

            let now = SystemTime::now()
                .duration_since(UNIX_EPOCH)?
                .as_millis() as i64;

            let trajectory = vec![
                TrajectoryPoint::new(0, slider.secret_y.unwrap_or(80), now - 1000),
                TrajectoryPoint::new(50, slider.secret_y.unwrap_or(80) + 5, now - 800),
                TrajectoryPoint::new(100, slider.secret_y.unwrap_or(80) - 3, now - 500),
                TrajectoryPoint::new(150, slider.secret_y.unwrap_or(80) + 2, now - 200),
                TrajectoryPoint::new(200, slider.secret_y.unwrap_or(80), now),
            ];

            match client
                .verify_slider_captcha(
                    &slider.session_id,
                    200,
                    slider.secret_y,
                    Some(trajectory),
                )
                .await
            {
                Ok(result) => {
                    println!("  验证结果: {}", result.success);
                    println!("  消息: {}", result.message);
                    if let Some(score) = result.risk_score {
                        println!("  风险分数: {:.2}", score);
                    }
                }
                Err(e) => println!("  验证失败: {}", e),
            }
        }
        Err(e) => println!("  获取验证码失败: {}", e),
    }

    // 示例2: 点击验证码
    println!("\n[2] 点击验证码示例");
    match client.get_click_captcha("number", 3, true).await {
        Ok(click) => {
            println!("  会话ID: {}", click.session_id);
            println!("  模式: {}", click.mode);
            println!("  最大点数: {}", click.max_points);
            println!("  提示: {}", click.hint);

            let points = vec![vec![100, 100], vec![200, 150], vec![300, 200]];
            match client
                .verify_click_captcha(&click.session_id, points, None)
                .await
            {
                Ok(result) => {
                    println!("  验证结果: {}", result.success);
                    println!("  消息: {}", result.message);
                }
                Err(e) => println!("  验证失败: {}", e),
            }
        }
        Err(e) => println!("  获取验证码失败: {}", e),
    }

    // 示例3: 图形验证码
    println!("\n[3] 图形验证码示例");
    match client.get_image_captcha("mixed", 4).await {
        Ok(image) => {
            println!("  挑战ID: {}", image.challenge_id);
            println!("  图片长度: {} bytes", image.image.len());

            match client
                .verify_image_captcha(&image.challenge_id, "TEST")
                .await
            {
                Ok(result) => {
                    println!("  验证结果: {}", result.success);
                    println!("  消息: {}", result.message);
                }
                Err(e) => println!("  验证失败: {}", e),
            }
        }
        Err(e) => println!("  获取验证码失败: {}", e),
    }

    // 示例4: 手势验证码
    println!("\n[4] 手势验证码示例");
    match client.get_gesture_captcha().await {
        Ok(gesture) => {
            println!("  会话ID: {}", gesture.session_id);
            println!("  网格大小: {:?}", gesture.grid_size);

            let pattern = vec![1, 2, 3, 6, 9, 8, 7, 4];
            match client
                .verify_gesture_captcha(&gesture.session_id, pattern)
                .await
            {
                Ok(result) => {
                    println!("  验证结果: {}", result.success);
                    println!("  消息: {}", result.message);
                }
                Err(e) => println!("  验证失败: {}", e),
            }
        }
        Err(e) => println!("  获取验证码失败: {}", e),
    }

    // 示例5: 拼图验证码
    println!("\n[5] 拼图验证码示例");
    match client.get_jigsaw_captcha(300, 300, 3).await {
        Ok(jigsaw) => {
            println!("  会话ID: {}", jigsaw.session_id);
            println!("  碎片数量: {}", jigsaw.pieces.len());
            println!("  网格大小: {}", jigsaw.grid_size);

            let pieces: Vec<_> = jigsaw
                .pieces
                .iter()
                .take(3)
                .map(|p| hjtpx_captcha::JigsawPiece {
                    index: p.index,
                    original_x: p.original_x,
                    original_y: p.original_y,
                    current_x: p.current_x,
                    current_y: p.current_y,
                    width: p.width,
                    height: p.height,
                    rotation: p.rotation,
                })
                .collect();

            match client
                .verify_jigsaw_captcha(&jigsaw.session_id, pieces)
                .await
            {
                Ok(result) => {
                    println!("  验证结果: {}", result.success);
                    println!("  消息: {}", result.message);
                }
                Err(e) => println!("  验证失败: {}", e),
            }
        }
        Err(e) => println!("  获取验证码失败: {}", e),
    }

    // 示例6: 用户登录
    println!("\n[6] 用户登录示例");
    let login_req = LoginRequest {
        username: "testuser".to_string(),
        password: "password123".to_string(),
        captcha_token: None,
    };

    match client.login(&login_req).await {
        Ok(response) => {
            println!("  登录成功!");
            println!("  用户名: {}", response.user.username);
            println!("  邮箱: {}", response.user.email);
            println!("  令牌过期时间: {} 秒", response.expires_in);

            match client.logout().await {
                Ok(_) => println!("  登出成功!"),
                Err(e) => println!("  登出失败: {}", e),
            }
        }
        Err(e) => println!("  登录失败: {}", e),
    }

    println!("\n" + "=" .repeat(50));
    println!("示例执行完成");
    println!("=" .repeat(50));

    Ok(())
}
