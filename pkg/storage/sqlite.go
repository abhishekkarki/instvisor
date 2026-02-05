package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/abhishekkarki/instvisor/pkg/metrics"
	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStorage implements Storage using SQLite
type SQLiteStorage struct {
	db *sql.DB
}

// NewSQLiteStorage creates a new SQLite storage
func NewSQLiteStorage(path string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set pragmas for better performance
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA cache_size=-64000", // 64MB cache
		"PRAGMA temp_store=MEMORY",
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			return nil, fmt.Errorf("failed to set pragma: %w", err)
		}
	}

	storage := &SQLiteStorage{db: db}

	if err := storage.initialize(); err != nil {
		return nil, err
	}

	return storage, nil
}

func (s *SQLiteStorage) initialize() error {
	schema := `
    CREATE TABLE IF NOT EXISTS metrics (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        timestamp INTEGER NOT NULL,
        metric_name TEXT NOT NULL,
        value REAL NOT NULL,
        labels TEXT,
        unit TEXT
    );

    CREATE INDEX IF NOT EXISTS idx_metric_time ON metrics(metric_name, timestamp);
    CREATE INDEX IF NOT EXISTS idx_timestamp ON metrics(timestamp);
    `

	_, err := s.db.Exec(schema)
	return err
}

func (s *SQLiteStorage) WriteMetrics(metricsData []metrics.Metric) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
        INSERT INTO metrics (timestamp, metric_name, value, labels, unit)
        VALUES (?, ?, ?, ?, ?)
    `)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, m := range metricsData {
		labelsJSON, err := json.Marshal(m.Labels)
		if err != nil {
			return fmt.Errorf("failed to marshal labels: %w", err)
		}

		_, err = stmt.Exec(
			m.Timestamp.Unix(),
			m.Name,
			m.Value,
			string(labelsJSON),
			m.Unit,
		)
		if err != nil {
			return fmt.Errorf("failed to insert metric: %w", err)
		}
	}

	return tx.Commit()
}

func (s *SQLiteStorage) QueryMetrics(name string, start, end time.Time, labels map[string]string) ([]metrics.Metric, error) {
	query := `
        SELECT timestamp, metric_name, value, labels, unit
        FROM metrics
        WHERE metric_name = ? AND timestamp >= ? AND timestamp <= ?
        ORDER BY timestamp ASC
    `

	rows, err := s.db.Query(query, name, start.Unix(), end.Unix())
	if err != nil {
		return nil, fmt.Errorf("failed to query metrics: %w", err)
	}
	defer rows.Close()

	var result []metrics.Metric
	for rows.Next() {
		var m metrics.Metric
		var timestamp int64
		var labelsJSON string

		err := rows.Scan(&timestamp, &m.Name, &m.Value, &labelsJSON, &m.Unit)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		m.Timestamp = time.Unix(timestamp, 0)

		if err := json.Unmarshal([]byte(labelsJSON), &m.Labels); err != nil {
			return nil, fmt.Errorf("failed to unmarshal labels: %w", err)
		}

		// Filter by labels if specified
		if len(labels) > 0 {
			match := true
			for k, v := range labels {
				if m.Labels[k] != v {
					match = false
					break
				}
			}
			if !match {
				continue
			}
		}

		result = append(result, m)
	}

	return result, rows.Err()
}

func (s *SQLiteStorage) DeleteOldMetrics(before time.Time) error {
	_, err := s.db.Exec("DELETE FROM metrics WHERE timestamp < ?", before.Unix())
	if err != nil {
		return fmt.Errorf("failed to delete old metrics: %w", err)
	}

	// Vacuum to reclaim space
	_, err = s.db.Exec("VACUUM")
	return err
}

func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}
