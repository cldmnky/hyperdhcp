// Copyright 2018-present the CoreDHCP Authors. All rights reserved
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package leasedb

import (
	"database/sql"
	"fmt"
	"net"
	"testing"
	"time"

	_ "github.com/chaisql/chai/driver"
	"github.com/stretchr/testify/assert"
)

func testDBSetup() (*sql.DB, error) {
	db, err := sql.Open("chai", ":memory:")
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec("CREATE TABLE IF NOT EXISTS leases4 (mac TEXT NOT NULL, ip TEXT NOT NULL, expiry INTEGER, PRIMARY KEY (mac, ip))"); err != nil {
		return nil, fmt.Errorf("table creation failed: %w", err)
	}
	for _, record := range records {
		stmt, err := db.Prepare("INSERT INTO leases4(mac, ip, expiry) values (?, ?, ?)")
		if err != nil {
			return nil, fmt.Errorf("failed to prepare insert statement: %w", err)
		}
		defer stmt.Close()
		if _, err := stmt.Exec(record.mac, record.ip.IP.String(), record.ip.expires); err != nil {
			return nil, fmt.Errorf("failed to insert record into test db: %w", err)
		}
	}
	return db, nil
}

var expire = int(time.Date(2000, 01, 01, 00, 00, 00, 00, time.UTC).Unix())
var records = []struct {
	mac string
	ip  *Record
}{
	{"02:00:00:00:00:00", &Record{net.IPv4(10, 0, 0, 0), expire}},
	{"02:00:00:00:00:01", &Record{net.IPv4(10, 0, 0, 1), expire}},
	{"02:00:00:00:00:02", &Record{net.IPv4(10, 0, 0, 2), expire}},
	{"02:00:00:00:00:03", &Record{net.IPv4(10, 0, 0, 3), expire}},
	{"02:00:00:00:00:04", &Record{net.IPv4(10, 0, 0, 4), expire}},
	{"02:00:00:00:00:05", &Record{net.IPv4(10, 0, 0, 5), expire}},
}

func TestLoadRecords(t *testing.T) {
	db, err := testDBSetup()
	if err != nil {
		t.Fatalf("Failed to set up test DB: %v", err)
	}

	parsedRec, err := loadRecords(db)
	if err != nil {
		t.Fatalf("Failed to load records from file: %v", err)
	}

	mapRec := make(map[string]*Record)
	for _, rec := range records {
		var (
			ip, mac string
			expiry  int
		)
		if err := db.QueryRow("SELECT mac, ip, expiry FROM leases4 WHERE mac = ?", rec.mac).Scan(&mac, &ip, &expiry); err != nil {
			t.Fatalf("record not found for mac=%s: %v", rec.mac, err)
		}
		mapRec[mac] = &Record{IP: net.ParseIP(ip), expires: expiry}
	}

	assert.Equal(t, mapRec, parsedRec, "Loaded records differ from what's in the DB")
}

func TestWriteRecords(t *testing.T) {
	pl := PluginState{}
	if err := pl.registerBackingDB(":memory:"); err != nil {
		t.Fatalf("Could not setup file: %v", err)
	}

	mapRec := make(map[string]*Record)
	for _, rec := range records {
		hwaddr, err := net.ParseMAC(rec.mac)
		if err != nil {
			// bug in testdata
			panic(err)
		}
		if err := pl.saveIPAddress(hwaddr, rec.ip); err != nil {
			t.Errorf("Failed to save ip for %s: %v", hwaddr, err)
		}
		mapRec[hwaddr.String()] = &Record{IP: rec.ip.IP, expires: rec.ip.expires}
	}

	parsedRec, err := loadRecords(pl.leasedb)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, mapRec, parsedRec, "Loaded records differ from what's in the DB")
}

func TestDuplicateRec(t *testing.T) {
	pl := PluginState{}
	if err := pl.registerBackingDB(":memory:"); err != nil {
		t.Fatalf("Could not setup file: %v", err)
	}

	mapRec := make(map[string]*Record)
	for _, rec := range records {
		hwaddr, err := net.ParseMAC(rec.mac)
		if err != nil {
			// bug in testdata
			panic(err)
		}
		if err := pl.saveIPAddress(hwaddr, rec.ip); err != nil {
			t.Errorf("Failed to save ip for %s: %v", hwaddr, err)
		}
		mapRec[hwaddr.String()] = &Record{IP: rec.ip.IP, expires: rec.ip.expires}
	}
	// Add duplicate record
	hwaddr, err := net.ParseMAC(records[0].mac)
	if err != nil {
		// bug in testdata
		panic(err)
	}
	if err := pl.saveIPAddress(hwaddr, records[0].ip); err != nil {
		t.Errorf("Failed to save ip for %s: %v", hwaddr, err)
	}

	parsedRec, err := loadRecords(pl.leasedb)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, mapRec, parsedRec, "Loaded records differ from what's in the DB")
}
