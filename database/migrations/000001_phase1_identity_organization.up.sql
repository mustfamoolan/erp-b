-- Phase 1: Organization & Identity Foundation
-- Roadmap §12 — PHASE 1: DOMAIN & ORGANIZATION FOUNDATION
--
-- Rules enforced:
-- Rule 5:  Every operational entity must have clear scope ownership
-- Rule 6:  Administration and Factory data must remain distinguishable
-- Rule 7:  Factory A must never automatically access Factory B
-- Rule 8:  Security must be enforced by the Go backend
-- Rule 19: Use database constraints for critical data integrity

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- 1. ORGANIZATION SCOPES
-- Every record in the system must trace back to a scope.
-- Roadmap §5 — ORGANIZATIONAL MODEL
-- ============================================================
CREATE TABLE organization_scopes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type        VARCHAR(20)  NOT NULL CHECK (type IN ('ADMINISTRATION', 'FACTORY')),
    name        VARCHAR(150) NOT NULL,
    code        VARCHAR(50)  NOT NULL,
    parent_id   UUID         REFERENCES organization_scopes(id) ON DELETE RESTRICT,
    status      VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'INACTIVE')),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT organization_scopes_name_unique UNIQUE (name),
    CONSTRAINT organization_scopes_code_unique UNIQUE (code)
);

CREATE INDEX idx_organization_scopes_type   ON organization_scopes(type);
CREATE INDEX idx_organization_scopes_status ON organization_scopes(status);

COMMENT ON TABLE  organization_scopes         IS 'Organizational units: Administration or Factory. Every operational record must trace to a scope.';
COMMENT ON COLUMN organization_scopes.type    IS 'ADMINISTRATION or FACTORY';
COMMENT ON COLUMN organization_scopes.code    IS 'Short identifier, e.g. ADM, FAC-A, FAC-B';
COMMENT ON COLUMN organization_scopes.parent_id IS 'For future hierarchical factory/branch relationships';

-- ============================================================
-- 2. EMPLOYEES
-- Employees belong to one scope. Roadmap §5
-- ============================================================
CREATE TABLE employees (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scope_id     UUID         NOT NULL REFERENCES organization_scopes(id) ON DELETE RESTRICT,
    employee_no  VARCHAR(50)  NOT NULL,
    full_name    VARCHAR(200) NOT NULL,
    national_id  VARCHAR(50),
    phone        VARCHAR(30),
    email        VARCHAR(255),
    job_title    VARCHAR(150),
    hire_date    DATE         NOT NULL,
    status       VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'INACTIVE', 'TERMINATED')),
    notes        TEXT,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT employees_no_unique       UNIQUE (employee_no),
    CONSTRAINT employees_national_unique UNIQUE (national_id)
);

CREATE INDEX idx_employees_scope_id ON employees(scope_id);
CREATE INDEX idx_employees_status   ON employees(status);

-- ============================================================
-- 3. ROLES
-- Roadmap §9 — named collection of permissions.
-- ============================================================
CREATE TABLE roles (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name         VARCHAR(50)  NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    description  TEXT,
    is_system    BOOLEAN      NOT NULL DEFAULT FALSE, -- system roles cannot be deleted
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT roles_name_unique UNIQUE (name)
);

-- ============================================================
-- 4. PERMISSIONS
-- Roadmap §10 — granular permissions. Format: resource.action
-- ============================================================
CREATE TABLE permissions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code         VARCHAR(100) NOT NULL,
    display_name VARCHAR(150) NOT NULL,
    description  TEXT,
    grp          VARCHAR(50)  NOT NULL, -- group: request, cashbox, warehouse, accounting, etc.
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT permissions_code_unique UNIQUE (code)
);

CREATE INDEX idx_permissions_grp ON permissions(grp);

-- ============================================================
-- 5. ROLE PERMISSIONS
-- ============================================================
CREATE TABLE role_permissions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id       UUID NOT NULL REFERENCES roles(id)       ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT role_permissions_unique UNIQUE (role_id, permission_id)
);

CREATE INDEX idx_role_permissions_role_id       ON role_permissions(role_id);
CREATE INDEX idx_role_permissions_permission_id ON role_permissions(permission_id);

-- ============================================================
-- 6. USERS
-- Authenticated identities. Roadmap §8
-- ============================================================
CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username      VARCHAR(100) NOT NULL,
    email         VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name     VARCHAR(200) NOT NULL,
    status        VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'INACTIVE', 'SUSPENDED')),
    employee_id   UUID         REFERENCES employees(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT users_username_unique UNIQUE (username),
    CONSTRAINT users_email_unique    UNIQUE (email)
);

