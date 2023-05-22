package main

import (
	"github.com/bwmarrin/lit"
	"time"
)

const tblUsers = "CREATE TABLE IF NOT EXISTS `channels` ( `id` int(11) NOT NULL AUTO_INCREMENT, `chatID` bigint(20) NOT NULL, `feed` mediumtext NOT NULL, `latestDate` datetime NOT NULL, PRIMARY KEY (`id`) );"

// Executes a simple query given a DB
func execQuery(query ...string) {
	for _, q := range query {
		_, err := db.Exec(q)
		if err != nil {
			lit.Error("Error executing query, %s", err)
		}
	}
}

func loadUsers() {
	var chatID int64
	var id int
	var feed string
	var latestDate string
	channels = make(map[int]*channel)

	rows, _ := db.Query("SELECT id, chatID, feed, latestDate FROM channels")
	for rows.Next() {
		_ = rows.Scan(&id, &chatID, &feed, &latestDate)

		t, _ := time.Parse("2006-01-02 15:04:05", latestDate)
		channels[id] = &channel{
			chatID:     chatID,
			Feed:       feed,
			LatestDate: t,
		}
	}
}

func updateLatestDate(id int, latestDate time.Time) {
	_, err := db.Exec("UPDATE channels SET latestDate = ? WHERE id = ?", latestDate, id)
	if err != nil {
		lit.Error("Error while updating latest date, %s", err)
	}

	channels[id].LatestDate = latestDate
}
