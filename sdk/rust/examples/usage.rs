use hjtpx_captcha::{CaptchaClient, CaptchaError, Result};
use std::time::Duration;

#[tokio::main]
async fn main() -> Result<()> {
    println!("========================================");
    println!("   Hjtpx Captcha Rust SDK v15.0 Examples");
    println!("========================================\n");

    let client = CaptchaClient::new("http://localhost:8080")
        .with_api_key("your-api-key")
        .with_timeout(Duration::from_secs(30))
        .with_retry_config(hjtpx_captcha::utils::RetryConfig {
            max_retries: 3,
            base_delay_ms: 100,
            max_delay_ms: 5000,
        });

    slider_captcha_example(&client).await?;
    click_captcha_example(&client).await?;
    image_captcha_example(&client).await?;
    batch_verify_example(&client).await?;
    async_verify_example(&client).await?;
    authentication_example(&client).await?;

    println!("========================================");
    println!("             Examples Complete");
    println!("========================================");

    Ok(())
}

async fn slider_captcha_example(client: &CaptchaClient) -> Result<()> {
    println!("=== Slider Captcha Example ===");

    let captcha = client.get_slider_captcha(320, 160, 8).await?;
    println!("Session ID: {}", captcha.session_id);
    println!("Image URL: {}", captcha.image_url);
    println!("Secret Y: {:?}", captcha.secret_y);

    let trajectory = vec![
        hjtpx_captcha::TrajectoryPoint {
            x: 0,
            y: captcha.secret_y.unwrap_or(50),
            t: chrono::Utc::now().timestamp_millis() - 1000,
        },
        hjtpx_captcha::TrajectoryPoint {
            x: 50,
            y: captcha.secret_y.unwrap_or(50) + 5,
            t: chrono::Utc::now().timestamp_millis() - 800,
        },
        hjtpx_captcha::TrajectoryPoint {
            x: 100,
            y: captcha.secret_y.unwrap_or(50) - 3,
            t: chrono::Utc::now().timestamp_millis() - 500,
        },
        hjtpx_captcha::TrajectoryPoint {
            x: 150,
            y: captcha.secret_y.unwrap_or(50) + 2,
            t: chrono::Utc::now().timestamp_millis() - 200,
        },
        hjtpx_captcha::TrajectoryPoint {
            x: 185,
            y: captcha.secret_y.unwrap_or(50),
            t: chrono::Utc::now().timestamp_millis(),
        },
    ];

    let result = client
        .verify_slider_captcha(
            &captcha.session_id,
            185,
            captcha.secret_y,
            Some(trajectory),
        )
        .await?;

    println!(
        "Verification Success: {}",
        if result.success { "Yes" } else { "No" }
    );
    println!("Message: {}", result.message);
    println!();

    Ok(())
}

async fn click_captcha_example(client: &CaptchaClient) -> Result<()> {
    println!("=== Click Captcha Example ===");

    let captcha = client.get_click_captcha("number", 3, true).await?;
    println!("Session ID: {}", captcha.session_id);
    println!("Image URL: {}", captcha.image_url);
    println!("Hint: {}", captcha.hint);
    println!("Hint Order: {:?}", captcha.hint_order);

    let points = vec![vec![100, 100], vec![200, 150], vec![150, 200]];
    let click_sequence = vec![1, 2, 3];

    let result = client
        .verify_click_captcha(&captcha.session_id, points, Some(click_sequence))
        .await?;

    println!(
        "Verification Success: {}",
        if result.success { "Yes" } else { "No" }
    );
    println!();

    Ok(())
}

async fn image_captcha_example(client: &CaptchaClient) -> Result<()> {
    println!("=== Image Captcha Example ===");

    let captcha = client.get_image_captcha("mixed", 4).await?;
    println!("Challenge ID: {}", captcha.challenge_id);
    println!(
        "Image (base64): {}...",
        &captcha.image[..captcha.image.len().min(50)]
    );

    let result = client
        .verify_image_captcha(&captcha.challenge_id, "ABCD")
        .await?;

    println!(
        "Verification Success: {}",
        if result.success { "Yes" } else { "No" }
    );
    println!();

    Ok(())
}

async fn batch_verify_example(client: &CaptchaClient) -> Result<()> {
    println!("=== Batch Verify Example ===");

    let requests = vec![
        hjtpx_captcha::VerifyCaptchaRequest {
            session_id: "session-1".to_string(),
            x: 100,
            y: Some(50),
            trajectory: None,
            r#type: "slider".to_string(),
        },
        hjtpx_captcha::VerifyCaptchaRequest {
            session_id: "session-2".to_string(),
            x: 150,
            y: Some(60),
            trajectory: None,
            r#type: "slider".to_string(),
        },
        hjtpx_captcha::VerifyCaptchaRequest {
            session_id: "session-3".to_string(),
            x: 200,
            y: Some(70),
            trajectory: None,
            r#type: "slider".to_string(),
        },
    ];

    let result = client.batch_verify(requests).await?;

    println!("Success Count: {}", result.success_count);
    println!("Failed Count: {}", result.failed_count);
    println!("Total Time: {}ms", result.total_time_ms);

    for r in &result.results {
        println!(
            "- Session {}: {}",
            r.session_id,
            if r.success { "Success" } else { "Failed" }
        );
    }
    println!();

    Ok(())
}

async fn async_verify_example(client: &CaptchaClient) -> Result<()> {
    println!("=== Async Verify Example ===");

    let async_result = client
        .async_verify(
            "session-async-1",
            150,
            Some(50),
            None,
            Some("https://example.com/callback".to_string()),
        )
        .await?;

    println!("Task ID: {}", async_result.task_id);
    println!("Status: {}", async_result.status);

    let final_result = client
        .wait_async_result(&async_result.task_id, 10, 500)
        .await?;

    println!("Final Status: {}", final_result.status);
    if let Some(result) = final_result.result {
        println!("Verification Success: {}", result.success);
    }
    println!();

    Ok(())
}

async fn authentication_example(client: &CaptchaClient) -> Result<()> {
    println!("=== Authentication Example ===");

    let result = client
        .login("username", "password", None)
        .await?;

    println!("Access Token: {}...", &result.access_token[..result.access_token.len().min(20)]);
    println!("Expires In: {}s", result.expires_in);
    println!("User: {}", result.user.username);

    client.logout().await?;
    println!("Logged out successfully\n");

    Ok(())
}
