#[cfg(test)]
mod tests {
    use hjtpx_captcha::*;
    use mockito::{Server, Mock, matcher};
    use tokio;

    #[tokio::test]
    async fn test_new_client() {
        let client = CaptchaClient::new("http://localhost:8080");
        assert!(true);
    }

    #[tokio::test]
    async fn test_client_with_options() {
        let client = CaptchaClient::new("http://localhost:8080")
            .with_api_key("test-api-key")
            .with_timeout(std::time::Duration::from_secs(60));

        assert!(true);
    }

    #[tokio::test]
    async fn test_slider_captcha_response_deserialization() {
        let json = r#"{
            "session_id": "test-session-123",
            "image_url": "http://example.com/image.png",
            "puzzle_url": "http://example.com/puzzle.png",
            "secret_y": 50,
            "image_width": 320,
            "image_height": 160
        }"#;

        let response: SliderCaptchaResponse = serde_json::from_str(json).unwrap();
        assert_eq!(response.session_id, "test-session-123");
        assert_eq!(response.image_url, "http://example.com/image.png");
        assert_eq!(response.secret_y, Some(50));
    }

    #[tokio::test]
    async fn test_verify_captcha_response_deserialization() {
        let json = r#"{
            "success": true,
            "message": "Verification passed",
            "remaining_attempts": 3,
            "risk_score": 0.15
        }"#;

        let response: VerifyCaptchaResponse = serde_json::from_str(json).unwrap();
        assert!(response.success);
        assert_eq!(response.message, "Verification passed");
        assert_eq!(response.remaining_attempts, Some(3));
    }

    #[tokio::test]
    async fn test_trajectory_point() {
        let point = TrajectoryPoint {
            x: 100,
            y: 50,
            t: 1700000000000,
        };

        let json = serde_json::to_string(&point).unwrap();
        let deserialized: TrajectoryPoint = serde_json::from_str(&json).unwrap();

        assert_eq!(deserialized.x, 100);
        assert_eq!(deserialized.y, 50);
        assert_eq!(deserialized.t, 1700000000000);
    }

    #[tokio::test]
    async fn test_batch_verify_response() {
        let json = r#"{
            "results": [
                {"session_id": "s1", "success": true, "message": "OK"},
                {"session_id": "s2", "success": false, "message": "Failed"}
            ],
            "success_count": 1,
            "failed_count": 1,
            "total_time_ms": 150
        }"#;

        let response: BatchVerifyResponse = serde_json::from_str(json).unwrap();
        assert_eq!(response.results.len(), 2);
        assert_eq!(response.success_count, 1);
        assert_eq!(response.failed_count, 1);
        assert_eq!(response.total_time_ms, 150);
    }

    #[tokio::test]
    async fn test_async_verify_response() {
        let json = r#"{
            "task_id": "task-123",
            "status": "pending",
            "created_at": 1700000000
        }"#;

        let response: AsyncVerifyResponse = serde_json::from_str(json).unwrap();
        assert_eq!(response.task_id, "task-123");
        assert_eq!(response.status, "pending");
        assert_eq!(response.created_at, 1700000000);
    }

    #[tokio::test]
    async fn test_async_result_response() {
        let json = r#"{
            "task_id": "task-123",
            "status": "completed",
            "result": {
                "success": true,
                "message": "OK"
            },
            "completed_at": 1700000001
        }"#;

        let response: AsyncResultResponse = serde_json::from_str(json).unwrap();
        assert_eq!(response.task_id, "task-123");
        assert_eq!(response.status, "completed");
        assert!(response.result.is_some());
        assert_eq!(response.completed_at, Some(1700000001));
    }

    #[tokio::test]
    async fn test_login_response() {
        let json = r#"{
            "access_token": "token123",
            "refresh_token": "refresh456",
            "expires_in": 3600,
            "user": {
                "id": 1,
                "username": "testuser",
                "email": "test@example.com"
            }
        }"#;

        let response: LoginResponse = serde_json::from_str(json).unwrap();
        assert_eq!(response.access_token, "token123");
        assert_eq!(response.refresh_token, "refresh456");
        assert_eq!(response.expires_in, 3600);
        assert_eq!(response.user.username, "testuser");
    }

    #[tokio::test]
    async fn test_click_captcha_response() {
        let json = r#"{
            "session_id": "click-session",
            "image_url": "http://example.com/click.png",
            "hint": "Click 1, 2, 3 in order",
            "hint_order": [1, 2, 3],
            "max_points": 3,
            "mode": "number",
            "allow_shuffle": true,
            "points": [[100, 100], [200, 150], [150, 200]]
        }"#;

        let response: ClickCaptchaResponse = serde_json::from_str(json).unwrap();
        assert_eq!(response.session_id, "click-session");
        assert_eq!(response.hint_order, vec![1, 2, 3]);
        assert_eq!(response.max_points, 3);
    }

    #[tokio::test]
    async fn test_jigsaw_captcha_response() {
        let json = r#"{
            "session_id": "jigsaw-session",
            "image_url": "http://example.com/jigsaw.png",
            "pieces": [
                {"index": 0, "original_x": 0, "original_y": 0, "current_x": 10, "current_y": 10, "width": 100, "height": 100, "rotation": 0},
                {"index": 1, "original_x": 100, "original_y": 0, "current_x": 120, "current_y": 5, "width": 100, "height": 100, "rotation": 0}
            ],
            "grid_size": 3,
            "piece_width": 100,
            "piece_height": 100,
            "image_width": 300,
            "image_height": 300
        }"#;

        let response: JigsawCaptchaResponse = serde_json::from_str(json).unwrap();
        assert_eq!(response.session_id, "jigsaw-session");
        assert_eq!(response.pieces.len(), 2);
        assert_eq!(response.grid_size, 3);
    }

    #[tokio::test]
    async fn test_error_types() {
        let api_error = CaptchaError::api_error("Test error", 400);
        assert!(!api_error.is_retryable());

        let network_error = CaptchaError::NetworkError(reqwest::Error::new(
            reqwest::error::Kind::Request,
            None,
        ));
        assert!(network_error.is_retryable());

        let validation_error = CaptchaError::validation_error("Invalid input");
        assert!(!validation_error.is_retryable());
    }

    #[tokio::test]
    async fn test_retry_config() {
        let config = utils::RetryConfig::default();
        assert_eq!(config.max_retries, 3);
        assert_eq!(config.base_delay_ms, 100);
        assert_eq!(config.max_delay_ms, 5000);

        let delay = utils::calculate_retry_delay(0, &config);
        assert_eq!(delay.as_millis(), 100);

        let delay = utils::calculate_retry_delay(2, &config);
        assert_eq!(delay.as_millis(), 400);
    }
}
