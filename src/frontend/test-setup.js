import { expect, afterEach, vi } from 'vitest';
import { cleanup } from '@testing-library/react';
import '@testing-library/jest-dom';

afterEach(() => {
  cleanup();
});

// Mock localStorage with working implementation
const localStorageMock = (() => {
  let store = {};
  return {
    getItem: (key) => store[key] || null,
    setItem: (key, value) => { store[key] = String(value); },
    removeItem: (key) => { delete store[key]; },
    clear: () => { store = {}; }
  };
})();

global.localStorage = localStorageMock;

global.sessionStorage = {
  getItem: () => null,
  setItem: () => {},
  removeItem: () => {},
  clear: () => {}
};

// Mock jest for compatibility
global.jest = {
  fn: vi.fn
};

Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: (query) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => {}
  })
});

window.IntersectionObserver = class IntersectionObserver {
  constructor() {}
  observe() {}
  unobserve() {}
  disconnect() {}
  takeRecords() {
    return [];
  }
};

window.ResizeObserver = class ResizeObserver {
  constructor() {}
  observe() {}
  unobserve() {}
  disconnect() {}
};

window.PerformanceObserver = class PerformanceObserver {
  constructor() {}
  observe() {}
  disconnect() {}
  takeRecords() {
    return [];
  }
};

global.navigator = {
  ...global.navigator,
  onLine: true,
  connection: {
    effectiveType: '4g',
    downlink: 10,
    addEventListener: () => {},
    removeEventListener: () => {}
  }
};

console.log = vi.fn();
console.warn = vi.fn();
console.error = vi.fn();