CREATE INDEX idx_users_status      ON users(status);
CREATE INDEX idx_users_employee_id ON users(employee_id);

-- ============================================================
-- 7. USER ROLES
-- A user may have multiple roles, each optionally scoped.
-- Roadmap §9 — Use User + Role + Permission + Scope together.
-- ============================================================
CREATE TABLE user_roles (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id)               ON DELETE CASCADE,
    role_id    UUID        NOT NULL REFERENCES roles(id)               ON DELETE CASCADE,
    scope_id   UUID        REFERENCES organization_scopes(id)          ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- A user cannot have the same role for the same scope twice
    CONSTRAINT user_roles_unique UNIQUE (user_id, role_id, scope_id)
);

CREATE INDEX idx_user_roles_user_id  ON user_roles(user_id);
CREATE INDEX idx_user_roles_role_id  ON user_roles(role_id);
CREATE INDEX idx_user_roles_scope_id ON user_roles(scope_id);

-- ============================================================
-- 8. USER SCOPE ACCESS
-- Explicit scope access grants. Factory A user cannot access Factory B.
-- Roadmap §2.2, Rule 7
-- ============================================================
CREATE TABLE user_scope_access (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id)              ON DELETE CASCADE,
    scope_id    UUID        NOT NULL REFERENCES organization_scopes(id) ON DELETE CASCADE,
    granted_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    granted_by  UUID        NOT NULL REFERENCES users(id)              ON DELETE RESTRICT,

    CONSTRAINT user_scope_access_unique UNIQUE (user_id, scope_id)
);

CREATE INDEX idx_user_scope_access_user_id  ON user_scope_access(user_id);
CREATE INDEX idx_user_scope_access_scope_id ON user_scope_access(scope_id);

-- ============================================================
-- 9. SEED: System Roles (Roadmap §9)
-- ============================================================
INSERT INTO roles (name, display_name, description, is_system) VALUES
    ('SUPER_ADMIN',        'Super Administrator', 'Full system access',            TRUE),
    ('ADMINISTRATOR',      'Administrator',       'Company administration access', TRUE),
    ('AUDITOR',            'Auditor',             'Read-only audit access',        TRUE),
    ('ACCOUNTANT',         'Accountant',          'Administration accounting',     TRUE),
    ('FACTORY_ACCOUNTANT', 'Factory Accountant',  'Factory-level accounting',      TRUE),
    ('FACTORY_EMPLOYEE',   'Factory Employee',    'Factory operations',            TRUE);

-- ============================================================
-- 10. SEED: All Permissions (Roadmap §10)
-- ============================================================
INSERT INTO permissions (code, display_name, grp) VALUES
    -- Requests
    ('request.create',  'Create Request',  'request'),
    ('request.view',    'View Request',    'request'),
    ('request.review',  'Review Request',  'request'),
    ('request.approve', 'Approve Request', 'request'),
    ('request.reject',  'Reject Request',  'request'),
    ('request.pay',     'Pay Request',     'request'),
    ('request.receive', 'Receive Request', 'request'),
    -- Cashbox
    ('cashbox.view',     'View Cashbox',     'cashbox'),
    ('cashbox.transfer', 'Transfer Cashbox', 'cashbox'),
    -- Warehouse
    ('warehouse.view',     'View Warehouse',     'warehouse'),
    ('warehouse.receive',  'Receive Warehouse',  'warehouse'),
    ('warehouse.issue',    'Issue Warehouse',    'warehouse'),
    ('warehouse.transfer', 'Transfer Warehouse', 'warehouse'),
    ('warehouse.adjust',   'Adjust Warehouse',   'warehouse'),
    -- Accounting
    ('accounting.view',         'View Accounting',         'accounting'),
    ('accounting.post',         'Post Journal Entry',      'accounting'),
    ('accounting.reverse',      'Reverse Journal Entry',   'accounting'),
    ('accounting.close_period', 'Close Accounting Period', 'accounting'),
    -- Audit & Reports
    ('audit.view',   'View Audit Log', 'audit'),
    ('reports.view', 'View Reports',   'reports'),
    -- Administration
    ('users.manage',     'Manage Users',     'admin'),
    ('roles.manage',     'Manage Roles',     'admin'),
    ('factories.manage', 'Manage Factories', 'admin'),
    ('employees.manage', 'Manage Employees', 'admin');

-- ============================================================
-- 11. SEED: Initial Administration Scope
-- Every system must have exactly one Administration scope.
-- ============================================================
INSERT INTO organization_scopes (type, name, code, status) VALUES
    ('ADMINISTRATION', 'الإدارة العامة', 'ADM', 'ACTIVE');
