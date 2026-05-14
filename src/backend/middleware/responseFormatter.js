const responseFormatter = (req, res, next) => {
  res.success = (data, message = 'Success', statusCode = 200) => {
    return res.status(statusCode).json({
      success: true,
      data,
      message,
      timestamp: new Date().toISOString()
    });
  };

  res.error = (message, statusCode = 500, code = 'INTERNAL_ERROR', details = null) => {
    const errorResponse = {
      success: false,
      error: {
        code,
        message
      },
      timestamp: new Date().toISOString()
    };

    if (details) {
      errorResponse.error.details = details;
    }

    return res.status(statusCode).json(errorResponse);
  };

  res.created = (data, message = 'Resource created successfully') => {
    return res.status(201).json({
      success: true,
      data,
      message,
      timestamp: new Date().toISOString()
    });
  };

  res.noContent = () => {
    return res.status(204).send();
  };

  res.notFound = (message = 'Resource not found') => {
    return res.status(404).json({
      success: false,
      error: {
        code: 'NOT_FOUND',
        message
      },
      timestamp: new Date().toISOString()
    });
  };

  res.unauthorized = (message = 'Unauthorized access') => {
    return res.status(401).json({
      success: false,
      error: {
        code: 'UNAUTHORIZED',
        message
      },
      timestamp: new Date().toISOString()
    });
  };

  res.forbidden = (message = 'Access forbidden') => {
    return res.status(403).json({
      success: false,
      error: {
        code: 'FORBIDDEN',
        message
      },
      timestamp: new Date().toISOString()
    });
  };

  res.badRequest = (message = 'Bad request', details = null) => {
    const response = {
      success: false,
      error: {
        code: 'BAD_REQUEST',
        message
      },
      timestamp: new Date().toISOString()
    };

    if (details) {
      response.error.details = details;
    }

    return res.status(400).json(response);
  };

  res.tooManyRequests = (message = 'Too many requests', retryAfter = 60) => {
    return res.status(429).json({
      success: false,
      error: {
        code: 'TOO_MANY_REQUESTS',
        message,
        retryAfter
      },
      timestamp: new Date().toISOString()
    });
  };

  next();
};

module.exports = responseFormatter;
