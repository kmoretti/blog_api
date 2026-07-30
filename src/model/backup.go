package model

// ImportResult holds per-module import statistics.
type ImportResult struct {
	Imported int `json:"imported"`
	Skipped  int `json:"skipped"`
	Replaced int `json:"replaced"`
}

// RestoreResult reports the outcome of a full backup restore.
type RestoreResult struct {
	BackupPath string `json:"backup_path"`
	Notice     string `json:"notice"`
}

// ExportEnvelope is the canonical JSON wrapper for module exports.
type ExportEnvelope struct {
	Version    string      `json:"version"`
	Module     string      `json:"module"`
	ExportedAt string      `json:"exported_at"`
	Count      int         `json:"count"`
	Items      interface{} `json:"items"`
}
