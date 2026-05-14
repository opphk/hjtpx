const schemas = require('../utils/validationRules');

const validator = (schemaName, property = 'body') => {
  return (req, res, next) => {
    const schema = schemas[schemaName];
    
    if (!schema) {
      return res.status(500).json({
        success: false,
        error: {
          code: 'VALIDATION_SCHEMA_NOT_FOUND',
          message: `Validation schema '${schemaName}' not found`
        }
      });
    }

    const dataToValidate = property === 'body' ? req.body :
                          property === 'query' ? req.query :
                          property === 'params' ? req.params : req.body;

    const { error, value } = schema.validate(dataToValidate, {
      abortEarly: false,
      stripUnknown: true
    });

    if (error) {
      const errors = error.details.map(detail => ({
        field: detail.path.join('.'),
        message: detail.message,
        type: detail.type
      }));

      return res.status(400).json({
        success: false,
        error: {
          code: 'VALIDATION_ERROR',
          message: 'Validation failed',
          details: errors
        }
      });
    }

    if (property === 'body') {
      req.body = value;
    } else if (property === 'query') {
      req.query = value;
    } else if (property === 'params') {
      req.params = value;
    }

    next();
  };
};

module.exports = validator;
