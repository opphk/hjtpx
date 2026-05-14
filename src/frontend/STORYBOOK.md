# Component Documentation Guide

## Overview

This project uses Storybook for component documentation and development environment.

## Installation

To install and set up Storybook in this project:

```bash
cd src/frontend
npm install --save-dev @storybook/react @storybook/react-vite @storybook/addon-essentials @storybook/addon-a11y @storybook/addon-docs @storybook/addon-controls @storybook/addon-actions @storybook/addon-jest @storybook/blocks @storybook/testing-library @storybook/jest @vitejs/plugin-react
npx storybook@latest init
```

## Running Storybook

Start the Storybook development server:

```bash
cd src/frontend
npm run storybook
```

## Build Static Site

Build a static Storybook site for deployment:

```bash
npm run build-storybook
```

## Story Structure

Stories are located in the `stories/` directory:

```
src/frontend/
├── stories/
│   ├── Button.stories.jsx
│   ├── Input.stories.jsx
│   ├── Alert.stories.jsx
│   ├── Loading.stories.jsx
│   └── Modal.stories.jsx
├── components/
│   ├── Button.jsx
│   ├── Input.jsx
│   ├── Alert.jsx
│   ├── Loading.jsx
│   └── Modal.jsx
└── .storybook/
    ├── main.js
    └── preview.js
```

## Writing Stories

Each component should have a corresponding story file with:

- **Default export**: Component metadata (title, component, parameters)
- **Named exports**: Individual stories with different states

Example:

```javascript
import React from 'react';
import Button from '../components/Button';

export default {
  title: 'Components/Button',
  component: Button,
};

export const Primary = {
  args: {
    variant: 'primary',
    children: 'Primary Button',
  },
};
```

## Args & Controls

Use argTypes to define controls:

```javascript
argTypes: {
  variant: {
    control: { type: 'select' },
    options: ['primary', 'secondary'],
  },
  disabled: {
    control: 'boolean',
  },
}
```

## Props Documentation

Storybook automatically generates props tables from component definitions when using `@storybook/addon-docs`.

## Deployment

Storybook can be deployed to GitHub Pages, Netlify, or any static hosting:

```bash
npm run build-storybook
# Upload the storybook-static directory
```

## Addons

The following addons are configured:

- **Essentials**: Controls, actions, docs, viewport, backgrounds
- **A11y**: Accessibility testing
- **Controls**: Interactive props editing
- **Actions**: Action logging
- **Jest**: Jest test results

## Best Practices

1. Write stories for all component states (default, hover, disabled, error, etc.)
2. Use args to make stories interactive
3. Document props with descriptions
4. Create compound stories showing component combinations
5. Test accessibility with the a11y addon
