using System;
using System.Collections.Generic;
using System.Threading.Tasks;
using Hjtpx.Captcha.Client;
using Hjtpx.Captcha.Models;

namespace Hjtpx.Captcha.Examples
{
    public class AdvancedExamples
    {
        private readonly CaptchaClient _client;

        public AdvancedExamples()
        {
            _client = new CaptchaClient(
                "http://localhost:8080",
                "your-api-key",
                "your-secret-key"
            );
        }

        public async Task RunAllExamplesAsync()
        {
            Console.WriteLine("Starting advanced examples...\n");

            await SliderCaptchaWithTrajectoryExampleAsync();
            await ClickCaptchaExampleAsync();
            await GestureCaptchaExampleAsync();
            await RotationCaptchaExampleAsync();
            await JigsawCaptchaExampleAsync();
            await VoiceCaptchaExampleAsync();
            await UserAuthenticationExampleAsync();
            await EnvironmentDetectionExampleAsync();
            await ErrorHandlingExampleAsync();

            Console.WriteLine("\nAll advanced examples completed!");
        }

        private async Task SliderCaptchaWithTrajectoryExampleAsync()
        {
            Console.WriteLine("[Slider Captcha with Trajectory]");

            try
            {
                var slider = await _client.GetSliderCaptchaAsync(320, 160, 8);
                Console.WriteLine($"  Session ID: {slider.SessionId}");
                Console.WriteLine($"  Secret Y: {slider.SecretY}");

                var trajectory = GenerateTrajectory(slider.SecretY);
                Console.WriteLine($"  Generated {trajectory.Count} trajectory points");

                var result = await _client.VerifySliderCaptchaAsync(
                    slider.SessionId,
                    150,
                    slider.SecretY,
                    trajectory
                );

                Console.WriteLine($"  Verification: {(result.Success ? "Success" : "Failed")}");
                Console.WriteLine($"  Score: {result.Score}");
                if (result.RiskLevel != null)
                    Console.WriteLine($"  Risk Level: {result.RiskLevel}");
            }
            catch (Exception ex)
            {
                Console.WriteLine($"  Error: {ex.Message}");
            }

            Console.WriteLine();
        }

        private List<TrajectoryPoint> GenerateTrajectory(int secretY)
        {
            var trajectory = new List<TrajectoryPoint>();
            long baseTime = DateTimeOffset.UtcNow.ToUnixTimeMilliseconds();

            trajectory.Add(new TrajectoryPoint(0, secretY, baseTime - 1000));
            trajectory.Add(new TrajectoryPoint(30, secretY + 2, baseTime - 800));
            trajectory.Add(new TrajectoryPoint(60, secretY - 1, baseTime - 600));
            trajectory.Add(new TrajectoryPoint(100, secretY + 3, baseTime - 400));
            trajectory.Add(new TrajectoryPoint(140, secretY - 2, baseTime - 200));
            trajectory.Add(new TrajectoryPoint(180, secretY, baseTime));

            return trajectory;
        }

        private async Task ClickCaptchaExampleAsync()
        {
            Console.WriteLine("[Click Captcha]");

            try
            {
                var click = await _client.GetClickCaptchaAsync("number", true, 4);
                Console.WriteLine($"  Session ID: {click.SessionId}");
                Console.WriteLine($"  Hint: {click.Hint}");
                Console.WriteLine($"  Target Index: {click.TargetIndex}");
                Console.WriteLine($"  Icon Count: {click.IconPositions?.Count ?? 0}");

                var clicks = new List<ClickData>
                {
                    new ClickData { X = 100, Y = 100, Duration = 500 },
                    new ClickData { X = 200, Y = 150, Duration = 300 },
                    new ClickData { X = 300, Y = 200, Duration = 400 }
                };

                var result = await _client.VerifyClickCaptchaAsync(
                    click.SessionId,
                    clicks,
                    new List<int> { 0, 1, 2 }
                );

                Console.WriteLine($"  Verification: {(result.Success ? "Success" : "Failed")}");
                Console.WriteLine($"  Remaining Attempts: {result.RemainingAttempts}");
            }
            catch (Exception ex)
            {
                Console.WriteLine($"  Error: {ex.Message}");
            }

            Console.WriteLine();
        }

        private async Task GestureCaptchaExampleAsync()
        {
            Console.WriteLine("[Gesture Captcha]");

            try
            {
                var gesture = await _client.GetGestureCaptchaAsync();
                Console.WriteLine($"  Session ID: {gesture.SessionId}");
                Console.WriteLine($"  Pattern: {gesture.Pattern}");
                Console.WriteLine($"  Grid Size: {gesture.GridSize}");

                var pattern = new List<int> { 0, 1, 2, 4, 8 };
                var result = await _client.VerifyGestureCaptchaAsync(
                    gesture.SessionId,
                    pattern
                );

                Console.WriteLine($"  Verification: {(result.Success ? "Success" : "Failed")}");
            }
            catch (Exception ex)
            {
                Console.WriteLine($"  Error: {ex.Message}");
            }

            Console.WriteLine();
        }

