# Database Migrations Guide

## Overview

This project uses a custom migration management system to handle database schema changes. The migration system provides features like rollback support, performance tracking, checksum validation, and batch execution.

## Migration Structure

### Directory Structure

```
migrations/
├── 001_initial_schema.sql
├── 002_add_roles.sql
├── ...
└── rollbacks/
    ├── 001_initial_schema_rollback.sql
    ├── 002_add_roles_rollback.sql
    └── ...
```

## Usage

### Running Migrations

```bash
# Apply all pending migrations
node scripts/migrate.js up

# Apply with rollback script generation
node scripts/migrate.js up --generate-rollback

# Dry run (preview only)
node scripts/migrate.js up --dry-run
```

### Rolling Back Migrations

```bash
# Rollback last migration
node scripts/migrate.js down

# Rollback to specific migration
node scripts/migrate.js down --to 001_initial_schema.sql

# Force rollback without rollback script
node scripts/migrate.js down --force
```

### Other Commands

```bash
# Check migration status
node scripts/migrate.js status

# Detailed status with history
node scripts/migrate.js status --verbose

# Redo last migration (rollback + apply)
node scripts/migrate.js redo

# Reset all migrations
node scripts/migrate.js reset

# Create new migration
node scripts/migrate.js create add_new_table

# Create migration with rollback template
node scripts/migrate.js create add_new_table --generate-rollback
```

## Migration Features

### 1. Performance Tracking

The system tracks:
- Migration duration
- Success/failure status
- Batch execution
- Average execution time

### 2. Checksum Validation

Each migration is validated using SHA-256 checksum to ensure integrity.

### 3. Batch Execution

Migrations are grouped into batches for better organization and rollback management.

### 4. Rollback Support

- **Automatic Generation**: Use `--generate-rollback` to auto-generate rollback scripts
- **Manual Rollbacks**: Rollback scripts in `migrations/rollbacks/` directory
- **Selective Rollback**: Rollback to specific migration with `--to` flag

### 5. Safety Features

- SQL validation for dangerous operations
- Transaction support (BEGIN/COMMIT)
- Rollback on error
- Force option for skipping safety checks

## Migration Format

### Naming Convention

```
YYYYMMDDHHMMSS_description.sql
Example: 20260101120000_add_user_profiles.sql
```

### Template

```sql
-- Migration: description
-- Created: YYYY-MM-DDTHH:mm:ss.sssZ
-- Description: What this migration does

BEGIN;

-- SQL statements here

COMMIT;
```

## Best Practices

1. **Always use transactions** - Wrap migrations in BEGIN/COMMIT
2. **Test rollbacks** - Ensure rollback scripts work correctly
3. **Use meaningful names** - Describe what the migration does
4. **Small, focused changes** - One logical change per migration
5. **Check dependencies** - Consider foreign key constraints
6. **Use IF NOT EXISTS** - For tables, indexes, and columns
7. **Document data changes** - Include comments for complex operations

## Rollback Script Format

```sql
-- Rollback for: migration_name.sql
-- Generated: YYYY-MM-DDTHH:mm:ss.sssZ
-- This script reverts the changes made by migration_name.sql

BEGIN;

-- Rollback statements here

COMMIT;
```

## Environment Variables

```bash
DB_HOST=localhost
DB_PORT=5432
DB_NAME=hjtpx
DB_USER=postgres
DB_PASSWORD=postgres
DB_MAX_CONNECTIONS=20
DB_IDLE_TIMEOUT=30000
DB_CONNECTION_TIMEOUT=2000
```

## Troubleshooting

### Migration Stuck

If a migration is stuck, check for:
- Uncommitted transactions
- Locked tables
- Connection issues

### Rollback Fails

Common causes:
- Missing rollback script
- Data dependencies
- Schema changes from other sources

### Checksum Mismatch

If checksum validation fails:
- Migration file may have been modified
- Use `--force` to skip validation

## API Integration

The migration system can be integrated with API endpoints:

```javascript
const { migrate, status } = require('./scripts/migrate');

// Run migrations programmatically
const result = await migrate({ rollback: false });
console.log(result);

// Get status
const status = await status({ verbose: true });
console.log(status);
```

## Monitoring

Check migration performance metrics:

```bash
node scripts/migrate.js status --verbose
```

Output includes:
- Total migrations
- Success rate
- Average duration
- Batch information
