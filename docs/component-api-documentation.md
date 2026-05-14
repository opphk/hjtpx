# Component & API Documentation

This project includes comprehensive documentation tools for both frontend components and backend API.

## Frontend Component Documentation (Storybook)

### Setup

Storybook is already configured in the frontend project. To install dependencies:

```bash
cd src/frontend
npm install
```

### Running Storybook

Start the Storybook development server:

```bash
npm run storybook
```

This will start Storybook at `http://localhost:6006`.

### Building Storybook

Build the static Storybook site:

```bash
npm run build-storybook
```

### Available Components

The following components are documented:

- **Button** - Versatile button component with multiple variants and sizes
- **Input** - Form input component with validation and error handling
- **Alert** - Notification and alert component
- **Loading** - Loading spinner and indicator
- **Modal** - Dialog and modal component

### Props Documentation

Each component story includes:

- Full prop documentation with types
- Default values
- Interactive controls in the Controls tab
- Multiple examples for different use cases

## Backend API Documentation (Swagger)

### Swagger Setup

Swagger is already configured in the backend project. The API documentation is available at:

- UI: `/api-docs`
- JSON: `/api-docs/json`
- YAML: `/api-docs/yaml`

### API Documentation Features

#### 1. Automatic Documentation Generation

The API documentation is automatically generated from JSDoc comments and route definitions. To update documentation:

```bash
POST /documentation/update
```

#### 2. API Change Detection

Track changes between API versions:

```bash
GET /documentation/diff
```

This detects:
- Added endpoints
- Removed endpoints
- Modified endpoints
- Breaking changes

#### 3. Version Management

Manage API versions:

```bash
GET  /documentation/versions           # List all versions
POST /documentation/versions/:version  # Create new version
GET  /documentation/versions/:version  # Get specific version
POST /documentation/versions/:v1/compare/:v2  # Compare versions
POST /documentation/versions/:version/deprecate # Deprecate version
```

#### 4. Usage Statistics

Track API usage:

```bash
GET /documentation/usage/stats   # Get usage statistics
GET /documentation/usage/report  # Get daily report
GET /documentation/coverage      # Get documentation coverage
```

### Documentation CI/CD

A GitHub Actions workflow is configured to:

1. Validate OpenAPI specification on every push
2. Check for breaking changes
3. Generate documentation coverage reports
4. Export documentation in JSON and YAML formats
5. Comment on PRs with documentation changes

See `.github/workflows/api-docs-ci.yml` for configuration.

### Documentation Best Practices

1. **Always document new endpoints** - Add summary and description to all routes
2. **Use proper HTTP methods** - GET, POST, PUT, DELETE, PATCH
3. **Define request/response schemas** - Use components/schemas
4. **Add examples** - Include request/response examples
5. **Mark deprecated endpoints** - Use the `deprecated: true` flag

### Example Route Documentation

```javascript
/**
 * @swagger
 * /api/users:
 *   get:
 *     summary: Get all users
 *     description: Retrieve a list of all registered users
 *     tags:
 *       - Users
 *     parameters:
 *       - name: page
 *         in: query
 *         schema:
 *           type: integer
 *         description: Page number
 *     responses:
 *       200:
 *         description: Successful response
 *         content:
 *           application/json:
 *             schema:
 *               type: array
 *               items:
 *                 $ref: '#/components/schemas/User'
 */
router.get('/users', async (req, res) => {
  // Route implementation
});
```

## Deployment

### Storybook Deployment

Storybook is automatically deployed to GitHub Pages via the `storybook-deploy.yml` workflow when changes are pushed to main or develop branches.

### API Documentation

API documentation is available in the running application at `/api-docs`. The CI workflow also exports documentation to the `docs/api/` directory.

## Additional Resources

- [Storybook Documentation](https://storybook.js.org/)
- [OpenAPI Specification](https://swagger.io/specification/)
- [Swagger JSdoc](https://github.com/Surnet/swagger-jsdoc)
