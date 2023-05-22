package main

import (
	"database/sql"
	"github.com/bwmarrin/lit"
	"github.com/go-co-op/gocron"
	_ "github.com/go-sql-driver/mysql"
	"github.com/kkyr/fig"
	"github.com/microcosm-cc/bluemonday"
	"github.com/mmcdole/gofeed"
	tb "gopkg.in/telebot.v3"
	"strings"
	"time"
)

var (
	token     string
	db        *sql.DB
	channels  map[int]*channel
	b         *tb.Bot
	sanitizer *bluemonday.Policy
)

func init() {
	lit.LogLevel = lit.LogError

	var cfg config
	err := fig.Load(&cfg, fig.File("config.yml"))
	if err != nil {
		lit.Error(err.Error())
		return
	}

	token = cfg.Token

	// Set lit.LogLevel to the given value
	switch strings.ToLower(cfg.LogLevel) {
	case "logwarning", "warning":
		lit.LogLevel = lit.LogWarning

	case "loginformational", "informational":
		lit.LogLevel = lit.LogInformational

	case "logdebug", "debug":
		lit.LogLevel = lit.LogDebug
	}

	db, err = sql.Open("mysql", cfg.DSN)
	if err != nil {
		lit.Error(err.Error())
		return
	}

	sanitizer = bluemonday.NewPolicy().AllowElements("b", "i", "a", "code", "pre")

	execQuery(tblUsers)
	loadUsers()
	go startCron()
}

func main() {
	var err error

	// Create bot
	b, err = tb.NewBot(tb.Settings{
		Token:  token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		lit.Error(err.Error())
		return
	}

	lit.Info("rssTelegram started")
	b.Start()
}

func startCron() {
	time.Sleep(time.Second)

	cron := gocron.NewScheduler(time.Local)
	_, _ = cron.Every(6).Hours().Do(checkAndSend)
	cron.StartAsync()
}

func checkAndSend() {
	for id, c := range channels {
		rss := getRSS(c.Feed)
		// Iterate backwards over all the items, and send only the ones that are newer than the latestDate
		for i := len(rss) - 1; i >= 0; i-- {
			item := rss[i]
			if item.PublishedParsed.After(c.LatestDate) {
				_, _ = b.Send(tb.ChatID(c.chatID), sanitizer.Sanitize(item.Description)+"\n\n"+item.Link, tb.ModeHTML)
				updateLatestDate(id, *item.PublishedParsed)
			}
		}
	}
}

// getRSS returns the RSS feed as a slice of items
func getRSS(feed string) []*gofeed.Item {
	fp := gofeed.NewParser()
	f, err := fp.ParseURL(feed)
	if err != nil {
		lit.Error(err.Error())
		return nil
	}

	return f.Items
}
