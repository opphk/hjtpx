class QueryCache {
  constructor(maxSize = 100, ttl = 300000) {
    this.cache = new Map();
    this.maxSize = maxSize;
    this.ttl = ttl;
  }

  generateKey(query, params) {
    return JSON.stringify({ query, params });
  }

  get(query, params) {
    const key = this.generateKey(query, params);
    const entry = this.cache.get(key);

    if (!entry) {
      return null;
    }

    if (Date.now() - entry.timestamp > this.ttl) {
      this.cache.delete(key);
      return null;
    }

    return entry.result;
  }

  set(query, params, result) {
    const key = this.generateKey(query, params);

    if (this.cache.size >= this.maxSize) {
      const firstKey = this.cache.keys().next().value;
      this.cache.delete(firstKey);
    }

    this.cache.set(key, {
      result,
      timestamp: Date.now(),
    });
  }

  clear() {
    this.cache.clear();
  }

  getSize() {
    return this.cache.size;
  }
}

class QueryAnalyzer {
  analyze(query, executionTime) {
    const analysis = {
      query,
      executionTime,
      suggestions: [],
      complexity: this.estimateComplexity(query),
    };

    if (executionTime > 1000) {
      analysis.suggestions.push('Query execution time is over 1 second. Consider optimization.');
    }

    if (query.toUpperCase().includes('SELECT *')) {
      analysis.suggestions.push('Avoid SELECT * for better performance.');
    }

    if (query.toUpperCase().includes('JOIN') && query.toUpperCase().includes('SELECT')) {
      analysis.suggestions.push('Ensure JOINs have proper indexes.');
    }

    if (!query.toUpperCase().includes('LIMIT') && query.toUpperCase().includes('SELECT')) {
      analysis.suggestions.push('Consider adding LIMIT to restrict result set size.');
    }

    return analysis;
  }

  estimateComplexity(query) {
    const upperQuery = query.toUpperCase();
    let complexity = 1;

    if (upperQuery.includes('JOIN')) complexity += 2;
    if (upperQuery.includes('SUBQUERY') || upperQuery.includes('SELECT (')) complexity += 2;
    if (upperQuery.includes('GROUP BY')) complexity += 1;
    if (upperQuery.includes('ORDER BY')) complexity += 1;
    if (upperQuery.includes('DISTINCT')) complexity += 1;
    if (upperQuery.includes('LIKE')) complexity += 1;

    return complexity <= 3 ? 'low' : complexity <= 6 ? 'medium' : 'high';
  }
}

class BatchQueryOptimizer {
  constructor(db) {
    this.db = db;
  }

  async batchSelect(table, ids, options = {}) {
    const { batchSize = 100, cache = null } = options;

    if (!ids || ids.length === 0) {
      return [];
    }

    const results = [];
    for (let i = 0; i < ids.length; i += batchSize) {
      const batch = ids.slice(i, i + batchSize);
      const query = `SELECT * FROM ${table} WHERE id = ANY($1)`;
      const params = [batch];

      if (cache) {
        const cached = cache.get(query, params);
        if (cached) {
          results.push(...cached.rows);
          continue;
        }
      }

      const result = await this.db.query(query, params);
      if (cache) {
        cache.set(query, params, result);
      }
      results.push(...result.rows);
    }

    return results;
  }

  async batchInsert(table, rows, options = {}) {
    const { batchSize = 100, returning = '*' } = options;

    if (!rows || rows.length === 0) {
      return [];
    }

    const results = [];
    for (let i = 0; i < rows.length; i += batchSize) {
      const batch = rows.slice(i, i + batchSize);
      const columns = Object.keys(batch[0]);
      const values = batch.map((row, rowIndex) =>
        columns.map((col, colIndex) => `$${rowIndex * columns.length + colIndex + 1}`)
      ).flat();

      const placeholders = batch.map((_, rowIndex) =>
        `(${columns.map((_, colIndex) => `$${rowIndex * columns.length + colIndex + 1}`).join(', ')})`
      ).join(', ');

      const query = `
        INSERT INTO ${table} (${columns.join(', ')})
        VALUES ${placeholders}
        ${returning ? `RETURNING ${returning}` : ''}
      `;

      const params = batch.flatMap(row => columns.map(col => row[col]));
      const result = await this.db.query(query, params);
      results.push(...result.rows);
    }

    return results;
  }

  async batchUpdate(table, updates, options = {}) {
    const { batchSize = 100, returning = '*' } = options;

    if (!updates || updates.length === 0) {
      return [];
    }

    const results = [];
    for (let i = 0; i < updates.length; i += batchSize) {
      const batch = updates.slice(i, i + batchSize);

      for (const update of batch) {
        const { id, ...fields } = update;
        const setClause = Object.keys(fields)
          .map((col, idx) => `${col} = $${idx + 2}`)
          .join(', ');
        const query = `
          UPDATE ${table}
          SET ${setClause}
          WHERE id = $1
          ${returning ? `RETURNING ${returning}` : ''}
        `;
        const params = [id, ...Object.values(fields)];
        const result = await this.db.query(query, params);
        if (result.rows.length > 0) {
          results.push(...result.rows);
        }
      }
    }

    return results;
  }
}

class QueryOptimizer {
  constructor(db, options = {}) {
    this.db = db;
    this.cache = new QueryCache(options.cacheSize, options.cacheTtl);
    this.analyzer = new QueryAnalyzer();
    this.batchOptimizer = new BatchQueryOptimizer(db);
  }

  async query(queryText, params = [], options = {}) {
    const { useCache = true, analyze = false } = options;

    if (useCache) {
      const cached = this.cache.get(queryText, params);
      if (cached) {
        if (analyze) {
          return {
            rows: cached.rows,
            fromCache: true,
            analysis: null,
          };
        }
        return { rows: cached.rows, fromCache: true };
      }
    }

    const start = Date.now();
    const result = await this.db.query(queryText, params);
    const executionTime = Date.now() - start;

    if (useCache) {
      this.cache.set(queryText, params, result);
    }

    if (analyze) {
      return {
        rows: result.rows,
        fromCache: false,
        analysis: this.analyzer.analyze(queryText, executionTime),
      };
    }

    return { rows: result.rows, fromCache: false };
  }

  clearCache() {
    this.cache.clear();
  }

  getCacheStats() {
    return {
      size: this.cache.getSize(),
      maxSize: this.cache.maxSize,
    };
  }

  async batchSelect(table, ids, options = {}) {
    return this.batchOptimizer.batchSelect(table, ids, {
      ...options,
      cache: options.useCache ? this.cache : null,
    });
  }

  async batchInsert(table, rows, options = {}) {
    return this.batchOptimizer.batchInsert(table, rows, options);
  }

  async batchUpdate(table, updates, options = {}) {
    return this.batchOptimizer.batchUpdate(table, updates, options);
  }

  analyzeQuery(query, executionTime) {
    return this.analyzer.analyze(query, executionTime);
  }
}

module.exports = {
  QueryCache,
  QueryAnalyzer,
  BatchQueryOptimizer,
  QueryOptimizer,
};
