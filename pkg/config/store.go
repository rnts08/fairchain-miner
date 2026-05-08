// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

// Package config provides persistent storage for miner settings.
package config

import (
	"database/sql"
	"fmt"

	"github.com/awnumar/memguard"
	_ "modernc.org/sqlite" // CGO-free sqlite driver
)

type Config struct {
	StratumAddr           []byte
	StratumUser           []byte
	NumaEnabled           bool
	HugepagesEnabled      bool
	AffinityEnabled       bool
	PowerLimit            int
	ThermalLimit          int
	PowerSavingsEnabled   bool
	PowerSavingsThreshold int
	TurboModeEnabled      bool
	TemplateVerification  bool
	TotalAcceptedShares   int64
	TotalRejectedShares   int64
	TotalStaleShares      int64
	ElectricityCost       float64
	PriceOracleAPI        string
	HardwareTDP           int
	CoinPrice             float64
}

// Zero securely wipes sensitive credentials from memory.
func (cfg *Config) Zero() {
	if cfg == nil {
		return
	}
	memguard.WipeBytes(cfg.StratumAddr)
	memguard.WipeBytes(cfg.StratumUser)
}

type Store struct {
	db *sql.DB
}

func NewStore(path string, passphrase string) (*Store, error) {
	// Use the passphrase to open an encrypted SQLite database.
	// The connection string format depends on the specific driver support for encryption.
	dsn := path
	if passphrase != "" {
		dsn = fmt.Sprintf("%s?_pragma=key(%s)", path, passphrase)
	}

	db, err := sql.Open("sqlite", dsn)
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
		template_verification INTEGER,
		total_accepted_shares INTEGER,
		total_rejected_shares INTEGER,
		total_stale_shares INTEGER,
		elec_cost REAL,
		price_oracle_api TEXT,
		hw_tdp INTEGER,		
		coin_price REAL
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
		TemplateVerification:  true,
		ElectricityCost:       0.12,
		HardwareTDP:           250,
		CoinPrice:             0.0,
		PriceOracleAPI:        "https://api.fairchain.org/v1/price", // Default API endpoint
	}

	var addr, user string
	err := s.db.QueryRow("SELECT stratum_addr, stratum_user, numa_enabled, hugepages_enabled, affinity_enabled, power_limit, thermal_limit, power_savings_enabled, power_savings_threshold, turbo_mode_enabled, template_verification, total_accepted_shares, total_rejected_shares, total_stale_shares, elec_cost, price_oracle_api, hw_tdp, coin_price FROM settings WHERE id = 1").
		Scan(&addr, &user, &cfg.NumaEnabled, &cfg.HugepagesEnabled, &cfg.AffinityEnabled, &cfg.PowerLimit, &cfg.ThermalLimit, &cfg.PowerSavingsEnabled, &cfg.PowerSavingsThreshold, &cfg.TurboModeEnabled, &cfg.TemplateVerification, &cfg.TotalAcceptedShares, &cfg.TotalRejectedShares, &cfg.TotalStaleShares, &cfg.ElectricityCost, &cfg.PriceOracleAPI, &cfg.HardwareTDP, &cfg.CoinPrice)

	cfg.StratumAddr = []byte(addr)
	cfg.StratumUser = []byte(user)

	if err == sql.ErrNoRows {
		// If no rows, return default config
		return cfg, nil
	}
	return cfg, err
}

func (s *Store) Save(cfg *Config) error {
	query := `
	INSERT OR REPLACE INTO settings (id, stratum_addr, stratum_user, numa_enabled, hugepages_enabled, affinity_enabled, power_limit, thermal_limit, power_savings_enabled, power_savings_threshold, turbo_mode_enabled, template_verification, total_accepted_shares, total_rejected_shares, total_stale_shares, elec_cost, price_oracle_api, hw_tdp, coin_price)
	VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`

	_, err := s.db.Exec(query,
		string(cfg.StratumAddr),
		string(cfg.StratumUser),
		boolToInt(cfg.NumaEnabled),
		boolToInt(cfg.HugepagesEnabled),
		boolToInt(cfg.AffinityEnabled),
		cfg.PowerLimit,
		cfg.ThermalLimit,
		boolToInt(cfg.PowerSavingsEnabled),
		cfg.PowerSavingsThreshold,
		boolToInt(cfg.TurboModeEnabled),
		boolToInt(cfg.TemplateVerification),
		cfg.TotalAcceptedShares,
		cfg.TotalRejectedShares,
		cfg.TotalStaleShares,
		cfg.ElectricityCost,
		cfg.PriceOracleAPI,
		cfg.HardwareTDP,
		cfg.CoinPrice,
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