        private async Task RotationCaptchaExampleAsync()
        {
            Console.WriteLine("[Rotation Captcha]");

            try
            {
                var rotation = await _client.GetRotationCaptchaAsync();
                Console.WriteLine($"  Challenge ID: {rotation.ChallengeId}");
                Console.WriteLine($"  Image URL: {rotation.ImageUrl}");

                var testAngles = new[] { 45, 90, 135, 180 };
                foreach (var angle in testAngles)
                {
                    var result = await _client.VerifyRotationCaptchaAsync(
                        rotation.ChallengeId,
                        angle
                    );
                    Console.WriteLine($"  Angle {angle}: {(result.Success ? "Success" : "Failed")}");
                }
            }
            catch (Exception ex)
            {
                Console.WriteLine($"  Error: {ex.Message}");
            }

            Console.WriteLine();
        }

        private async Task JigsawCaptchaExampleAsync()
        {
            Console.WriteLine("[Jigsaw Captcha]");

            try
            {
                var jigsaw = await _client.GetJigsawCaptchaAsync(300, 300, 3);
                Console.WriteLine($"  Session ID: {jigsaw.SessionId}");
                Console.WriteLine($"  Grid Size: {jigsaw.GridSize}");
                Console.WriteLine($"  Piece Count: {jigsaw.Pieces?.Count ?? 0}");

                var pieces = new List<JigsawPiece>();
                if (jigsaw.Pieces != null)
                {
                    foreach (var original in jigsaw.Pieces)
                    {
                        pieces.Add(new JigsawPiece
                        {
                            Index = original.Index,
                            OriginalX = original.OriginalX,
                            OriginalY = original.OriginalY,
                            CurrentX = original.OriginalX + 5,
                            CurrentY = original.OriginalY + 5,
                            Width = original.Width,
                            Height = original.Height,
                            Rotation = 0
                        });
                    }
                }

                var result = await _client.VerifyJigsawCaptchaAsync(jigsaw.SessionId, pieces);
                Console.WriteLine($"  Verification: {(result.Success ? "Success" : "Failed")}");
            }
            catch (Exception ex)
            {
                Console.WriteLine($"  Error: {ex.Message}");
            }

            Console.WriteLine();
        }

        private async Task VoiceCaptchaExampleAsync()
        {
            Console.WriteLine("[Voice Captcha]");

            try
            {
                var voice = await _client.GetVoiceCaptchaAsync("zh-CN");
                Console.WriteLine($"  Session ID: {voice.SessionId}");
                Console.WriteLine($"  Audio URL: {voice.AudioUrl}");
                Console.WriteLine($"  Text: {voice.Text}");

                var result = await _client.VerifyVoiceCaptchaAsync(
                    voice.SessionId,
                    voice.Text ?? "123456"
                );

                Console.WriteLine($"  Verification: {(result.Success ? "Success" : "Failed")}");
            }
            catch (Exception ex)
            {
                Console.WriteLine($"  Error: {ex.Message}");
            }

            Console.WriteLine();
        }

        private async Task UserAuthenticationExampleAsync()
        {
            Console.WriteLine("[User Authentication]");

            try
            {
                var login = await _client.LoginAsync("testuser", "password123");
                Console.WriteLine($"  Login Success: {login.AccessToken != null}");
                Console.WriteLine($"  User: {login.User?.Username}");
                Console.WriteLine($"  Expires In: {login.ExpiresIn} seconds");

                await _client.LogoutAsync();
                Console.WriteLine("  Logout completed");
            }
            catch (Exception ex)
            {
                Console.WriteLine($"  Error: {ex.Message}");
            }

            Console.WriteLine();
        }

        private async Task EnvironmentDetectionExampleAsync()
        {
            Console.WriteLine("[Environment Detection]");

            try
            {
                var script = await _client.GetDetectionScriptAsync();
                Console.WriteLine($"  Script Length: {script?.Length ?? 0} characters");

                var detectionData = new Dictionary<string, object>
                {
                    ["fingerprint"] = "test-fingerprint",
                    ["canvas_hash"] = "canvas-test",
                    ["webgl_vendor"] = "Test Vendor",
                    ["webgl_renderer"] = "Test Renderer",
                    ["timezone"] = "Asia/Shanghai",
                    ["language"] = "zh-CN"
                };

                var submitResult = await _client.SubmitDetectionAsync(detectionData);
                Console.WriteLine($"  Detection Result: {(bool)(submitResult?["success"] ?? false)}");
            }
            catch (Exception ex)
            {
                Console.WriteLine($"  Error: {ex.Message}");
            }

            Console.WriteLine();
        }

        private async Task ErrorHandlingExampleAsync()
        {
            Console.WriteLine("[Error Handling]");

            try
            {
                var invalidCaptcha = await _client.GetSliderCaptchaAsync(-1, -1, -1);
            }
            catch (Exceptions.ApiException ex)
            {
                Console.WriteLine($"  API Exception: Code={ex.Code}, Message={ex.Message}");
            }
            catch (Exceptions.NetworkException ex)
            {
                Console.WriteLine($"  Network Exception: {ex.Message}");
            }
            catch (Exceptions.CaptchaException ex)
            {
                Console.WriteLine($"  Captcha Exception: {ex.Message}");
            }
            catch (Exception ex)
            {
                Console.WriteLine($"  General Exception: {ex.Message}");
            }

            Console.WriteLine();
        }

        public void Dispose()
        {
            _client?.Dispose();
        }
    }
}
