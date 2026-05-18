namespace Hjtpx.Captcha.Exceptions;

public class ValidationException : CaptchaException
{
    public IReadOnlyDictionary<string, string[]>? Errors { get; }
    public string? FieldName { get; }
    public object? InvalidValue { get; }

    public ValidationException(string message) : base(message)
    {
    }

    public ValidationException(string message, string fieldName) : base(message)
    {
        FieldName = fieldName;
    }

    public ValidationException(string message, string fieldName, object invalidValue) : base(message)
    {
        FieldName = fieldName;
        InvalidValue = invalidValue;
    }

    public ValidationException(string message, IReadOnlyDictionary<string, string[]> errors) : base(message)
    {
        Errors = errors;
    }

    public ValidationException(string message, Exception innerException) : base(message, innerException)
    {
    }

    public static ValidationException Required(string fieldName) =>
        new($"The {fieldName} field is required", fieldName);

    public static ValidationException InvalidFormat(string fieldName, string expectedFormat) =>
        new($"The {fieldName} field has an invalid format. Expected: {expectedFormat}", fieldName);

    public static ValidationException OutOfRange(string fieldName, object value, object min, object max) =>
        new($"The {fieldName} field value {value} is out of range [{min}, {max}]", fieldName, value);

    public static ValidationException TooLong(string fieldName, int actualLength, int maxLength) =>
        new($"The {fieldName} field exceeds maximum length of {maxLength} characters (actual: {actualLength})", fieldName);

    public static ValidationException InvalidEnumValue<TEnum>(string fieldName, object value) where TEnum : struct, Enum =>
        new($"The {fieldName} field has invalid enum value: {value}", fieldName, value);

    public static ValidationException EmptySessionId() =>
        new("Session ID cannot be empty");

    public static ValidationException EmptyCaptchaType() =>
        new("Captcha type cannot be empty");

    public static ValidationException InvalidTrajectory(string reason) =>
        new($"Invalid trajectory data: {reason}");

    public static ValidationException InvalidPoints(string reason) =>
        new($"Invalid points data: {reason}");
}
