package main

import (
	"fmt"
	"time"

	captchago "github.com/hjtpx/hjtpx/sdk/go"
)

type LoginService struct {
	captchaClient *captchago.CaptchaClient
}

func NewLoginService(baseURL, appID, appSecret string) *LoginService {
	client := captchago.NewCaptchaClient(appID, appSecret, &captchago.Config{
		BaseURL:     baseURL,
		MaxRetries:  3,
		HTTPTimeout: 10 * time.Second,
	})
	return &LoginService{captchaClient: client}
}

func (s *LoginService) Close() error {
	return s.captchaClient.Close()
}

type LoginRequest struct {
	Username string
	Password string
}

type LoginResult struct {
	Success      bool
	Token        string
	ErrorMessage string
}

func (s *LoginService) LoginWithCaptcha(req *LoginRequest) (*LoginResult, error) {
	fmt.Printf("Processing login for user: %s\n", req.Username)

	sliderResp, err := s.captchaClient.GenerateSliderCaptcha()
	if err != nil {
		return &LoginResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to generate captcha: %v", err),
		}, nil
	}
	fmt.Printf("Captcha generated: %s\n", sliderResp.ChallengeID)

	simulatedUserX := 150
	verifyResp, err := s.captchaClient.VerifySliderCaptcha(
		sliderResp.ChallengeID,
		fmt.Sprintf("%d", simulatedUserX),
	)
	if err != nil {
		return &LoginResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to verify captcha: %v", err),
		}, nil
	}

	if !verifyResp.Success {
		return &LoginResult{
			Success:      false,
			ErrorMessage: "Captcha verification failed",
		}, nil
	}
	fmt.Printf("Captcha verified successfully. Score: %.2f\n", verifyResp.Score)

	token := "mock-jwt-token-" + fmt.Sprintf("%d", time.Now().Unix())
	fmt.Printf("User %s logged in successfully. Token: %s\n", req.Username, token)

	return &LoginResult{
		Success: true,
		Token:   token,
	}, nil
}

type RegisterService struct {
	captchaClient *captchago.CaptchaClient
}

func NewRegisterService(baseURL, appID, appSecret string) *RegisterService {
	client := captchago.NewCaptchaClient(appID, appSecret, &captchago.Config{
		BaseURL:     baseURL,
		MaxRetries:  3,
		HTTPTimeout: 10 * time.Second,
	})
	return &RegisterService{captchaClient: client}
}

func (s *RegisterService) Close() error {
	return s.captchaClient.Close()
}

type RegisterRequest struct {
	Username string
	Email    string
	Password string
}

type RegisterResult struct {
	Success      bool
	UserID       string
	ErrorMessage string
}

func (s *RegisterService) RegisterWithCaptcha(req *RegisterRequest) (*RegisterResult, error) {
	fmt.Printf("Processing registration for user: %s\n", req.Username)

	clickResp, err := s.captchaClient.GenerateClickCaptcha()
	if err != nil {
		return &RegisterResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to generate captcha: %v", err),
		}, nil
	}
	fmt.Printf("Click captcha generated: %s\n", clickResp.ChallengeID)
	fmt.Printf("Target index: %d\n", clickResp.TargetIndex)

	clicks := []captchago.ClickData{
		{
			X:        clickResp.IconPositions[clickResp.TargetIndex][0],
			Y:        clickResp.IconPositions[clickResp.TargetIndex][1],
			Duration: 500,
		},
	}

	verifyResp, err := s.captchaClient.VerifyClickCaptcha(clickResp.ChallengeID, clicks)
	if err != nil {
		return &RegisterResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to verify captcha: %v", err),
		}, nil
	}

	if !verifyResp.Success {
		return &RegisterResult{
			Success:      false,
			ErrorMessage: "Captcha verification failed",
		}, nil
	}
	fmt.Printf("Click captcha verified successfully. Score: %.2f\n", verifyResp.Score)

	userID := fmt.Sprintf("user-%d", time.Now().UnixNano())
	fmt.Printf("User %s registered successfully. UserID: %s\n", req.Username, userID)

	return &RegisterResult{
		Success: true,
		UserID:  userID,
	}, nil
}

type CommentService struct {
	captchaClient *captchago.CaptchaClient
}

