const { AppError, ValidationError, AuthenticationError } = require('../utils/appErrors');
const { ErrorLogService } = require('../utils/errorLogger');
const ErrorCode = require('../utils/errorCodes');

function errorHandler(err, req, res, next) {
  let error = err;

  if (!(err instanceof AppError)) {
    if (err.name === 'ValidationError' && err.errors) {
      error = new ValidationError('Validation failed', err.errors);
    } else if (err.name === 'JsonWebTokenError') {
      error = new AuthenticationError('AUTH_003', 'Invalid token');
    } else if (err.name === 'TokenExpiredError') {
      error = new AuthenticationError('AUTH_002', 'Token expired');
    } else {
      error = new AppError(
        'SRV_001',
        process.env.NODE_ENV === 'production' ? 'Internal server error' : err.message,
        500
      );
    }
  }

  const logContext = {
    path: req.path,
    method: req.method,
    ip: req.ip,
    userId: req.user?.id,
    requestId: req.requestId
  };

  ErrorLogService.log(error, logContext);

  const statusCode = error.statusCode || 500;
  const response = error.toResponse();

  if (statusCode === 500 && process.env.NODE_ENV === 'production') {
    response.error.message = 'Internal server error';
    delete response.error.details;
  }

  res.status(statusCode).json(response);
}

function notFoundHandler(req, res, next) {
  const error = new AppError('DB_004', `Route ${req.originalUrl} not found`, 404);
  res.status(404).json(error.toResponse());
}

function asyncHandler(fn) {
  return (req, res, next) => {
    Promise.resolve(fn(req, res, next)).catch(next);
  };
}

module.exports = {
  errorHandler,
  notFoundHandler,
  asyncHandler
};
