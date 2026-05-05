// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package config

import (
	"database/sql"

	_ "modernc.org/sqlite" // CGO-free sqlite driver
)

type Config struct {
	StratumAddr           string
	StratumUser           string
	NumaEnabled           bool
	HugepagesEnabled      bool
	AffinityEnabled       bool
	PowerLimit            int
	ThermalLimit          int
	PowerSavingsEnabled   bool
	PowerSavingsThreshold int
	TurboModeEnabled      bool
	TotalAcceptedShares   int64
	TotalRejectedShares   int64
	TotalStaleShares      int64
}

type Store struct {
	db *sql.DB
}

func NewStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	schema := `
	CREATE TABLE IF NOT EXISTS settings (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		stratum_addr TEXT,
		stratum_user TEXT,
		numa_enabled INTEGER,
		hugepages_enabled INTEGER,
		affinity_enabled INTEGER,
		power_limit INTEGER,
		thermal_limit INTEGER,
		power_savings_enabled INTEGER,
		power_savings_threshold INTEGER,
		turbo_mode_enabled INTEGER,
		total_accepted_shares INTEGER,
		total_rejected_shares INTEGER,
		total_stale_shares INTEGER
	);`

	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}

	return &Store{db: db}, nil
}

func (s *Store) Load() (*Config, error) {
	cfg := &Config{
		PowerLimit:            100,
		NumaEnabled:           true,
		HugepagesEnabled:      true,
		AffinityEnabled:       true,
		ThermalLimit:          80,
		PowerSavingsEnabled:   false,
		PowerSavingsThreshold: 20,
		TurboModeEnabled:      false,
	}

	err := s.db.QueryRow("SELECT stratum_addr, stratum_user, numa_enabled, hugepages_enabled, affinity_enabled, power_limit, thermal_limit, power_savings_enabled, power_savings_threshold, turbo_mode_enabled, total_accepted_shares, total_rejected_shares, total_stale_shares FROM settings WHERE id = 1").
		Scan(&cfg.StratumAddr, &cfg.StratumUser, &cfg.NumaEnabled, &cfg.HugepagesEnabled, &cfg.AffinityEnabled, &cfg.PowerLimit, &cfg.ThermalLimit, &cfg.PowerSavingsEnabled, &cfg.PowerSavingsThreshold, &cfg.TurboModeEnabled, &cfg.TotalAcceptedShares, &cfg.TotalRejectedShares, &cfg.TotalStaleShares)

	if err == sql.ErrNoRows {
		return cfg, nil
	}
	return cfg, err
}

func (s *Store) Save(cfg *Config) error {
	query := `
	INSERT OR REPLACE INTO settings (id, stratum_addr, stratum_user, numa_enabled, hugepages_enabled, affinity_enabled, power_limit, thermal_limit, power_savings_enabled, power_savings_threshold, turbo_mode_enabled, total_accepted_shares, total_rejected_shares, total_stale_shares)
	VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`

	_, err := s.db.Exec(query,
		cfg.StratumAddr, cfg.StratumUser,
		boolToInt(cfg.NumaEnabled), boolToInt(cfg.HugepagesEnabled), boolToInt(cfg.AffinityEnabled),
		cfg.PowerLimit, cfg.ThermalLimit, boolToInt(cfg.PowerSavingsEnabled), cfg.PowerSavingsThreshold,
		boolToInt(cfg.TurboModeEnabled), cfg.TotalAcceptedShares, cfg.TotalRejectedShares, cfg.TotalStaleShares,
	)
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (s *Store) Close() error {
	return s.db.Close()
}