func NewCommentService(baseURL, appID, appSecret string) *CommentService {
	client := captchago.NewCaptchaClient(appID, appSecret, &captchago.Config{
		BaseURL:     baseURL,
		MaxRetries:  3,
		HTTPTimeout: 10 * time.Second,
	})
	return &CommentService{captchaClient: client}
}

func (s *CommentService) Close() error {
	return s.captchaClient.Close()
}

type CommentRequest struct {
	ArticleID  string
	UserID    string
	Content   string
	SessionID string
	X         int
	Y         int
}

type CommentResult struct {
	Success      bool
	CommentID    string
	ErrorMessage string
}

func (s *CommentService) PostCommentWithCaptcha(req *CommentRequest) (*CommentResult, error) {
	fmt.Printf("Processing comment from user: %s on article: %s\n", req.UserID, req.ArticleID)

	if req.SessionID == "" {
		sliderResp, err := s.captchaClient.GenerateSliderCaptcha()
		if err != nil {
			return &CommentResult{
				Success:      false,
				ErrorMessage: fmt.Sprintf("Failed to generate captcha: %v", err),
			}, nil
		}
		req.SessionID = sliderResp.ChallengeID
		req.X = 120
		fmt.Printf("New captcha session: %s\n", req.SessionID)
	}

	verifyResp, err := s.captchaClient.VerifySliderCaptcha(req.SessionID, fmt.Sprintf("%d", req.X))
	if err != nil {
		return &CommentResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to verify captcha: %v", err),
		}, nil
	}

	if !verifyResp.Success {
		return &CommentResult{
			Success:      false,
			ErrorMessage: "Captcha verification failed",
		}, nil
	}

	commentID := fmt.Sprintf("comment-%d", time.Now().UnixNano())
	fmt.Printf("Comment posted successfully: %s\n", commentID)

	return &CommentResult{
		Success:   true,
		CommentID: commentID,
	}, nil
}

type ECommerceCheckoutService struct {
	captchaClient *captchago.CaptchaClient
}

func NewECommerceCheckoutService(baseURL, appID, appSecret string) *ECommerceCheckoutService {
	client := captchago.NewCaptchaClient(appID, appSecret, &captchago.Config{
		BaseURL:     baseURL,
		MaxRetries:  3,
		HTTPTimeout: 10 * time.Second,
	})
	return &ECommerceCheckoutService{captchaClient: client}
}

func (s *ECommerceCheckoutService) Close() error {
	return s.captchaClient.Close()
}

type CheckoutRequest struct {
	UserID      string
	OrderID     string
	TotalAmount float64
}

type CheckoutResult struct {
	Success      bool
	OrderStatus  string
	ErrorMessage string
}

func (s *ECommerceCheckoutService) CheckoutWithCaptcha(req *CheckoutRequest) (*CheckoutResult, error) {
	fmt.Printf("Processing checkout for order: %s, amount: $%.2f\n", req.OrderID, req.TotalAmount)

	if req.TotalAmount > 100 {
		imageResp, err := s.captchaClient.GenerateImageCaptcha(&captchago.ImageCaptchaRequest{
			Type:  captchago.CaptchaTypeMixed,
			Count: 4,
		})
		if err != nil {
			return &CheckoutResult{
				Success:      false,
				OrderStatus:  "failed",
				ErrorMessage: fmt.Sprintf("Failed to generate captcha: %v", err),
			}, nil
		}
		fmt.Printf("Image captcha generated: %s\n", imageResp.ChallengeID)

		verifyResp, err := s.captchaClient.VerifyImageCaptcha(imageResp.ChallengeID, "test")
		if err != nil {
			return &CheckoutResult{
				Success:      false,
				OrderStatus:  "failed",
				ErrorMessage: fmt.Sprintf("Failed to verify captcha: %v", err),
			}, nil
		}

		if !verifyResp.Success {
			return &CheckoutResult{
				Success:      false,
				OrderStatus:  "failed",
				ErrorMessage: "Captcha verification failed",
			}, nil
		}
		fmt.Printf("Image captcha verified successfully\n")
	} else {
		sliderResp, err := s.captchaClient.GenerateSliderCaptcha()
		if err != nil {
			return &CheckoutResult{
				Success:      false,
				OrderStatus:  "failed",
				ErrorMessage: fmt.Sprintf("Failed to generate captcha: %v", err),
			}, nil
		}

		verifyResp, err := s.captchaClient.VerifySliderCaptcha(sliderResp.ChallengeID, "100")
		if err != nil {
			return &CheckoutResult{
				Success:      false,
				OrderStatus:  "failed",
				ErrorMessage: fmt.Sprintf("Failed to verify captcha: %v", err),
			}, nil
		}

		if !verifyResp.Success {
			return &CheckoutResult{
				Success:      false,
				OrderStatus:  "failed",
				ErrorMessage: "Captcha verification failed",
			}, nil
		}
		fmt.Printf("Slider captcha verified successfully\n")
	}

	fmt.Printf("Order %s completed successfully\n", req.OrderID)
	return &CheckoutResult{
		Success:     true,
		OrderStatus: "completed",
	}, nil
}

