const { Pool } = require('pg');

class QueryOptimizer {
  constructor(config = {}) {
    this.pool = new Pool({
      host: config.host || process.env.DB_HOST || 'localhost',
      port: config.port || parseInt(process.env.DB_PORT || '5432', 10),
      database: config.database || process.env.DB_NAME || 'hjtpx',
      user: config.user || process.env.DB_USER || 'postgres',
      password: config.password || process.env.DB_PASSWORD || 'postgres',
      max: config.max || 20,
      idleTimeoutMillis: config.idleTimeoutMillis || 30000,
      connectionTimeoutMillis: config.connectionTimeoutMillis || 2000,
    });

    this.stats = {
      totalQueries: 0,
      slowQueries: 0,
      failedQueries: 0,
      averageQueryTime: 0,
      slowQueryThreshold: config.slowQueryThreshold || 1000
    };

    this.slowQueryLog = [];
    this.maxSlowQueries = config.maxSlowQueries || 100;
  }

  async execute(query, params = []) {
    const startTime = Date.now();
    this.stats.totalQueries++;

    try {
      const result = await this.pool.query(query, params);
      const duration = Date.now() - startTime;

      this.updateAverageQueryTime(duration);

      if (duration > this.stats.slowQueryThreshold) {
        this.stats.slowQueries++;
        this.logSlowQuery(query, params, duration);
      }

      return {
        rows: result.rows,
        rowCount: result.rowCount,
        duration,
        success: true
      };
    } catch (error) {
      this.stats.failedQueries++;
      return {
        rows: [],
        rowCount: 0,
        duration: Date.now() - startTime,
        success: false,
        error: error.message
      };
    }
  }

  updateAverageQueryTime(newTime) {
    const total = this.stats.totalQueries;
    const current = this.stats.averageQueryTime;
    this.stats.averageQueryTime = ((current * (total - 1)) + newTime) / total;
  }

  logSlowQuery(query, params, duration) {
    const logEntry = {
      query: query.substring(0, 200),
      params: params ? params.map(p => String(p).substring(0, 100)) : [],
      duration,
      timestamp: new Date().toISOString()
    };

    this.slowQueryLog.unshift(logEntry);

    if (this.slowQueryLog.length > this.maxSlowQueries) {
      this.slowQueryLog.pop();
    }

    console.warn(`Slow query detected (${duration}ms):`, query.substring(0, 100));
  }

  async analyzeQuery(query) {
    try {
      const explainResult = await this.pool.query(`EXPLAIN ANALYZE ${query}`);
      return {
        success: true,
        plan: explainResult.rows
      };
    } catch (error) {
      return {
        success: false,
        error: error.message
      };
    }
  }

  async getTableIndexes(tableName) {
    const query = `
      SELECT 
        i.relname AS index_name,
        a.attname AS column_name,
        ix.indisunique AS is_unique,
        ix.indisprimary AS is_primary,
        pg_get_indexdef(ix.indexrelid) AS index_def
      FROM 
        pg_index ix
        JOIN pg_class t ON t.oid = ix.indrelid
        JOIN pg_class i ON i.oid = ix.indexrelid
        JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
      WHERE 
        t.relname = $1
        AND NOT ix.indisprimary
      ORDER BY 
        i.relname, a.attnum;
    `;

    const result = await this.execute(query, [tableName]);
    return result.rows;
  }

  async getTableStats(tableName) {
    const query = `
      SELECT 
        schemaname,
        relname,
        n_live_tup AS row_count,
        n_dead_tup AS dead_rows,
        last_vacuum,
        last_autovacuum,
        last_analyze,
        last_autoanalyze,
        vacuum_count,
        autovacuum_count,
        analyze_count,
        autoanalyze_count
      FROM 
        pg_stat_user_tables
      WHERE 
        relname = $1;
    `;

    const result = await this.execute(query, [tableName]);
    return result.rows[0] || null;
  }

