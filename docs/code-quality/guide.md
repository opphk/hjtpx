# Code Quality Guide

## Overview

This document describes the code quality standards and practices implemented in the HJTPX project.

## Code Style

### General Principles

1. **Consistency**: Follow existing patterns in the codebase
2. **Clarity**: Write code that is easy to understand
3. **Simplicity**: Prefer simple solutions over complex ones
4. **Maintainability**: Write code that is easy to maintain and modify

### JavaScript Style Guide

- Use `const` by default, `let` when necessary
- Use arrow functions for callbacks
- Use template literals for string interpolation
- Use destructuring for objects and arrays
- Use spread operator for copying objects/arrays
- Use async/await instead of raw promises
- Use optional chaining (`?.`) when accessing nested properties
- Use nullish coalescing (`??`) for default values

### React Style Guide

- Use functional components with hooks
- Use PascalCase for component names
- Use camelCase for props and methods
- Keep components small and focused
- Extract reusable logic into custom hooks
- Use React.memo for performance optimization
- Use useCallback and useMemo appropriately

## Linting

### ESLint Configuration

The project uses ESLint with the following plugins:
- `react`: React-specific rules
- `react-hooks`: React Hooks rules
- `jsx-a11y`: Accessibility rules
- `import`: Import/export rules
- `prettier`: Prettier integration

### Running ESLint

```bash
npm run lint
```

### Fixing Lint Errors

```bash
npm run lint -- --fix
```

### ESLint Rules

| Rule | Severity | Description |
|------|----------|-------------|
| `prettier/prettier` | error | Enforce Prettier formatting |
| `no-unused-vars` | warn | Warn about unused variables |
| `no-console` | warn | Warn about console statements |
| `react/prop-types` | warn | Require prop types |
| `import/order` | error | Enforce import order |

## Formatting

### Prettier Configuration

The project uses Prettier with the following settings:
- Single quotes for strings
- Semicolons at the end of statements
- Trailing commas: none
- Print width: 100
- Tab width: 2 spaces
- Arrow functions: avoid parens

### Running Prettier

Format all files:
```bash
npm run format
```

Check formatting without modifying:
```bash
npm run format -- --check
```

### VS Code Integration

Install the following extensions:
- ESLint (dbaeumer.vscode-eslint)
- Prettier (esbenp.prettier-vscode)
- EditorConfig (editorconfig.editorconfig)

The workspace settings will automatically format on save.

## Git Hooks

### Husky

The project uses Husky for Git hooks.

### Pre-commit Hook

Runs before each commit:
1. Lint-staged checks
2. ESLint on staged files
3. Prettier formatting

### Commit-msg Hook

Validates commit messages follow conventional commits format:
```
<type>(<scope>): <description>

Types:
- build, chore, ci, docs, feat, fix, perf, refactor, test, style, workflow
```

### Installing Husky

```bash
npm install husky lint-staged -D
npx husky install
```

### Skipping Hooks

To skip hooks during commit:
```bash
git commit --no-verify -m "message"
```

## Conventional Commits

### Format

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

### Types

| Type | Description |
|------|-------------|
| build | Build system changes |
| chore | Maintenance tasks |
| ci | CI/CD changes |
| docs | Documentation |
| feat | New feature |
| fix | Bug fix |
| perf | Performance improvement |
| refactor | Code refactoring |
| style | Code style changes |
| test | Test changes |
| workflow | Workflow changes |

### Examples

```
feat(auth): add password reset functionality
fix(api): resolve user lookup performance issue
docs(readme): update installation instructions
refactor(users): extract validation logic
```

## Testing Requirements

### Test Coverage

- Minimum 80% coverage for new code
- All critical paths must be tested
- Edge cases should be covered

### Test Organization

```
tests/
├── unit/           # Unit tests
├── integration/    # Integration tests
├── api/           # API tests
└── e2e/           # End-to-end tests
```

### Writing Tests

1. Use descriptive test names
2. Follow AAA pattern (Arrange, Act, Assert)
3. Mock external dependencies
4. Test behavior, not implementation

## Code Review

### Checklist

- [ ] Code follows style guide
- [ ] All tests pass
- [ ] No lint errors
- [ ] Proper error handling
- [ ] Security considerations
- [ ] Performance implications
- [ ] Documentation updated
- [ ] No commented-out code
- [ ] No debug statements
- [ ] Proper logging

### Review Process

1. Create feature branch
2. Make changes
3. Run tests locally
4. Submit pull request
5. Address review comments
6. Merge after approval

## Performance

### Best Practices

1. **Avoid Premature Optimization**
   - Profile before optimizing
   - Focus on algorithmic complexity

2. **Memory Management**
   - Avoid memory leaks
   - Clean up event listeners
   - Use appropriate data structures

3. **Database Queries**
   - Use indexes appropriately
   - Avoid N+1 queries
   - Optimize slow queries

4. **React Performance**
   - Use React.memo for pure components
   - Use useMemo and useCallback appropriately
   - Implement virtualization for long lists
   - Lazy load components and routes

## Security

### Best Practices

1. **Input Validation**
   - Validate all user input
   - Use parameterized queries
   - Sanitize HTML output

2. **Authentication & Authorization**
   - Use secure token generation
   - Implement proper session management
   - Check permissions on every request

3. **Secrets Management**
   - Never commit secrets
   - Use environment variables
   - Rotate secrets regularly

4. **Dependencies**
   - Keep dependencies updated
   - Audit for vulnerabilities
   - Remove unused dependencies

## Documentation

### When to Document

- Public APIs and interfaces
- Complex algorithms
- Non-obvious code
- Configuration options
- Known limitations

### Documentation Format

```javascript
/**
 * Calculates the sum of two numbers
 *
 * @param {number} a - First number
 * @param {number} b - Second number
 * @returns {number} The sum of a and b
 *
 * @example
 * sum(1, 2) // returns 3
 */
function sum(a, b) {
  return a + b;
}
```

## Tools and Commands

### Code Quality Commands

```bash
# Run all quality checks
npm run lint
npm run format

# Run tests
npm test

# Run with coverage
npm run test:coverage

# Audit dependencies
npm audit

# Security scan
npx snyk test
```

### IDE Setup

Install recommended VS Code extensions:
1. ESLint
2. Prettier
3. EditorConfig
4. GitLens
5. Jest Runner

## Continuous Integration

The CI pipeline runs the following checks:
1. ESLint validation
2. Prettier formatting check
3. Unit tests
4. Integration tests
5. Security audit
6. Build verification

All checks must pass before merging.

## Resources

- [JavaScript Style Guide](https://github.com/airbnb/javascript)
- [React Style Guide](https://github.com/airbnb/javascript/tree/master/react)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [ESLint Rules](https://eslint.org/docs/rules/)
- [Prettier Options](https://prettier.io/docs/en/options.html)
