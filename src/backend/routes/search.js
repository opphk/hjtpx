const express = require('express');
const router = express.Router();
const searchService = require('../services/searchService');
const { authMiddleware } = require('../middleware/auth');

router.use(authMiddleware);

router.get('/:model', async (req, res, next) => {
  try {
    const { model } = req.params;
    const {
      q,
      filter,
      sort,
      page,
      limit,
      fields,
      include_count
    } = req.query;

    const filters = filter ? parseFilter(filter) : {};

    const result = await searchService.search(model, {
      query: q,
      filters,
      sort: parseSort(sort),
      pagination: {
        page: parseInt(page) || 1,
        limit: parseInt(limit) || 20
      },
      fields: fields ? fields.split(',') : null,
      includeCount: include_count !== 'false'
    });

    res.json({
      success: true,
      data: result.data,
      pagination: result.pagination,
      meta: result.meta
    });
  } catch (error) {
    next(error);
  }
});

router.post('/:model/advanced', async (req, res, next) => {
  try {
    const { model } = req.params;
    const {
      conditions,
      logic = 'AND',
      sort,
      pagination,
      fields,
      groupBy,
      having
    } = req.body;

    const result = await searchService.advancedSearch(model, {
      conditions,
      logic,
      sort: parseSort(sort),
      pagination: pagination || { page: 1, limit: 20 },
      fields,
      groupBy,
      having
    });

    res.json({
      success: true,
      data: result.data,
      meta: result.meta
    });
  } catch (error) {
    next(error);
  }
});

router.get('/:model/suggestions', async (req, res, next) => {
  try {
    const { model } = req.params;
    const { field, q, limit = 10 } = req.query;

    if (!field) {
      return res.status(400).json({
        success: false,
        error: 'Field parameter is required'
      });
    }

    const suggestions = await searchService.getSuggestions(
      model,
      field,
      q || '',
      parseInt(limit) || 10
    );

    res.json({
      success: true,
      data: suggestions
    });
  } catch (error) {
    next(error);
  }
});

router.get('/:model/fields', async (req, res, next) => {
  try {
    const { model } = req.params;

    if (!searchService.SEARCHABLE_MODELS[model]) {
      return res.status(404).json({
        success: false,
        error: `Model "${model}" not found or not searchable`
      });
    }

    res.json({
      success: true,
      data: {
        model,
        searchableFields: searchService.SEARCHABLE_MODELS[model],
        filterOperators: Object.keys(searchService.FILTER_OPERATORS)
      }
    });
  } catch (error) {
    next(error);
  }
});

function parseFilter(filterString) {
  try {
    return JSON.parse(filterString);
  } catch {
    return {};
  }
}

function parseSort(sortString) {
  if (!sortString) {
    return { field: 'created_at', order: 'desc' };
  }

  const parts = sortString.split(':');
  return {
    field: parts[0] || 'created_at',
    order: parts[1] === 'asc' ? 'asc' : 'desc'
  };
}

module.exports = router;