  async getMissingIndexes() {
    const query = `
      SELECT 
        schemaname,
        tablename,
        seq_scan,
        idx_scan,
        seq_scan::float / NULLIF(idx_scan, 0) AS seq_scan_ratio,
        pg_size_pretty(pg_relation_size(schemaname || '.' || tablename)) AS table_size
      FROM 
        pg_stat_user_tables
      WHERE 
        seq_scan > idx_scan * 10
        AND pg_relation_size(schemaname || '.' || tablename) > 1024 * 1024
      ORDER BY 
        seq_scan DESC;
    `;

    const result = await this.execute(query);
    return result.rows;
  }

  async getIndexUsage() {
    const query = `
      SELECT 
        schemaname,
        tablename,
        indexname,
        idx_scan,
        idx_tup_read,
        idx_tup_fetch,
        pg_size_pretty(pg_relation_size(indexrelname)) AS index_size
      FROM 
        pg_stat_user_indexes ui
        JOIN pg_index i ON ui.indexrelid = i.indexrelid
      WHERE 
        idx_scan = 0
        AND NOT i.indisprimary
      ORDER BY 
        pg_relation_size(indexrelname) DESC;
    `;

    const result = await this.execute(query);
    return result.rows;
  }

  async getLargeTables() {
    const query = `
      SELECT 
        schemaname,
        relname AS table_name,
        n_live_tup AS row_count,
        n_dead_tup AS dead_rows,
        pg_size_pretty(pg_total_relation_size(relid)) AS total_size,
        pg_size_pretty(pg_relation_size(relid)) AS table_size,
        pg_size_pretty(pg_total_relation_size(relid) - pg_relation_size(relid)) AS indexes_size
      FROM 
        pg_stat_user_tables
      WHERE 
        n_live_tup > 0
      ORDER BY 
        n_live_tup DESC;
    `;

    const result = await this.execute(query);
    return result.rows;
  }

  async getSlowestQueries(limit = 20) {
    const query = `
      SELECT 
        query,
        calls,
        mean_exec_time,
        total_exec_time,
        min_exec_time,
        max_exec_time,
        stddev_exec_time,
        rows,
        shared_blks_hit,
        shared_blks_read
      FROM 
        pg_stat_statements
      WHERE 
        query NOT LIKE '%pg_stat_statements%'
      ORDER BY 
        mean_exec_time DESC
      LIMIT $1;
    `;

    try {
      const result = await this.pool.query(query, [limit]);
      return result.rows;
    } catch (error) {
      console.warn('pg_stat_statements extension may not be enabled:', error.message);
      return this.slowQueryLog.slice(0, limit);
    }
  }

  getStats() {
    return {
      ...this.stats,
      slowQueryLogSize: this.slowQueryLog.length,
      poolStats: {
        total: this.pool.totalCount,
        idle: this.pool.idleCount,
        waiting: this.pool.waitingCount
      }
    };
  }

  getSlowQueryLog() {
    return [...this.slowQueryLog];
  }

  async close() {
    await this.pool.end();
  }
}

class IndexRecommendation {
  constructor(optimizer) {
    this.optimizer = optimizer;
    this.recommendations = [];
  }

  async analyze() {
    this.recommendations = [];

    const missingIndexes = await this.optimizer.getMissingIndexes();
    for (const table of missingIndexes) {
      this.recommendations.push({
        type: 'missing_index',
        priority: 'high',
        table: table.tablename,
        reason: `High seq_scan ratio (${table.seq_scan_ratio.toFixed(2)})`,
        suggestion: `CREATE INDEX idx_${table.tablename}_ ON ${table.schemaname}.${table.tablename}(column_name);`
      });
    }

    const unusedIndexes = await this.optimizer.getIndexUsage();
    for (const index of unusedIndexes) {
      this.recommendations.push({
        type: 'unused_index',
        priority: 'medium',
        table: index.tablename,
        index: index.indexname,
        reason: 'Index has never been scanned',
        suggestion: `Consider dropping: DROP INDEX ${index.schemaname}.${index.indexname};`
      });
    }

    return this.recommendations;
  }

  getRecommendations() {
    return [...this.recommendations];
  }
}

module.exports = { QueryOptimizer, IndexRecommendation };
