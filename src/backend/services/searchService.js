const pool = require('../../../config/database/db');
const searchOptimizer = require('../utils/searchOptimizer');

const SEARCHABLE_MODELS = {
  users: ['email', 'name', 'role'],
  products: ['name', 'description', 'sku', 'category'],
  orders: ['order_number', 'status', 'customer_name'],
  notifications: ['title', 'message', 'type']
};

const FILTER_OPERATORS = {
  eq: '=',
  ne: '!=',
  gt: '>',
  gte: '>=',
  lt: '<',
  lte: '<=',
  in: 'IN',
  nin: 'NOT IN',
  like: 'ILIKE',
  between: 'BETWEEN',
  isnull: 'IS NULL',
  notnull: 'IS NOT NULL'
};

async function search(model, options = {}) {
  const {
    query,
    filters = {},
    sort = {},
    pagination = { page: 1, limit: 20 },
    fields = null,
    includeCount = true
  } = options;

  if (!SEARCHABLE_MODELS[model]) {
    throw new Error(`Model "${model}" is not searchable`);
  }

  const searchableFields = SEARCHABLE_MODELS[model];
  const page = Math.max(1, parseInt(pagination.page) || 1);
  const limit = Math.min(100, Math.max(1, parseInt(pagination.limit) || 20));
  const offset = (page - 1) * limit;

  let whereConditions = [];
  let queryParams = [];
  let paramIndex = 1;

  if (query && query.trim()) {
    const searchConditions = searchableFields.map(field => {
      const paramName = `search_${paramIndex}`;
      queryParams.push(`%${query.trim()}%`);
      paramIndex++;
      return `${field} ILIKE $${paramName}`;
    });
    whereConditions.push(`(${searchConditions.join(' OR ')})`);
  }

  for (const [field, filter] of Object.entries(filters)) {
    if (!filter || filter.value === undefined) continue;

    const { operator = 'eq', value } = filter;

    if (operator === 'isnull') {
      whereConditions.push(`${field} IS NULL`);
    } else if (operator === 'notnull') {
      whereConditions.push(`${field} IS NOT NULL`);
    } else if (operator === 'in' || operator === 'nin') {
      const values = Array.isArray(value) ? value : value.split(',');
      if (values.length > 0) {
        const placeholders = values.map((_, i) => `$${paramIndex + i}`).join(', ');
        queryParams.push(...values);
        paramIndex += values.length;
        const sqlOp = operator === 'in' ? 'IN' : 'NOT IN';
        whereConditions.push(`${field} ${sqlOp} (${placeholders})`);
      }
    } else if (operator === 'between') {
      if (Array.isArray(value) && value.length === 2) {
        queryParams.push(value[0], value[1]);
        paramIndex += 2;
        whereConditions.push(`${field} BETWEEN $${paramIndex - 1} AND $${paramIndex}`);
      }
    } else {
      const sqlOp = FILTER_OPERATORS[operator] || '=';
      queryParams.push(value);
      queryParams.push(`%${value}%`);
      if (sqlOp === 'ILIKE') {
        whereConditions.push(`${field} ${sqlOp} $${paramIndex}`);
        paramIndex++;
      } else {
        whereConditions.push(`${field} ${sqlOp} $${paramIndex}`);
        paramIndex++;
      }
    }
  }

  let sql = 'SELECT';
  if (fields && fields.length > 0) {
    sql += ` ${fields.join(', ')}`;
  } else {
    sql += ' *';
  }
  sql += ` FROM ${model}`;

  if (whereConditions.length > 0) {
    sql += ` WHERE ${whereConditions.join(' AND ')}`;
  }

  const sortField = sort.field || 'created_at';
  const sortOrder = sort.order === 'asc' ? 'ASC' : 'DESC';
  sql += ` ORDER BY ${sortField} ${sortOrder}`;

  sql += ` LIMIT $${paramIndex} OFFSET $${paramIndex + 1}`;
  queryParams.push(limit, offset);

  const startTime = Date.now();
  const result = await pool.query(sql, queryParams);
  const queryTime = Date.now() - startTime;

  let total = 0;
  if (includeCount) {
    let countSql = `SELECT COUNT(*) FROM ${model}`;
    if (whereConditions.length > 0) {
      countSql += ` WHERE ${whereConditions.join(' AND ')}`;
    }
    const countResult = await pool.query(
      countSql,
      queryParams.slice(0, -2)
    );
    total = parseInt(countResult.rows[0].count);
  }

  return {
    data: result.rows,
    pagination: {
      page,
      limit,
      total,
      pages: Math.ceil(total / limit),
      hasNext: page * limit < total,
      hasPrev: page > 1
    },
    meta: {
      queryTime,
      searchedFields: searchableFields,
      filtersApplied: Object.keys(filters).length
    }
  };
}

