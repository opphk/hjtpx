const { generateRequestId } = require('../utils/logger');

const REQUEST_ID_HEADER = 'X-Request-ID';

const requestIdMiddleware = (req, res, next) => {
  req.headers = req.headers || {};
  
  const incomingRequestId = req.headers[REQUEST_ID_HEADER.toLowerCase()];

  const requestId = incomingRequestId || generateRequestId();

  req.requestId = requestId;
  res.requestId = requestId;

  if (typeof res.setHeader === 'function') {
    res.setHeader(REQUEST_ID_HEADER, requestId);
  }

  if (req.headers['x-trace-id']) {
    req.traceId = req.headers['x-trace-id'];
  } else {
    req.traceId = requestId;
  }

  next();
};

module.exports = requestIdMiddleware;
