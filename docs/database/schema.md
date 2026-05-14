# Database Schema Documentation

## Overview

This document describes the database schema for the HJTPX project, including table structures, field definitions, indexes, and relationships.

## Database Type

- **Database Engine**: PostgreSQL
- **Version**: 12+
- **Encoding**: UTF8

---

## Tables

### 1. users

The `users` table stores user account information and authentication details.

#### Table Structure

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY, DEFAULT gen_random_uuid() | Unique identifier for the user |
| email | VARCHAR(255) | UNIQUE, NOT NULL | User's email address |
| name | VARCHAR(255) | NOT NULL | User's display name |
| password | VARCHAR(255) | NOT NULL | Hashed password |
| role | user_role | DEFAULT 'user' | User's role (admin, moderator, user, guest) |
| is_active | BOOLEAN | DEFAULT true | Whether the user account is active |
| last_login | TIMESTAMP | NULLABLE | Timestamp of last login |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Account creation timestamp |
| updated_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Last update timestamp |

#### Fields Description

- **id**: UUID v4 generated unique identifier
- **email**: User's email address, must be unique across the system
- **name**: Display name shown in the UI
- **password**: Bcrypt hashed password (cost factor 10)
- **role**: User role enum for access control
- **is_active**: Soft delete flag, inactive users cannot log in
- **last_login**: Tracks user activity for analytics
- **created_at**: Immutable timestamp when account was created
- **updated_at**: Auto-updated timestamp on any record modification

#### Role Values

The `user_role` enum type has the following values:

| Value | Description |
|-------|-------------|
| admin | Full system access |
| moderator | Elevated permissions for content management |
| user | Standard user with basic access |
| guest | Limited read-only access |

---

### 2. sessions

The `sessions` table manages user authentication sessions and tokens.

#### Table Structure

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY, DEFAULT gen_random_uuid() | Unique session identifier |
| user_id | UUID | FOREIGN KEY REFERENCES users(id) ON DELETE CASCADE | Associated user |
| token | VARCHAR(500) | NOT NULL | JWT or session token |
| expires_at | TIMESTAMP | NOT NULL | Session expiration time |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Session creation timestamp |

#### Fields Description

- **id**: UUID v4 generated unique identifier
- **user_id**: Foreign key linking to the users table, cascade deletes sessions when user is deleted
- **token**: JWT or session token string
- **expires_at**: Timestamp when the session expires
- **created_at**: Immutable timestamp when session was created

---

### 3. migrations

The `migrations` table tracks database schema changes.

#### Table Structure

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | SERIAL | PRIMARY KEY | Auto-incrementing ID |
| name | VARCHAR(255) | UNIQUE, NOT NULL | Migration filename |
| applied_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | When migration was applied |

---

## Indexes

### Users Table Indexes

| Index Name | Columns | Type | Description |
|------------|---------|------|-------------|
| idx_users_email | email | B-tree | Fast email lookups for authentication |
| idx_users_role | role | B-tree | Efficient role-based queries |
| idx_users_created_at | created_at DESC | B-tree | Sorted queries by creation date |
| idx_users_is_active | is_active | B-tree | Filter active/inactive users |
| idx_users_email_role | email, role | B-tree | Composite index for auth with role check |
| idx_users_is_active_role | is_active, role | B-tree | Composite index for user filtering |

### Sessions Table Indexes

| Index Name | Columns | Type | Description |
|------------|---------|------|-------------|
| idx_sessions_user_id | user_id | B-tree | Fast user session lookups |
| idx_sessions_token | token | B-tree | Fast token validation lookups |
| idx_sessions_expires_at | expires_at | B-tree | Session cleanup queries |
| idx_sessions_active | user_id, expires_at | B-tree | Partial index for active sessions |
| idx_sessions_token_expires | token, expires_at | B-tree | Composite index for token validation |

---

## Triggers

### update_users_updated_at

Automatically updates the `updated_at` column when a row is modified.

```sql
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

---

## Relationships

```
┌─────────────────┐
│     users       │
├─────────────────┤
│ id (PK)         │
│ email           │
│ name            │
│ password        │
│ role            │
│ is_active       │
│ last_login      │
│ created_at      │
│ updated_at      │
└────────┬────────┘
         │
         │ 1:N
         ▼
┌─────────────────┐
│   sessions      │
├─────────────────┤
│ id (PK)         │
│ user_id (FK)────┼──► users(id)
│ token           │
│ expires_at      │
│ created_at      │
└─────────────────┘
```

---

## Query Examples

### Find user by email

```sql
SELECT * FROM users WHERE email = 'user@example.com';
```

### Get user's active sessions

```sql
SELECT * FROM sessions
WHERE user_id = $1
AND expires_at > CURRENT_TIMESTAMP
ORDER BY created_at DESC;
```

### Clean up expired sessions

```sql
DELETE FROM sessions WHERE expires_at < CURRENT_TIMESTAMP;
```

### Get user count by role

```sql
SELECT role, COUNT(*) as count
FROM users
WHERE is_active = true
GROUP BY role;
```

---

## Performance Considerations

### Index Strategy

1. **Primary Key Indexes**: Automatically created on primary key columns
2. **Unique Indexes**: Created for email to ensure uniqueness
3. **Foreign Key Indexes**: Created on user_id for session lookups
4. **Composite Indexes**: Used for multi-column queries like (email, role)

### Query Optimization

1. Use `SELECT` only required columns instead of `SELECT *`
2. Always filter by indexed columns when possible
3. Use `LIMIT` for large result sets
4. Consider partial indexes for frequently filtered conditions
5. Regular `ANALYZE` operations to update statistics

### Caching Strategy

Implement query caching for:
- Frequently accessed reference data
- User profile information
- Aggregation results
- Session validation (with short TTL)

---

## Security Considerations

1. **Password Storage**: Passwords are hashed using bcrypt with cost factor 10
2. **Email Uniqueness**: Enforced at database level
3. **Session Tokens**: Stored as VARCHAR, should be JWT or secure random tokens
4. **Soft Delete**: Use `is_active` flag instead of hard deletes for audit purposes
5. **Cascade Deletes**: Sessions are automatically deleted when user is removed

---

## Migration Files

| File | Description |
|------|-------------|
| 001_initial_schema.sql | Creates initial users and sessions tables |
| 002_add_roles.sql | Adds role management and test users |
| 003_create_indexes.sql | Creates additional performance indexes |

---

## Version History

| Date | Version | Changes |
|------|---------|---------|
| 2024-01-01 | 1.0 | Initial schema with users and sessions |
| 2024-01-02 | 1.1 | Added role management and user status |
| 2024-01-03 | 1.2 | Added performance indexes |
