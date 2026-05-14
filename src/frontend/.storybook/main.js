const path = require('path');

module.exports = {
  stories: [
    '../stories/**/*.stories.mdx',
    '../stories/**/*.stories.js',
    '../stories/**/*.stories.jsx',
    '../components/**/*.stories.mdx',
    '../components/**/*.stories.js',
    '../components/**/*.stories.jsx',
  ],
  addons: [
    '@storybook/addon-essentials',
    '@storybook/addon-a11y',
    '@storybook/addon-docs',
    '@storybook/addon-controls',
    '@storybook/addon-actions',
    '@storybook/addon-jest',
  ],
  webpackFinal: async (config) => {
    config.resolve.alias = {
      ...config.resolve.alias,
      '@': path.resolve(__dirname, '../'),
      '@components': path.resolve(__dirname, '../components'),
      '@hooks': path.resolve(__dirname, '../hooks'),
      '@context': path.resolve(__dirname, '../context'),
      '@services': path.resolve(__dirname, '../services'),
      '@utils': path.resolve(__dirname, '../utils'),
    };
    return config;
  },
  docs: {
    autodocs: 'tag',
  },
};
