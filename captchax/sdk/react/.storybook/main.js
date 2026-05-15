module.exports = {
  stories: ['../stories/**/*.stories.@(js|jsx|ts|tsx)'],
  addons: [
    '@storybook/addon-essentials',
    '@storybook/addon-links',
    '@storybook/addon-actions'
  ],
  framework: {
    name: '@storybook/react',
    options: {}
  },
  docs: {
    autodocs: 'tag'
  },
  staticDirs: ['../public']
};