async function advancedSearch(model, options = {}) {
  const {
    conditions = [],
    logic = 'AND',
    sort = {},
    pagination = { page: 1, limit: 20 },
    fields = null,
    groupBy = null,
    having = null
  } = options;

  if (!SEARCHABLE_MODELS[model]) {
    throw new Error(`Model "${model}" is not searchable`);
  }

  if (!Array.isArray(conditions) || conditions.length === 0) {
    throw new Error('At least one search condition is required');
  }

  const page = Math.max(1, parseInt(pagination.page) || 1);
  const limit = Math.min(100, Math.max(1, parseInt(pagination.limit) || 20));
  const offset = (page - 1) * limit;

  let queryParams = [];
  let paramIndex = 1;

  const whereClauses = conditions.map(condition => {
    const { field, operator, value, not = false } = condition;

    if (operator === 'isnull') {
      return not ? `${field} IS NOT NULL` : `${field} IS NULL`;
    }

    if (operator === 'in' || operator === 'nin') {
      const values = Array.isArray(value) ? value : value.split(',');
      if (values.length === 0) return null;
      const placeholders = values.map((_, i) => `$${paramIndex + i}`).join(', ');
      queryParams.push(...values);
      paramIndex += values.length;
      const sqlOp = operator === 'in' ? 'IN' : 'NOT IN';
      const clause = `${field} ${sqlOp} (${placeholders})`;
      return not ? `NOT (${clause})` : clause;
    }

    if (operator === 'between') {
      if (!Array.isArray(value) || value.length !== 2) return null;
      queryParams.push(value[0], value[1]);
      const clause = `${field} BETWEEN $${paramIndex} AND $${paramIndex + 1}`;
      paramIndex += 2;
      return not ? `NOT (${clause})` : clause;
    }

    if (operator === 'like' || operator === 'ilike') {
      queryParams.push(`%${value}%`);
      const clause = `${field} ${operator === 'ilike' ? 'ILIKE' : 'LIKE'} $${paramIndex}`;
      paramIndex++;
      return not ? `NOT (${clause})` : clause;
    }

    const sqlOp = FILTER_OPERATORS[operator] || '=';
    queryParams.push(value);
    const clause = `${field} ${sqlOp} $${paramIndex}`;
    paramIndex++;
    return not ? `NOT (${clause})` : clause;
  }).filter(Boolean);

  if (whereClauses.length === 0) {
    throw new Error('No valid conditions provided');
  }

  let sql = 'SELECT';
  if (fields && fields.length > 0) {
    sql += ` ${fields.join(', ')}`;
  } else {
    sql += ' *';
  }
  sql += ` FROM ${model}`;

  sql += ` WHERE ${whereClauses.join(` ${logic} `)}`;

  if (groupBy) {
    sql += ` GROUP BY ${groupBy}`;
    if (having) {
      sql += ` HAVING ${having}`;
    }
  }

  const sortField = sort.field || 'created_at';
  const sortOrder = sort.order === 'asc' ? 'ASC' : 'DESC';
  sql += ` ORDER BY ${sortField} ${sortOrder}`;

  sql += ` LIMIT $${paramIndex} OFFSET $${paramIndex + 1}`;
  queryParams.push(limit, offset);

  const result = await pool.query(sql, queryParams);

  return {
    data: result.rows,
    meta: {
      conditionsApplied: conditions.length,
      logic,
      sort: { field: sortField, order: sortOrder }
    }
  };
}

async function getSuggestions(model, field, query, limit = 10) {
  if (!SEARCHABLE_MODELS[model]) {
    throw new Error(`Model "${model}" is not searchable`);
  }

  if (!SEARCHABLE_MODELS[model].includes(field)) {
    throw new Error(`Field "${field}" is not searchable in model "${model}"`);
  }

  const sql = `
    SELECT DISTINCT ${field}
    FROM ${model}
    WHERE ${field} ILIKE $1
    ORDER BY ${field}
    LIMIT $2
  `;

  const result = await pool.query(sql, [`%${query}%`, limit]);

  return result.rows.map(row => row[field]);
}

module.exports = {
  search,
  advancedSearch,
  getSuggestions,
  SEARCHABLE_MODELS,
  FILTER_OPERATORS
};
