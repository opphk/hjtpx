const logger = (req, res, next) => {
  const startTime = Date.now();
  const requestId = generateRequestId();

  req.requestId = requestId;

  const logRequest = () => {
    const duration = Date.now() - startTime;
    const timestamp = new Date().toISOString();

    const logEntry = {
      requestId,
      timestamp,
      method: req.method,
      url: req.originalUrl || req.url,
      path: req.path,
      query: req.query,
      params: req.params,
      ip: req.ip || req.connection?.remoteAddress,
      userAgent: req.get('user-agent') || 'Unknown',
      userId: req.user?.id || null,
      contentType: req.get('content-type'),
      contentLength: req.get('content-length'),
      duration: `${duration}ms`,
      statusCode: res.statusCode
    };

    if (process.env.NODE_ENV === 'development') {
      console.log('\n[REQUEST]', JSON.stringify(logEntry, null, 2));
    } else {
      console.log(JSON.stringify(logEntry));
    }
  };

  res.on('finish', logRequest);
  res.on('close', logRequest);

  if (process.env.NODE_ENV === 'development') {
    console.log(`\n[${requestId}] --> ${req.method} ${req.path}`);
  }

  next();
};

const generateRequestId = () => {
  return `req_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
};

const logError = (error, req = null, context = {}) => {
  const timestamp = new Date().toISOString();
  const requestId = req?.requestId || 'unknown';

  const errorLog = {
    requestId,
    timestamp,
    level: 'error',
    message: error.message || 'Unknown error',
    stack: process.env.NODE_ENV === 'development' ? error.stack : undefined,
    name: error.name,
    code: error.code,
    context,
    url: req?.originalUrl || req?.url,
    method: req?.method,
    ip: req?.ip || req?.connection?.remoteAddress,
    userAgent: req?.get('user-agent')
  };

  console.error(JSON.stringify(errorLog));
};

const logWarning = (message, context = {}) => {
  const timestamp = new Date().toISOString();

  const warningLog = {
    timestamp,
    level: 'warning',
    message,
    ...context
  };

  console.warn(JSON.stringify(warningLog));
};

const logInfo = (message, context = {}) => {
  const timestamp = new Date().toISOString();

  const infoLog = {
    timestamp,
    level: 'info',
    message,
    ...context
  };

  console.log(JSON.stringify(infoLog));
};

module.exports = {
  logger,
  logError,
  logWarning,
  logInfo
};
