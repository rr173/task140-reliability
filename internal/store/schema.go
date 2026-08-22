package store

// schema is the authoritative DDL for the reliability engine. It is applied
// idempotently on Open. Derived result rows (analysis_results) are recomputable
// from the authoritative input tables (fta_*, rbd_blocks, fmea_rows,
// failure_events); ReconcileAll re-Solves and overwrites them.
const schema = `
CREATE TABLE IF NOT EXISTS schema_version (
  version INTEGER PRIMARY KEY
);
INSERT INTO schema_version(version) SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM schema_version);

CREATE TABLE IF NOT EXISTS components (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  category TEXT NOT NULL DEFAULT '',
  failure_rate_ppt INTEGER NOT NULL DEFAULT 0,
  note TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS failure_modes (
  id TEXT PRIMARY KEY,
  component_id TEXT NOT NULL,
  name TEXT NOT NULL,
  effect TEXT NOT NULL DEFAULT '',
  severity INTEGER NOT NULL,
  occurrence INTEGER NOT NULL,
  detection INTEGER NOT NULL,
  FOREIGN KEY(component_id) REFERENCES components(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS analyses (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  top_event TEXT NOT NULL DEFAULT '',
  state TEXT NOT NULL DEFAULT 'draft',
  version INTEGER NOT NULL DEFAULT 1,
  max_order INTEGER NOT NULL DEFAULT 4,
  s_crit INTEGER NOT NULL DEFAULT 8,
  rpn_high INTEGER NOT NULL DEFAULT 200,
  rpn_medium INTEGER NOT NULL DEFAULT 100,
  exact_limit INTEGER NOT NULL DEFAULT 8,
  ccf_beta REAL NOT NULL DEFAULT 0.1,
  mission_hours INTEGER NOT NULL DEFAULT 8760,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  baselined_at TEXT
);

CREATE TABLE IF NOT EXISTS fta_gates (
  analysis_id TEXT NOT NULL,
  id TEXT NOT NULL,
  type TEXT NOT NULL,
  k INTEGER NOT NULL DEFAULT 0,
  is_top INTEGER NOT NULL DEFAULT 0,
  seq INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (analysis_id, id),
  FOREIGN KEY(analysis_id) REFERENCES analyses(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_fta_gates_aid ON fta_gates(analysis_id);

CREATE TABLE IF NOT EXISTS fta_gate_inputs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  analysis_id TEXT NOT NULL,
  gate_id TEXT NOT NULL,
  seq INTEGER NOT NULL,
  child_gate_id TEXT,
  event_id TEXT,
  FOREIGN KEY(analysis_id, gate_id) REFERENCES fta_gates(analysis_id, id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_gate_inputs_gate ON fta_gate_inputs(analysis_id, gate_id);

CREATE TABLE IF NOT EXISTS fta_basic_events (
  analysis_id TEXT NOT NULL,
  id TEXT NOT NULL,
  label TEXT NOT NULL DEFAULT '',
  component_id TEXT,
  prob_micro INTEGER NOT NULL DEFAULT 0,
  ccf_group TEXT NOT NULL DEFAULT '',
  negated INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (analysis_id, id),
  FOREIGN KEY(analysis_id) REFERENCES analyses(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_fta_events_aid ON fta_basic_events(analysis_id);

CREATE TABLE IF NOT EXISTS fmea_rows (
  id TEXT PRIMARY KEY,
  analysis_id TEXT NOT NULL,
  function TEXT NOT NULL,
  failure_mode TEXT NOT NULL,
  effect TEXT NOT NULL DEFAULT '',
  severity INTEGER NOT NULL,
  occurrence INTEGER NOT NULL,
  detection INTEGER NOT NULL,
  rpn INTEGER NOT NULL DEFAULT 0,
  risk_class TEXT NOT NULL DEFAULT '',
  action TEXT NOT NULL DEFAULT '',
  action_state TEXT NOT NULL DEFAULT 'open',
  created_at TEXT NOT NULL,
  FOREIGN KEY(analysis_id) REFERENCES analyses(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_fmea_aid ON fmea_rows(analysis_id);

CREATE TABLE IF NOT EXISTS rbd_blocks (
  analysis_id TEXT NOT NULL,
  id TEXT NOT NULL,
  name TEXT NOT NULL DEFAULT '',
  type TEXT NOT NULL,
  k INTEGER NOT NULL DEFAULT 0,
  failure_rate_ppt INTEGER NOT NULL DEFAULT 0,
  repair_rate_ppt INTEGER NOT NULL DEFAULT 0,
  is_root INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (analysis_id, id),
  FOREIGN KEY(analysis_id) REFERENCES analyses(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_rbd_aid ON rbd_blocks(analysis_id);

CREATE TABLE IF NOT EXISTS rbd_children (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  analysis_id TEXT NOT NULL,
  block_id TEXT NOT NULL,
  seq INTEGER NOT NULL,
  child_id TEXT NOT NULL,
  FOREIGN KEY(analysis_id, block_id) REFERENCES rbd_blocks(analysis_id, id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_rbd_children_block ON rbd_children(analysis_id, block_id);

CREATE TABLE IF NOT EXISTS failure_events (
  id TEXT PRIMARY KEY,
  analysis_id TEXT NOT NULL,
  component_id TEXT,
  ttf_hours INTEGER NOT NULL,
  ttr_hours INTEGER NOT NULL DEFAULT 0,
  occurred_at TEXT NOT NULL,
  FOREIGN KEY(analysis_id) REFERENCES analyses(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_fe_aid ON failure_events(analysis_id);

CREATE TABLE IF NOT EXISTS analysis_results (
  analysis_id TEXT NOT NULL,
  version INTEGER NOT NULL,
  solved_at TEXT NOT NULL,
  result_json TEXT NOT NULL,
  compliant INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (analysis_id, version),
  FOREIGN KEY(analysis_id) REFERENCES analyses(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS revisions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  analysis_id TEXT NOT NULL,
  version INTEGER NOT NULL,
  state TEXT NOT NULL,
  created_at TEXT NOT NULL,
  note TEXT NOT NULL DEFAULT '',
  UNIQUE(analysis_id, version),
  FOREIGN KEY(analysis_id) REFERENCES analyses(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_revisions_aid ON revisions(analysis_id);
`
