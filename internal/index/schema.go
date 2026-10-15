package index

const schemaDDL = `
CREATE TABLE IF NOT EXISTS code_cache(path TEXT PRIMARY KEY, hash TEXT NOT NULL, data BLOB NOT NULL);
CREATE TABLE IF NOT EXISTS work_identities (node_id TEXT PRIMARY KEY, data BLOB NOT NULL);
CREATE TABLE IF NOT EXISTS approach_tasks (node_id TEXT PRIMARY KEY);
CREATE TABLE IF NOT EXISTS index_features (name TEXT PRIMARY KEY);
CREATE TABLE IF NOT EXISTS indexed_records (
 path TEXT PRIMARY KEY,
 kind TEXT NOT NULL,
 content_hash TEXT NOT NULL,
 node_id TEXT UNIQUE,
 data BLOB NOT NULL
);
CREATE TABLE IF NOT EXISTS record_edges (
 source_path TEXT NOT NULL,
 from_id TEXT NOT NULL,
 to_id TEXT NOT NULL,
 type TEXT NOT NULL,
 created_at TEXT NOT NULL,
 PRIMARY KEY(source_path, from_id, to_id, type)
);
CREATE INDEX IF NOT EXISTS idx_record_edges_identity ON record_edges(from_id,to_id,type);

CREATE TABLE IF NOT EXISTS nodes (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    parent_id TEXT,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL,
    confidence REAL DEFAULT 1.0,
    verification_status TEXT NOT NULL,
    source TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    file_path TEXT NOT NULL,
    content_hash TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS edges (
    from_id TEXT NOT NULL,
    to_id TEXT NOT NULL,
    type TEXT NOT NULL,
    created_at TEXT NOT NULL,
    PRIMARY KEY (from_id, to_id, type)
);

CREATE TABLE IF NOT EXISTS node_files (
    node_id TEXT NOT NULL,
    file_path TEXT NOT NULL,
    PRIMARY KEY (node_id, file_path)
);

CREATE TABLE IF NOT EXISTS node_tags (
    node_id TEXT NOT NULL,
    tag TEXT NOT NULL,
    PRIMARY KEY (node_id, tag)
);

CREATE TABLE IF NOT EXISTS node_constraints (
    node_id TEXT NOT NULL,
    constraint_text TEXT NOT NULL,
    PRIMARY KEY (node_id, constraint_text)
);

CREATE TABLE IF NOT EXISTS node_evidence (
    node_id TEXT NOT NULL,
    evidence_id TEXT NOT NULL,
    PRIMARY KEY (node_id, evidence_id)
);

CREATE TABLE IF NOT EXISTS node_failures (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    summary TEXT NOT NULL,
    causal_details TEXT,
    attempted_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_nodes_type ON nodes(type);
CREATE INDEX IF NOT EXISTS idx_nodes_status ON nodes(status);
CREATE INDEX IF NOT EXISTS idx_nodes_parent ON nodes(parent_id);
CREATE INDEX IF NOT EXISTS idx_edges_from ON edges(from_id);
CREATE INDEX IF NOT EXISTS idx_edges_to ON edges(to_id);
CREATE INDEX IF NOT EXISTS idx_edges_type ON edges(type);
CREATE INDEX IF NOT EXISTS idx_node_files_path ON node_files(file_path);
CREATE INDEX IF NOT EXISTS idx_node_tags_tag ON node_tags(tag);
CREATE INDEX IF NOT EXISTS idx_node_constraints_node ON node_constraints(node_id);
CREATE INDEX IF NOT EXISTS idx_node_evidence_node ON node_evidence(node_id);
CREATE INDEX IF NOT EXISTS idx_node_failures_task ON node_failures(task_id);
`

const dropDDL = `
DROP TABLE IF EXISTS indexed_records;
DROP TABLE IF EXISTS record_edges;
DROP TABLE IF EXISTS node_failures;
DROP TABLE IF EXISTS node_evidence;
DROP TABLE IF EXISTS node_constraints;
DROP TABLE IF EXISTS node_tags;
DROP TABLE IF EXISTS node_files;
DROP TABLE IF EXISTS edges;
DROP TABLE IF EXISTS nodes;
`

func (d *DB) initSchema() error {
	_, err := d.sqlDB.Exec(schemaDDL)
	return err
}

func (d *DB) dropSchema() error {
	_, err := d.sqlDB.Exec(dropDDL)
	return err
}
