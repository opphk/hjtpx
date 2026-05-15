const responseCache = new Map();
const CACHE_MAX_SIZE = 100;
const DEFAULT_TTL = 60000;

function getCacheKey(req) {
  return `${req.method}:${req.originalUrl}:${JSON.stringify(req.query)}`;
}

function generateETag(data) {
  const hash = require('crypto').createHash('md5').update(JSON.stringify(data)).digest('hex');
  return `"${hash}"`;
}

function shouldCompress(req, data) {
  const acceptEncoding = req.headers['accept-encoding'] || '';
  const contentLength = JSON.stringify(data).length;

  if (acceptEncoding.includes('gzip') && contentLength > 1024) {
    return true;
  }

  return false;
}

function compressResponse(data) {
  const zlib = require('zlib');
  const gzip = zlib.createGzip({
    level: 6,
    memLevel: 8
  });

  return new Promise((resolve, reject) => {
    const chunks = [];
    gzip.on('data', chunk => chunks.push(chunk));
    gzip.on('end', () => resolve(Buffer.concat(chunks)));
    gzip.on('error', reject);
    gzip.write(JSON.stringify(data));
    gzip.end();
  });
}

function cacheResponse(req, res, next) {
  if (req.method !== 'GET') {
    return next();
  }

  const cacheKey = getCacheKey(req);
  const cached = responseCache.get(cacheKey);

  if (cached && Date.now() - cached.timestamp < (req.cacheTTL || DEFAULT_TTL)) {
    const clientETag = req.headers['if-none-match'];

    if (clientETag && clientETag === cached.etag) {
      return res.status(304).end();
    }

    res.set({
      'X-Cache': 'HIT',
      ETag: cached.etag,
      'Cache-Control': 'public, max-age=60',
      'Last-Modified': new Date(cached.timestamp).toUTCString()
    });

    if (shouldCompress(req, cached.data)) {
      compressResponse(cached.data).then(compressed => {
        res.set('Content-Encoding', 'gzip');
        res.json(cached.data);
      });
    } else {
      return res.json(cached.data);
    }
  } else {
    res.set('X-Cache', 'MISS');
    next();
  }
}

function setResponseCache(req, res, next) {
  if (req.method !== 'GET' || res.headersSent) {
    return next();
  }

  const cacheKey = getCacheKey(req);

  const originalJson = res.json.bind(res);
  res.json = function (data) {
    if (data && data.success !== false) {
      const etag = generateETag(data);

      if (responseCache.size >= CACHE_MAX_SIZE) {
        const firstKey = responseCache.keys().next().value;
        responseCache.delete(firstKey);
      }

      responseCache.set(cacheKey, {
        data,
        etag,
        timestamp: Date.now()
      });
    }

    return originalJson(data);
  };

  next();
}

function invalidateCache(pattern) {
  const regex = new RegExp(pattern.replace(/\*/g, '.*'));
  let count = 0;

  for (const key of responseCache.keys()) {
    if (regex.test(key)) {
      responseCache.delete(key);
      count++;
    }
  }

  return count;
}

function clearCache() {
  responseCache.clear();
}

function getCacheStats() {
  return {
    size: responseCache.size,
    maxSize: CACHE_MAX_SIZE,
    keys: Array.from(responseCache.keys())
  };
}

function responseTimeTracker(req, res, next) {
  const start = process.hrtime.bigint();

  res.on('finish', () => {
    const end = process.hrtime.bigint();
    const duration = Number(end - start) / 1e6;

    res.set('X-Response-Time', `${duration.toFixed(2)}ms`);

    if (duration > 500) {
      console.warn(`Slow response (${duration.toFixed(2)}ms): ${req.method} ${req.path}`);
    }
  });

  next();
}

function apiVersionMiddleware(req, res, next) {
  req.apiVersion = req.headers['api-version'] || 'v1';
  res.set('API-Version', req.apiVersion);
  res.set('X-API-Version', req.apiVersion);
  next();
}

function paginationDefaults(req, res, next) {
  const defaults = {
    page: 1,
    pageSize: 20,
    maxPageSize: 100
  };

  req.pagination = {
    page: Math.max(1, parseInt(req.query.page) || defaults.page),
    pageSize: Math.min(
      defaults.maxPageSize,
      Math.max(1, parseInt(req.query.pageSize) || defaults.pageSize)
    )
  };

  req.pagination.offset = (req.pagination.page - 1) * req.pagination.pageSize;

  next();
}

function addPaginationHeaders(res, pagination) {
  res.set({
    'X-Pagination-Page': pagination.page,
    'X-Pagination-PageSize': pagination.pageSize,
    'X-Pagination-Total': pagination.total,
    'X-Pagination-TotalPages': pagination.totalPages
  });
}

function conditionalGet(req, res, next) {
  if (req.method !== 'GET') {
    return next();
  }

  const cacheKey = getCacheKey(req);
  const cached = responseCache.get(cacheKey);

  if (cached) {
    const ifNoneMatch = req.headers['if-none-match'];
    if (ifNoneMatch && ifNoneMatch === cached.etag) {
      return res.status(304).send();
    }

    const ifModifiedSince = req.headers['if-modified-since'];
    if (ifModifiedSince) {
      const cachedTime = new Date(cached.timestamp).getTime();
      const requestTime = new Date(ifModifiedSince).getTime();
      if (cachedTime <= requestTime) {
        return res.status(304).send();
      }
    }
  }

  next();
}

module.exports = {
  cacheResponse,
  setResponseCache,
  invalidateCache,
  clearCache,
  getCacheStats,
  responseTimeTracker,
  apiVersionMiddleware,
  paginationDefaults,
  addPaginationHeaders,
  conditionalGet,
  generateETag,
  compressResponse
};
