import React, { useState, useEffect, useCallback, useRef } from 'react';
import PropTypes from 'prop-types';
import axios from 'axios';

const SearchBar = ({
  model,
  placeholder = 'Search...',
  onSearch,
  onResultSelect,
  debounceDelay = 300,
  maxSuggestions = 10,
  className = '',
  autoFocus = false,
  showFilters = false,
  filters = []
}) => {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState([]);
  const [suggestions, setSuggestions] = useState([]);
  const [loading, setLoading] = useState(false);
  const [showDropdown, setShowDropdown] = useState(false);
  const [activeFilter, setActiveFilter] = useState(null);
  const [page, setPage] = useState(1);
  const [hasMore, setHasMore] = useState(false);

  const inputRef = useRef(null);
  const dropdownRef = useRef(null);
  const debounceTimer = useRef(null);

  const debouncedSearch = useCallback(
    (searchQuery) => {
      if (debounceTimer.current) {
        clearTimeout(debounceTimer.current);
      }

      debounceTimer.current = setTimeout(() => {
        performSearch(searchQuery);
      }, debounceDelay);
    },
    [debounceDelay, model]
  );

  const performSearch = async (searchQuery) => {
    if (!searchQuery.trim() && !activeFilter) {
      setResults([]);
      setShowDropdown(false);
      return;
    }

    setLoading(true);

    try {
      const params = {
        q: searchQuery,
        page: 1,
        limit: 20
      };

      if (activeFilter) {
        params.filter = JSON.stringify(activeFilter);
      }

      const response = await axios.get(`/api/v1/search/${model}`, {
        headers: {
          Authorization: `Bearer ${localStorage.getItem('token')}`
        },
        params
      });

      if (response.data.success) {
        setResults(response.data.data);
        setHasMore(response.data.pagination?.hasNext || false);
        setPage(1);
        setShowDropdown(true);

        if (onSearch) {
          onSearch(response.data);
        }
      }
    } catch (error) {
      console.error('Search error:', error);
      setResults([]);
    } finally {
      setLoading(false);
    }
  };

  const fetchSuggestions = useCallback(
    async (searchQuery) => {
      if (!searchQuery.trim() || searchQuery.length < 2) {
        setSuggestions([]);
        return;
      }

      try {
        const response = await axios.get(
          `/api/v1/search/${model}/suggestions`,
          {
            headers: {
              Authorization: `Bearer ${localStorage.getItem('token')}`
            },
            params: {
              field: 'name',
              q: searchQuery,
              limit: maxSuggestions
            }
          }
        );

        if (response.data.success) {
          setSuggestions(response.data.data);
        }
      } catch (error) {
        console.error('Suggestions error:', error);
        setSuggestions([]);
      }
    },
    [model, maxSuggestions]
  );

  const loadMore = async () => {
    if (!hasMore || loading) return;

    setLoading(true);

    try {
      const params = {
        q: query,
        page: page + 1,
        limit: 20
      };

      if (activeFilter) {
        params.filter = JSON.stringify(activeFilter);
      }

      const response = await axios.get(`/api/v1/search/${model}`, {
        headers: {
          Authorization: `Bearer ${localStorage.getItem('token')}`
        },
        params
      });

      if (response.data.success) {
        setResults((prev) => [...prev, ...response.data.data]);
        setHasMore(response.data.pagination?.hasNext || false);
        setPage((prev) => prev + 1);
      }
    } catch (error) {
      console.error('Load more error:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleInputChange = (e) => {
    const value = e.target.value;
    setQuery(value);
    debouncedSearch(value);
    fetchSuggestions(value);
  };

  const handleResultClick = (result) => {
    setQuery('');
    setShowDropdown(false);
    setResults([]);

    if (onResultSelect) {
      onResultSelect(result);
    }
  };

  const handleFilterChange = (filter) => {
    setActiveFilter(filter);
    performSearch(query);
  };

  const handleKeyDown = (e) => {
    if (e.key === 'Escape') {
      setShowDropdown(false);
      setQuery('');
      setResults([]);
    }

    if (e.key === 'Enter' && query.trim()) {
      setShowDropdown(false);
      performSearch(query);
    }
  };

  const handleClickOutside = (e) => {
    if (
      dropdownRef.current &&
      !dropdownRef.current.contains(e.target) &&
      !inputRef.current.contains(e.target)
    ) {
      setShowDropdown(false);
    }
  };

  useEffect(() => {
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  useEffect(() => {
    if (autoFocus && inputRef.current) {
      inputRef.current.focus();
    }
  }, [autoFocus]);

  return (
    <div className={`search-bar-container ${className}`}>
      {showFilters && filters.length > 0 && (
        <div className="search-filters">
          {filters.map((filter) => (
            <select
              key={filter.field}
              value={activeFilter?.[filter.field]?.value || ''}
              onChange={(e) =>
                handleFilterChange({
                  [filter.field]: {
                    operator: filter.operator || 'eq',
                    value: e.target.value
                  }
                })
              }
              className="search-filter-select"
            >
              <option value="">{filter.label}</option>
              {filter.options.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
          ))}
        </div>
      )}

      <div className="search-input-container">
        <input
          ref={inputRef}
          type="text"
          value={query}
          onChange={handleInputChange}
          onKeyDown={handleKeyDown}
          onFocus={() => query && setShowDropdown(true)}
          placeholder={placeholder}
          className="search-input"
          aria-label="Search"
        />

        {loading && <div className="search-loading">Loading...</div>}

        {showDropdown && (
          <div ref={dropdownRef} className="search-dropdown">
            {results.length === 0 && query && !loading && (
              <div className="search-no-results">No results found</div>
            )}

            {suggestions.length > 0 && query && (
              <div className="search-suggestions">
                <div className="search-suggestions-header">Suggestions</div>
                {suggestions.map((suggestion, index) => (
                  <div
                    key={index}
                    className="search-suggestion-item"
                    onClick={() => {
                      setQuery(suggestion);
                      performSearch(suggestion);
                    }}
                  >
                    {suggestion}
                  </div>
                ))}
              </div>
            )}

            {results.length > 0 && (
              <div className="search-results">
                {results.map((result, index) => (
                  <div
                    key={result.id || index}
                    className="search-result-item"
                    onClick={() => handleResultClick(result)}
                  >
                    <div className="search-result-title">
                      {result.name || result.title || result.email}
                    </div>
                    <div className="search-result-subtitle">
                      {result.description || result.message || ''}
                    </div>
                  </div>
                ))}

                {hasMore && (
                  <div className="search-load-more" onClick={loadMore}>
                    {loading ? 'Loading...' : 'Load more'}
                  </div>
                )}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
};

SearchBar.propTypes = {
  model: PropTypes.string.isRequired,
  placeholder: PropTypes.string,
  onSearch: PropTypes.func,
  onResultSelect: PropTypes.func,
  debounceDelay: PropTypes.number,
  maxSuggestions: PropTypes.number,
  className: PropTypes.string,
  autoFocus: PropTypes.bool,
  showFilters: PropTypes.bool,
  filters: PropTypes.arrayOf(
    PropTypes.shape({
      field: PropTypes.string.isRequired,
      label: PropTypes.string.isRequired,
      operator: PropTypes.string,
      options: PropTypes.arrayOf(
        PropTypes.shape({
          value: PropTypes.any.isRequired,
          label: PropTypes.string.isRequired
        })
      ).isRequired
    })
  )
};

SearchBar.defaultProps = {
  placeholder: 'Search...',
  onSearch: null,
  onResultSelect: null,
  debounceDelay: 300,
  maxSuggestions: 10,
  className: '',
  autoFocus: false,
  showFilters: false,
  filters: []
};

export default SearchBar;