func main() {
	fmt.Println("======================================")
	fmt.Println("  HJT Captcha Integration Examples")
	fmt.Println("======================================")
	fmt.Println()

	baseURL := "http://localhost:8080"
	appID := "your-app-id"
	appSecret := "your-app-secret"

	fmt.Println("[1] Login Service Example")
	fmt.Println("-----------------------------------")
	loginService := NewLoginService(baseURL, appID, appSecret)
	defer loginService.Close()

	loginResult, err := loginService.LoginWithCaptcha(&LoginRequest{
		Username: "testuser",
		Password: "password123",
	})
	if err != nil {
		fmt.Printf("Login error: %v\n", err)
	} else {
		fmt.Printf("Login result: Success=%v, Token=%s, Error=%s\n",
			loginResult.Success, loginResult.Token, loginResult.ErrorMessage)
	}
	fmt.Println()

	fmt.Println("[2] Register Service Example")
	fmt.Println("-----------------------------------")
	registerService := NewRegisterService(baseURL, appID, appSecret)
	defer registerService.Close()

	registerResult, err := registerService.RegisterWithCaptcha(&RegisterRequest{
		Username: "newuser",
		Email:    "newuser@example.com",
		Password: "securepassword",
	})
	if err != nil {
		fmt.Printf("Register error: %v\n", err)
	} else {
		fmt.Printf("Register result: Success=%v, UserID=%s, Error=%s\n",
			registerResult.Success, registerResult.UserID, registerResult.ErrorMessage)
	}
	fmt.Println()

	fmt.Println("[3] Comment Service Example")
	fmt.Println("-----------------------------------")
	commentService := NewCommentService(baseURL, appID, appSecret)
	defer commentService.Close()

	commentResult, err := commentService.PostCommentWithCaptcha(&CommentRequest{
		ArticleID: "article-123",
		UserID:    "user-456",
		Content:   "This is a great article!",
	})
	if err != nil {
		fmt.Printf("Comment error: %v\n", err)
	} else {
		fmt.Printf("Comment result: Success=%v, CommentID=%s, Error=%s\n",
			commentResult.Success, commentResult.CommentID, commentResult.ErrorMessage)
	}
	fmt.Println()

	fmt.Println("[4] E-Commerce Checkout Example (High Value)")
	fmt.Println("-----------------------------------")
	checkoutService := NewECommerceCheckoutService(baseURL, appID, appSecret)
	defer checkoutService.Close()

	checkoutResult, err := checkoutService.CheckoutWithCaptcha(&CheckoutRequest{
		UserID:      "user-789",
		OrderID:    "order-001",
		TotalAmount: 150.00,
	})
	if err != nil {
		fmt.Printf("Checkout error: %v\n", err)
	} else {
		fmt.Printf("Checkout result: Success=%v, Status=%s, Error=%s\n",
			checkoutResult.Success, checkoutResult.OrderStatus, checkoutResult.ErrorMessage)
	}
	fmt.Println()

	fmt.Println("[5] E-Commerce Checkout Example (Low Value)")
	fmt.Println("-----------------------------------")
	checkoutResult2, err := checkoutService.CheckoutWithCaptcha(&CheckoutRequest{
		UserID:      "user-789",
		OrderID:    "order-002",
		TotalAmount: 50.00,
	})
	if err != nil {
		fmt.Printf("Checkout error: %v\n", err)
	} else {
		fmt.Printf("Checkout result: Success=%v, Status=%s, Error=%s\n",
			checkoutResult2.Success, checkoutResult2.OrderStatus, checkoutResult2.ErrorMessage)
	}
	fmt.Println()

	fmt.Println("======================================")
	fmt.Println("  Integration examples completed!")
	fmt.Println("======================================")
}
