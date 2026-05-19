using System;
using System.Collections.Generic;
using System.Net.Http;
using System.Threading.Tasks;
using Microsoft.AspNetCore.Builder;
using Microsoft.AspNetCore.Hosting;
using Microsoft.AspNetCore.Http;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.Logging;
using Hjtpx.Captcha.Client;
using Hjtpx.Captcha.Models;

var builder = WebApplication.CreateBuilder(args);

builder.Services.AddControllers();
builder.Services.AddEndpointsApiExplorer();
builder.Services.AddSwaggerGen();

builder.Services.AddSingleton<CaptchaClient>(sp =>
{
    var config = new CaptchaClientConfig("http://localhost:8080")
    {
        ApiKey = "your-api-key",
        SecretKey = "your-secret-key"
    };

    config.ConnectionPoolConfig.MaxConnections = 200;
    config.ConnectionPoolConfig.MaxConnectionsPerRoute = 50;

    config.RetryConfig.MaxRetries = 3;
    config.RetryConfig.InitialDelayMs = 100;

    return new CaptchaClient(config);
});

var app = builder.Build();

if (app.Environment.IsDevelopment())
{
    app.UseSwagger();
    app.UseSwaggerUI();
}

app.UseHttpsRedirection();
app.UseAuthorization();
app.MapControllers();

app.MapPost("/captcha/slider", async (CaptchaClient client, int width = 320, int height = 160) =>
{
    try
    {
        var captcha = await client.GetSliderCaptchaAsync(width, height);
        return Results.Ok(new
        {
            success = true,
            sessionId = captcha.SessionId,
            imageUrl = captcha.ImageUrl,
            puzzleUrl = captcha.PuzzleUrl,
            secretY = captcha.SecretY
        });
    }
    catch (Exception ex)
    {
        return Results.BadRequest(new { success = false, error = ex.Message });
    }
});

app.MapPost("/captcha/slider/verify", async (CaptchaClient client, VerifyRequest request) =>
{
    try
    {
        var result = await client.VerifySliderCaptchaAsync(
            request.SessionId,
            request.X,
            request.Y,
            request.Trajectory
        );

        return Results.Ok(new
        {
            success = result.Success,
            message = result.Message,
            riskScore = result.RiskScore,
            remainingAttempts = result.RemainingAttempts
        });
    }
    catch (Exception ex)
    {
        return Results.BadRequest(new { success = false, error = ex.Message });
    }
});

app.MapPost("/captcha/click", async (CaptchaClient client, string mode = "number", int maxPoints = 3) =>
{
    try
    {
        var captcha = await client.GetClickCaptchaAsync(mode, maxPoints);
        return Results.Ok(new
        {
            success = true,
            sessionId = captcha.SessionId,
            imageUrl = captcha.ImageUrl,
            hint = captcha.Hint,
            hintOrder = captcha.HintOrder
        });
    }
    catch (Exception ex)
    {
        return Results.BadRequest(new { success = false, error = ex.Message });
    }
});

app.MapPost("/captcha/click/verify", async (CaptchaClient client, VerifyClickRequest request) =>
{
    try
    {
        var result = await client.VerifyClickCaptchaAsync(
            request.SessionId,
            request.Points,
            request.ClickSequence
        );

        return Results.Ok(new
        {
            success = result.Success,
            message = result.Message,
            riskScore = result.RiskScore,
            remainingAttempts = result.RemainingAttempts
        });
    }
    catch (Exception ex)
    {
        return Results.BadRequest(new { success = false, error = ex.Message });
    }
});

app.MapGet("/health", () => Results.Ok(new { status = "healthy" }));

app.Run();

public class VerifyRequest
{
    public string SessionId { get; set; }
    public int X { get; set; }
    public int? Y { get; set; }
    public List<TrajectoryPoint>? Trajectory { get; set; }
}

public class VerifyClickRequest
{
    public string SessionId { get; set; }
    public List<List<int>> Points { get; set; }
    public List<int>? ClickSequence { get; set; }
}
