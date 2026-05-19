//! 异步并发示例
//!
//! 展示Rust SDK的异步并发能力。

use hjtpx_captcha::{CaptchaClient, CaptchaConfig, TrajectoryPoint};
use std::time::{SystemTime, UNIX_EPOCH};
use tokio::time::{sleep, Duration};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    println!("=" .repeat(50));
    println!("HJTPX Rust SDK 异步并发示例");
    println!("=" .repeat(50));

    let config = CaptchaConfig::new()
        .with_timeout(Duration::from_secs(30))
        .with_max_retries(3)
        .with_pool_size(20);

    let client = CaptchaClient::new("http://localhost:8080", config);

    // 示例1: 并发获取多个验证码
    println!("\n[1] 并发获取验证码示例");
    let start = std::time::Instant::now();

    let tasks: Vec<_> = (0..10)
        .map(|i| {
            let client = client.clone();
            async move {
                match client.get_slider_captcha(320, 160, 8).await {
                    Ok(slider) => {
                        println!("  [任务{}] 成功: {}", i, slider.session_id);
                        Ok::<_, ()>(slider.session_id)
                    }
                    Err(e) => {
                        println!("  [任务{}] 失败: {}", i, e);
                        Err::<String, ()>(())
                    }
                }
            }
        })
        .collect();

    let results = futures::future::join_all(tasks).await;
    let success_count = results.iter().filter(|r| r.is_ok()).count();
    println!(
        "  完成! 成功率: {}/{}, 耗时: {:?}",
        success_count,
        results.len(),
        start.elapsed()
    );

    // 示例2: 批量验证
    println!("\n[2] 批量验证示例");
    let sliders: Vec<_> = (0..5)
        .filter_map(|_| client.get_slider_captcha(320, 160, 8).await.ok())
        .collect();

    if !sliders.is_empty() {
        let verify_tasks: Vec<_> = sliders
            .iter()
            .enumerate()
            .map(|(i, slider)| {
                let client = client.clone();
                let session_id = slider.session_id.clone();
                let secret_y = slider.secret_y;

                async move {
                    sleep(Duration::from_millis(100 * i as u64)).await;

                    let now = SystemTime::now()
                        .duration_since(UNIX_EPOCH)?
                        .as_millis() as i64;

                    let trajectory = vec![
                        TrajectoryPoint::new(0, secret_y.unwrap_or(80), now - 1000),
                        TrajectoryPoint::new(60, secret_y.unwrap_or(80), now - 600),
                        TrajectoryPoint::new(120, secret_y.unwrap_or(80), now - 300),
                        TrajectoryPoint::new(180, secret_y.unwrap_or(80), now),
                    ];

                    match client
                        .verify_slider_captcha(&session_id, 180, secret_y, Some(trajectory))
                        .await
                    {
                        Ok(result) => {
                            println!("  [验证{}] 成功: {}", i, result.success);
                            Ok(result)
                        }
                        Err(e) => {
                            println!("  [验证{}] 失败: {}", i, e);
                            Err(e)
                        }
                    }
                }
            })
            .collect();

        let verify_results = futures::future::join_all(verify_tasks).await;
        let verify_success = verify_results.iter().filter(|r| r.is_ok()).count();
        println!(
            "  批量验证完成! 成功率: {}/{}",
            verify_success,
            verify_results.len()
        );
    }

    // 示例3: 混合验证码并发
    println!("\n[3] 混合验证码并发示例");
    let mixed_tasks = vec![
        client.get_slider_captcha(320, 160, 8),
        client.get_click_captcha("number", 3, true),
        client.get_image_captcha("mixed", 4),
        client.get_gesture_captcha(),
        client.get_rotation_captcha(),
    ];

    let start = std::time::Instant::now();
    let mixed_results = futures::future::join_all(mixed_tasks).await;
    let mixed_success = mixed_results.iter().filter(|r| r.is_ok()).count();

    println!(
        "  混合验证码获取完成! 成功率: {}/{}, 耗时: {:?}",
        mixed_success,
        mixed_results.len(),
        start.elapsed()
    );

    for (i, result) in mixed_results.iter().enumerate() {
        match result {
            Ok(_) => println!("  [验证码{}] 获取成功", i + 1),
            Err(e) => println!("  [验证码{}] 获取失败: {}", i + 1, e),
        }
    }

    // 示例4: 错误处理和重试
    println!("\n[4] 错误处理示例");
    for i in 0..3 {
        match client
            .verify_slider_captcha("invalid-session-id", 100, None, None)
            .await
        {
            Ok(result) => {
                println!("  [尝试{}] 验证结果: {}", i + 1, result.success);
            }
            Err(e) => {
                println!("  [尝试{}] 错误: {}", i + 1, e);
            }
        }
        sleep(Duration::from_millis(500)).await;
    }

    // 示例5: 带超时的操作
    println!("\n[5] 超时控制示例");
    let quick_client = CaptchaClient::new(
        "http://localhost:8080",
        CaptchaConfig::new().with_timeout(Duration::from_millis(100)),
    );

    match quick_client.get_slider_captcha(320, 160, 8).await {
        Ok(_) => println!("  快速请求成功"),
        Err(hjtpx_captcha::CaptchaError::TimeoutError) => {
            println!("  请求超时!")
        }
        Err(e) => println!("  其他错误: {}", e),
    }

    println!("\n" + "=" .repeat(50));
    println!("异步示例执行完成");
    println!("=" .repeat(50));

    Ok(())
}
