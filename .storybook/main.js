module.exports = {
  stories: [
    '../src/frontend/src/**/*.stories.mdx',
    '../src/frontend/src/**/*.stories.@(js|jsx|ts|tsx)'
  ],
  addons: [
    '@storybook/addon-essentials',
    '@storybook/addon-a11y',
    '@storybook/addon-controls',
    '@storybook/addon-actions',
    '@storybook/addon-jest'
  ],
  framework: '@storybook/react',
  core: {
    builder: 'webpack5'
  },
  staticDirs: ['../public'],
  features: {
    postcss: false,
    babelModeV7: true
  }
};
