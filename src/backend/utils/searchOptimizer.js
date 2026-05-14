const MAX_QUERY_TIME = 5000;
const MAX_RESULTS = 1000;
const CACHE_TTL = 300;

const cache = new Map();

function optimizeSearchQuery(model, query, filters, options = {}) {
  const {
    useIndex = true,
    analyzeQuery = true,
    optimizeFilters = true
  } = options;

  let optimizedQuery = query;
  let warnings = [];

  if (analyzeQuery) {
    const analysis = analyzeQueryStructure(query, filters);
    warnings = analysis.warnings;
    optimizedQuery = analysis.optimizedQuery;
  }

  if (optimizeFilters) {
    optimizedQuery = optimizeFilterOrder(optimizedQuery, filters);
  }

  return {
    query: optimizedQuery,
    warnings,
    meta: {
      useIndex,
      analyzed: analyzeQuery,
      optimized: optimizeFilters
    }
  };
}

function analyzeQueryStructure(query, filters) {
  const warnings = [];
  let optimizedQuery = query;

  if (query && query.length > 100) {
    warnings.push('Long search query may impact performance');
  }

  if (Object.keys(filters).length > 10) {
    warnings.push('Too many filters may impact performance');
  }

  for (const [field, filter] of Object.entries(filters)) {
    if (filter.value && typeof filter.value === 'string' && filter.value.length > 50) {
      warnings.push(`Long value for filter "${field}" may impact performance`);
    }
  }

  return { optimizedQuery, warnings };
}

function optimizeFilterOrder(query, filters) {
  const filterEntries = Object.entries(filters);
  const sortedFilters = filterEntries.sort((a, b) => {
    const aComplexity = getFilterComplexity(a[1]);
    const bComplexity = getFilterComplexity(b[1]);
    return aComplexity - bComplexity;
  });

  const optimized = {};
  for (const [key, value] of sortedFilters) {
    optimized[key] = value;
  }

  return optimized;
}

function getFilterComplexity(filter) {
  if (!filter || !filter.operator) return 5;

  const complexityMap = {
    isnull: 1,
    notnull: 1,
    eq: 1,
    ne: 2,
    like: 3,
    in: 3,
    nin: 4,
    gt: 2,
    gte: 2,
    lt: 2,
    lte: 2,
    between: 4
  };

  return complexityMap[filter.operator] || 5;
}

function addToCache(key, data, ttl = CACHE_TTL) {
  const expiresAt = Date.now() + (ttl * 1000);
  cache.set(key, { data, expiresAt });

  setTimeout(() => {
    if (cache.has(key)) {
      const cached = cache.get(key);
      if (Date.now() > cached.expiresAt) {
        cache.delete(key);
      }
    }
  }, ttl * 1000);
}

function getFromCache(key) {
  if (!cache.has(key)) {
    return null;
  }

  const cached = cache.get(key);
  if (Date.now() > cached.expiresAt) {
    cache.delete(key);
    return null;
  }

  return cached.data;
}

function clearCache(pattern = null) {
  if (!pattern) {
    cache.clear();
    return { cleared: true, count: 0 };
  }

  let count = 0;
  for (const key of cache.keys()) {
    if (key.includes(pattern)) {
      cache.delete(key);
      count++;
    }
  }

  return { cleared: true, count };
}

function getCacheStats() {
  return {
    size: cache.size,
    maxSize: MAX_RESULTS,
    ttl: CACHE_TTL,
    entries: Array.from(cache.keys())
  };
}

function checkQueryPerformance(query, params, options = {}) {
  const { timeout = MAX_QUERY_TIME } = options;

  const startTime = Date.now();
  let isSlow = false;
  let estimatedTime = null;

  const complexityScore = calculateComplexityScore(query, params);

  if (complexityScore > 100) {
    isSlow = true;
    estimatedTime = complexityScore * 10;
  }

  return {
    isSlow,
    estimatedTime,
    complexityScore,
    timeout,
    recommendations: getRecommendations(complexityScore, query)
  };
}

function calculateComplexityScore(query, params) {
  let score = 0;

  score += (query.match(/ILIKE/g) || []).length * 20;
  score += (query.match(/JOIN/g) || []).length * 30;
  score += (query.match(/ORDER BY/g) || []).length * 10;
  score += (query.match(/GROUP BY/g) || []).length * 15;
  score += (query.match(/HAVING/g) || []).length * 25;

  if (params && params.length > 10) {
    score += (params.length - 10) * 5;
  }

  return score;
}

function getRecommendations(complexityScore, query) {
  const recommendations = [];

  if (complexityScore > 100) {
    recommendations.push('Query is complex, consider adding indexes');
  }

  if (query.includes('ILIKE')) {
    recommendations.push('Consider using indexed columns for searches');
  }

  if (query.includes('JOIN')) {
    recommendations.push('Review JOIN performance, ensure indexed columns');
  }

  if (query.includes('ORDER BY')) {
    recommendations.push('Sorting on indexed columns is recommended');
  }

  return recommendations;
}

function buildOptimizedQuery(model, options = {}) {
  const {
    requiredFields = ['*'],
    defaultFilters = {},
    sortField = 'created_at',
    sortOrder = 'DESC'
  } = options;

  let query = `SELECT ${requiredFields.join(', ')} FROM ${model}`;
  const conditions = [];
  const params = [];

  for (const [field, filter] of Object.entries(defaultFilters)) {
    if (filter.value !== undefined && filter.value !== null) {
      params.push(filter.value);
      conditions.push(`${field} = $${params.length}`);
    }
  }

  if (conditions.length > 0) {
    query += ` WHERE ${conditions.join(' AND ')}`;
  }

  query += ` ORDER BY ${sortField} ${sortOrder}`;

  return { query, params };
}

module.exports = {
  optimizeSearchQuery,
  analyzeQueryStructure,
  optimizeFilterOrder,
  addToCache,
  getFromCache,
  clearCache,
  getCacheStats,
  checkQueryPerformance,
  calculateComplexityScore,
  getRecommendations,
  buildOptimizedQuery,
  MAX_QUERY_TIME,
  MAX_RESULTS,
  CACHE_TTL
};
