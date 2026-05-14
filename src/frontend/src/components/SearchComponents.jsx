import React, { useState, useEffect, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import debounce from './utils/debounce';

const SearchBar = ({ onSearch, placeholder, suggestions = [] }) => {
  const { t } = useTranslation();
  const [query, setQuery] = useState('');
  const [showSuggestions, setShowSuggestions] = useState(false);
  const [filteredSuggestions, setFilteredSuggestions] = useState([]);
  const [recentSearches, setRecentSearches] = useState([]);
  const inputRef = useRef(null);

  useEffect(() => {
    const saved = localStorage.getItem('recentSearches');
    if (saved) {
      setRecentSearches(JSON.parse(saved));
    }
  }, []);

  const handleInputChange = (e) => {
    const value = e.target.value;
    setQuery(value);

    if (value.trim()) {
      const filtered = suggestions.filter(s =>
        s.toLowerCase().includes(value.toLowerCase())
      );
      setFilteredSuggestions(filtered);
      setShowSuggestions(true);
    } else {
      setShowSuggestions(false);
    }
  };

  const handleSearch = (searchQuery) => {
    const finalQuery = searchQuery || query;
    if (!finalQuery.trim()) return;

    const updatedRecent = [finalQuery, ...recentSearches.filter(s => s !== finalQuery)].slice(0, 10);
    setRecentSearches(updatedRecent);
    localStorage.setItem('recentSearches', JSON.stringify(updatedRecent));

    setShowSuggestions(false);
    if (onSearch) {
      onSearch(finalQuery);
    }
  };

  const handleKeyDown = (e) => {
    if (e.key === 'Enter') {
      handleSearch();
    }
  };

  const clearRecentSearches = () => {
    setRecentSearches([]);
    localStorage.removeItem('recentSearches');
  };

  return (
    <div className="search-bar-container">
      <div className="search-bar">
        <input
          ref={inputRef}
          type="text"
          value={query}
          onChange={handleInputChange}
          onKeyDown={handleKeyDown}
          placeholder={placeholder || t('search.placeholder')}
          className="search-input"
        />
        <button onClick={() => handleSearch()} className="search-button">
          {t('common.search')}
        </button>
      </div>

      {showSuggestions && (
        <div className="suggestions-dropdown">
          {filteredSuggestions.length > 0 && (
            <div className="suggestions-section">
              <h4>{t('search.suggestions')}</h4>
              <ul>
                {filteredSuggestions.map((suggestion, index) => (
                  <li key={index} onClick={() => handleSearch(suggestion)}>
                    {suggestion}
                  </li>
                ))}
              </ul>
            </div>
          )}

          {recentSearches.length > 0 && (
            <div className="recent-searches">
              <div className="section-header">
                <h4>{t('search.recentSearches')}</h4>
                <button onClick={clearRecentSearches} className="clear-btn">
                  {t('common.delete')}
                </button>
              </div>
              <ul>
                {recentSearches.map((search, index) => (
                  <li key={index} onClick={() => handleSearch(search)}>
                    {search}
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}
    </div>
  );
};

const AdvancedFilter = ({ filters, onFilterChange, onApply }) => {
  const { t } = useTranslation();
  const [localFilters, setLocalFilters] = useState(filters);

  const handleChange = (key, value) => {
    const updated = { ...localFilters, [key]: value };
    setLocalFilters(updated);
    if (onFilterChange) {
      onFilterChange(updated);
    }
  };

  const renderFilter = (filter) => {
    switch (filter.type) {
      case 'text':
        return (
          <input
            type="text"
            value={filter.value || ''}
            onChange={(e) => handleChange(filter.key, e.target.value)}
            placeholder={filter.placeholder}
            className="filter-input"
          />
        );

      case 'select':
        return (
          <select
            value={filter.value || ''}
            onChange={(e) => handleChange(filter.key, e.target.value)}
            className="filter-select"
          >
            <option value="">{t('common.filter')}</option>
            {filter.options?.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
        );

      case 'range':
        return (
          <div className="range-inputs">
            <input
              type="number"
              value={filter.min || ''}
              onChange={(e) => handleChange(filter.key, { ...filter.value, min: e.target.value })}
              placeholder="Min"
              className="filter-input-min"
            />
            <span>-</span>
            <input
              type="number"
              value={filter.max || ''}
              onChange={(e) => handleChange(filter.key, { ...filter.value, max: e.target.value })}
              placeholder="Max"
              className="filter-input-max"
            />
          </div>
        );

      case 'date':
        return (
          <input
            type="date"
            value={filter.value || ''}
            onChange={(e) => handleChange(filter.key, e.target.value)}
            className="filter-input"
          />
        );

      case 'multiselect':
        return (
          <div className="multiselect">
            {filter.options?.map((opt) => (
              <label key={opt.value} className="checkbox-label">
                <input
                  type="checkbox"
                  checked={(filter.value || []).includes(opt.value)}
                  onChange={(e) => {
                    const current = filter.value || [];
                    const updated = e.target.checked
                      ? [...current, opt.value]
                      : current.filter(v => v !== opt.value);
                    handleChange(filter.key, updated);
                  }}
                />
                {opt.label}
              </label>
            ))}
          </div>
        );

      default:
        return null;
    }
  };

  return (
    <div className="advanced-filter">
      <div className="filter-header">
        <h3>{t('search.filters')}</h3>
      </div>
      <div className="filter-list">
        {filters.map((filter) => (
          <div key={filter.key} className="filter-item">
            <label>{filter.label}</label>
            {renderFilter(filter)}
          </div>
        ))}
      </div>
      <div className="filter-actions">
        <button onClick={onApply} className="apply-button">
          {t('common.apply')}
        </button>
      </div>
    </div>
  );
};

const SearchResults = ({ results = [], loading = false, onResultClick }) => {
  const { t } = useTranslation();

  if (loading) {
    return (
      <div className="search-results loading">
        <p>{t('common.loading')}</p>
      </div>
    );
  }

  if (results.length === 0) {
    return (
      <div className="search-results empty">
        <p>{t('search.noResults')}</p>
      </div>
    );
  }

  return (
    <div className="search-results">
      {results.map((result, index) => (
        <div
          key={result.id || index}
          className="search-result-item"
          onClick={() => onResultClick?.(result)}
        >
          <h4>{result.title || result.name}</h4>
          <p>{result.description || result.email}</p>
          <div className="result-meta">
            {result.tags?.map((tag, i) => (
              <span key={i} className="tag">{tag}</span>
            ))}
          </div>
        </div>
      ))}
    </div>
  );
};

export { SearchBar, AdvancedFilter, SearchResults };
export default SearchBar;
