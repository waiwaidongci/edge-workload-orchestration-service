CREATE TABLE IF NOT EXISTS edge_nodes (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    region TEXT NOT NULL,
    zone TEXT NOT NULL DEFAULT '',
    labels JSONB NOT NULL DEFAULT '{}'::jsonb,
    capacity JSONB NOT NULL,
    allocated JSONB NOT NULL DEFAULT '{}'::jsonb,
    max_concurrent INTEGER NOT NULL CHECK (max_concurrent > 0),
    active_tasks INTEGER NOT NULL DEFAULT 0 CHECK (active_tasks >= 0),
    status TEXT NOT NULL CHECK (status IN ('online','offline','draining','maintenance')),
    last_heartbeat TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    version BIGINT NOT NULL DEFAULT 1
);
CREATE INDEX IF NOT EXISTS idx_edge_nodes_region_status ON edge_nodes(region, status);
CREATE TABLE IF NOT EXISTS edge_capabilities (
    node_id TEXT NOT NULL REFERENCES edge_nodes(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    id TEXT NOT NULL,
    version TEXT NOT NULL,
    attributes JSONB NOT NULL DEFAULT '{}'::jsonb,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    discovered BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (node_id, name)
);
CREATE TABLE IF NOT EXISTS task_templates (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    latest_version INTEGER NOT NULL DEFAULT 0,
    versions JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE IF NOT EXISTS scheduling_policies (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    priority INTEGER NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    constraints JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE IF NOT EXISTS task_executions (
    id TEXT PRIMARY KEY,
    template_id TEXT NOT NULL REFERENCES task_templates(id),
    template_version INTEGER NOT NULL,
    policy_id TEXT NOT NULL REFERENCES scheduling_policies(id),
    node_id TEXT REFERENCES edge_nodes(id),
    priority INTEGER NOT NULL DEFAULT 0,
    labels JSONB NOT NULL DEFAULT '{}'::jsonb,
    parameters JSONB NOT NULL DEFAULT '{}'::jsonb,
    resources JSONB NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('queued','dispatched','running','succeeded','failed','cancelled')),
    attempt INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 1,
    scheduled_at TIMESTAMPTZ NOT NULL,
    lease_expires_at TIMESTAMPTZ,
    timeout_at TIMESTAMPTZ,
    cancel_requested BOOLEAN NOT NULL DEFAULT FALSE,
    failure_reason TEXT NOT NULL DEFAULT '',
    receipt_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    events JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    version BIGINT NOT NULL DEFAULT 1
);
CREATE INDEX IF NOT EXISTS idx_task_executions_queue ON task_executions(status, scheduled_at, priority DESC);
CREATE INDEX IF NOT EXISTS idx_task_executions_node ON task_executions(node_id, status);
CREATE TABLE IF NOT EXISTS node_heartbeats (
    id TEXT PRIMARY KEY,
    node_id TEXT NOT NULL REFERENCES edge_nodes(id) ON DELETE CASCADE,
    observed_at TIMESTAMPTZ NOT NULL,
    agent_uptime_seconds BIGINT NOT NULL DEFAULT 0,
    agent_version TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb
);
CREATE INDEX IF NOT EXISTS idx_node_heartbeats_node_time ON node_heartbeats(node_id, observed_at DESC);
CREATE TABLE IF NOT EXISTS dead_letters (
    id TEXT PRIMARY KEY,
    execution_id TEXT NOT NULL REFERENCES task_executions(id),
    node_id TEXT,
    reason TEXT NOT NULL,
    attempts INTEGER NOT NULL,
    resolved BOOLEAN NOT NULL DEFAULT FALSE,
    resolution TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    resolved_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_dead_letters_unresolved ON dead_letters(resolved, created_at DESC);

