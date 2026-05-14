const pool = require('../../../config/database/db');

const DANGEROUS_PATTERNS = [
  /(\b(SELECT|INSERT|UPDATE|DELETE|DROP|CREATE|ALTER|TRUNCATE|EXEC|EXECUTE|UNION|SCRIPT)\b)/i,
  /(\b(OR|AND)\b\s*\d+\s*[=<>])/i,
  /('|(\\')|(;--)|(\-\-)|(\/\*)|(\*\/)|(@@)|(@\w+))/,
  /(0x[0-9a-f]+)/i,
  /((CHAR|CHR|ASCII|UNICODE)\s*\()/i,
  /((WAITFOR|DELAY)\s+@\w+)/i,
  /((BENCHMARK|SLEEP)\s*\()/i,
  /(INTO\s+(OUTFILE|DUMPFILE))/i,
  /((LOAD_FILE|OUTFILE|DUMPFILE)\s*\()/i,
  /((SLEEP|BENCHMARK)\s*\()/i,
  /((REVERSE|REPLACE)\s*\()/i,
  /((CONCAT|CHAR)\s*\()/i,
  /(<\s*script)/i,
  /(javascript\s*:)/i,
  /(\bunion\b.*\bselect\b)/i,
  /(select\s+.*\bfrom\b)/i,
  /(insert\s+into\b)/i,
  /(delete\s+from\b)/i,
  /(drop\s+table\b)/i,
  /(update\b.*\bset\b)/i
];

const RESERVED_KEYWORDS = [
  'SELECT', 'INSERT', 'UPDATE', 'DELETE', 'DROP', 'CREATE', 'ALTER', 'TRUNCATE',
  'EXEC', 'EXECUTE', 'UNION', 'SCRIPT', 'OR', 'AND', 'WHERE', 'FROM', 'TABLE',
  'DATABASE', 'SCHEMA', 'INDEX', 'VIEW', 'PROCEDURE', 'FUNCTION', 'TRIGGER'
];

function validateInput(input, options = {}) {
  const { allowSqlWildcards = false, maxLength = null } = options;
  
  if (input === null || input === undefined) {
    return { valid: true, sanitized: null };
  }
  
  let value = String(input);
  
  if (maxLength && value.length > maxLength) {
    value = value.substring(0, maxLength);
  }
  
  for (const pattern of DANGEROUS_PATTERNS) {
    if (pattern.test(value)) {
      return { valid: false, sanitized: null, pattern: pattern.toString() };
    }
  }
  
  if (!allowSqlWildcards) {
    value = value.replace(/%/g, '\\%').replace(/_/g, '\\_');
  }
  
  return { valid: true, sanitized: value };
}

function validateObject(obj, options = {}) {
  const errors = [];
  const sanitized = {};
  
  for (const [key, value] of Object.entries(obj)) {
    if (typeof value === 'string') {
      const result = validateInput(value, options);
      if (!result.valid) {
        errors.push({
          field: key,
          reason: 'Potentially dangerous SQL pattern detected',
          pattern: result.pattern
        });
        sanitized[key] = value;
      } else {
        sanitized[key] = result.sanitized;
      }
    } else if (Array.isArray(value)) {
      const validatedArray = [];
      for (let i = 0; i < value.length; i++) {
        if (typeof value[i] === 'string') {
          const result = validateInput(value[i], options);
          if (!result.valid) {
            errors.push({
              field: `${key}[${i}]`,
              reason: 'Potentially dangerous SQL pattern detected'
            });
            validatedArray.push(value[i]);
          } else {
            validatedArray.push(result.sanitized);
          }
        } else {
          validatedArray.push(value[i]);
        }
      }
      sanitized[key] = validatedArray;
    } else if (value && typeof value === 'object') {
      const nestedResult = validateObject(value, options);
      if (nestedResult.errors.length > 0) {
        errors.push(...nestedResult.errors);
      }
      sanitized[key] = nestedResult.sanitized;
    } else {
      sanitized[key] = value;
    }
  }
  
  return { errors, sanitized };
}

function escapeIdentifier(identifier) {
  if (!identifier || typeof identifier !== 'string') {
    throw new Error('Invalid identifier');
  }
  
  const trimmed = identifier.trim();
  
  if (!/^[a-zA-Z_][a-zA-Z0-9_]*$/.test(trimmed)) {
    throw new Error(`Invalid identifier: ${identifier}`);
  }
  
  const upper = trimmed.toUpperCase();
  if (RESERVED_KEYWORDS.includes(upper)) {
    throw new Error(`Reserved keyword cannot be used as identifier: ${identifier}`);
  }
  
  return `"${trimmed}"`;
}

function buildWhereClause(conditions, paramPrefix = 'p') {
  const clauses = [];
  const params = [];
  let paramIndex = 1;
  
  for (const [key, value] of Object.entries(conditions)) {
    if (value === undefined || value === null) {
      clauses.push(`${escapeIdentifier(key)} IS NULL`);
    } else if (Array.isArray(value)) {
      const placeholders = value.map(() => `$${paramIndex++}`).join(', ');
      clauses.push(`${escapeIdentifier(key)} IN (${placeholders})`);
      params.push(...value);
    } else {
      clauses.push(`${escapeIdentifier(key)} = $${paramIndex++}`);
      params.push(value);
    }
  }
  
  return {
    clause: clauses.length > 0 ? `WHERE ${clauses.join(' AND ')}` : '',
    params
  };
}

function buildUpdateSet(updates, paramPrefix = 'p') {
  const setClauses = [];
  const params = [];
  let paramIndex = 1;
  
  for (const [key, value] of Object.entries(updates)) {
    if (value !== undefined) {
      setClauses.push(`${escapeIdentifier(key)} = $${paramIndex++}`);
      params.push(value);
    }
  }
  
  return {
    clause: setClauses.length > 0 ? `SET ${setClauses.join(', ')}` : '',
    params
  };
}

function safeQuery(query, params = []) {
  const validation = validateQuery(query);
  if (!validation.valid) {
    throw new Error('Invalid SQL query detected');
  }
  
  const paramValidation = validateObject(params);
  if (paramValidation.errors.length > 0) {
    throw new Error('Potentially dangerous parameters detected');
  }
  
  return pool.query(query, paramValidation.sanitized);
}

function validateQuery(query) {
  if (!query || typeof query !== 'string') {
    return { valid: false, reason: 'Invalid query type' };
  }
  
  const trimmed = query.trim().toUpperCase();
  
  if (/^\s*(SELECT|INSERT|UPDATE|DELETE)\s+\*/.test(trimmed)) {
    return { valid: false, reason: 'SELECT * is not allowed for security reasons' };
  }
  
  if (/\bINTO\s+(OUTFILE|DUMPFILE)\b/i.test(trimmed)) {
    return { valid: false, reason: 'File operations are not allowed' };
  }
  
  if (/\bLOAD_FILE\s*\(/i.test(trimmed)) {
    return { valid: false, reason: 'LOAD_FILE is not allowed' };
  }
  
  if (/\bSLEEP\s*\(/i.test(trimmed)) {
    return { valid: false, reason: 'SLEEP functions are not allowed' };
  }
  
  if (/\bBENCHMARK\s*\(/i.test(trimmed)) {
    return { valid: false, reason: 'BENCHMARK is not allowed' };
  }
  
  return { valid: true };
}

function checkForSqlInjection(value) {
  if (typeof value !== 'string') {
    return false;
  }
  
  for (const pattern of DANGEROUS_PATTERNS) {
    if (pattern.test(value)) {
      return true;
    }
  }
  
  return false;
}

function createSafeQueryBuilder(tableName) {
  const table = escapeIdentifier(tableName);
  
  return {
    select: (columns = ['*'], conditions = {}) => {
      const cols = columns.map(col => {
        if (col === '*') return '*';
        return escapeIdentifier(col);
      }).join(', ');
      
      const { clause, params } = buildWhereClause(conditions);
      const query = `SELECT ${cols} FROM ${table} ${clause}`;
      
      return { query, params };
    },
    
    insert: (data) => {
      const keys = Object.keys(data);
      const columns = keys.map(k => escapeIdentifier(k)).join(', ');
      const placeholders = keys.map((_, i) => `$${i + 1}`).join(', ');
      const values = Object.values(data);
      
      const query = `INSERT INTO ${table} (${columns}) VALUES (${placeholders}) RETURNING *`;
      
      return { query, params: values };
    },
    
    update: (data, conditions) => {
      const { clause: setClause, params: setParams } = buildUpdateSet(data);
      const { clause: whereClause, params: whereParams } = buildWhereClause(conditions);
      
      const query = `UPDATE ${table} ${setClause} ${whereClause} RETURNING *`;
      
      return { query, params: [...setParams, ...whereParams] };
    },
    
    delete: (conditions) => {
      const { clause, params } = buildWhereClause(conditions);
      
      const query = `DELETE FROM ${table} ${clause} RETURNING *`;
      
      return { query, params };
    },
    
    count: (conditions = {}) => {
      const { clause, params } = buildWhereClause(conditions);
      
      const query = `SELECT COUNT(*) as count FROM ${table} ${clause}`;
      
      return { query, params };
    }
  };
}

function logSecurityEvent(eventType, details) {
  console.warn('[SECURITY]', eventType, JSON.stringify(details));
}

function createSqlInjectionProtection(options = {}) {
  const {
    enabled = true,
    logEvents = true,
    throwOnDetection = true,
    sanitizeInputs = true
  } = options;
  
  return async (req, res, next) => {
    if (!enabled) {
      return next();
    }
    
    const checkData = (data, location) => {
      if (!data) return;
      
      const toCheck = typeof data === 'string' ? { _: data } : data;
      const result = validateObject(toCheck);
      
      if (result.errors.length > 0 && logEvents) {
        logSecurityEvent('SQL_INJECTION_ATTEMPT', {
          path: req.path,
          method: req.method,
          ip: req.ip,
          location,
          errors: result.errors
        });
      }
      
      if (result.errors.length > 0 && throwOnDetection) {
        return res.status(400).json({
          success: false,
          error: 'Invalid input detected'
        });
      }
      
      if (sanitizeInputs && result.sanitized) {
        if (location === 'body') {
          req.body = result.sanitized;
        } else if (location === 'query') {
          req.query = result.sanitized;
        }
      }
    };
    
    checkData(req.body, 'body');
    checkData(req.query, 'query');
    checkData(req.params, 'params');
    
    next();
  };
}

module.exports = {
  validateInput,
  validateObject,
  escapeIdentifier,
  buildWhereClause,
  buildUpdateSet,
  safeQuery,
  validateQuery,
  checkForSqlInjection,
  createSafeQueryBuilder,
  logSecurityEvent,
  createSqlInjectionProtection,
  DANGEROUS_PATTERNS,
  RESERVED_KEYWORDS
};
