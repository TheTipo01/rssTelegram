package main

import (
	"database/sql"
	"github.com/bwmarrin/lit"
	"github.com/go-co-op/gocron"
	_ "github.com/go-sql-driver/mysql"
	"github.com/kkyr/fig"
	"github.com/microcosm-cc/bluemonday"
	"github.com/mmcdole/gofeed"
	"github.com/pkg/errors"
	tb "gopkg.in/telebot.v4"
	"strings"
	"time"
)

var (
	token     string
	duration  time.Duration
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
	duration = cfg.Duration

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

	db.SetConnMaxLifetime(time.Minute * 3)

	sanitizer = bluemonday.NewPolicy().AllowElements("b", "strong", "i", "em", "u", "ins", "s", "strike", "del", "a", "tg-emoji", "code", "pre")

	execQuery(tblUsers)
	loadUsers()
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

	go startCron()

	lit.Info("rssTelegram started")
	b.Start()
}

func startCron() {
	time.Sleep(time.Second)

	cron := gocron.NewScheduler(time.Local)
	_, err := cron.Every(duration).Do(checkAndSend)
	if err != nil {
		lit.Error("Error starting cron job, %s", err.Error())
	}
	cron.StartAsync()
}

func checkAndSend() {
	for id, c := range channels {
		rss := getRSS(c.Feed)
		// Iterate backwards over all the items, and send only the ones that are newer than the latestDate
		for i := len(rss) - 1; i >= 0; i-- {
			item := rss[i]
			if item.PublishedParsed.After(c.LatestDate) && item.PublishedParsed.Before(time.Now()) {
				_, err := b.Send(tb.ChatID(c.chatID), sanitizer.Sanitize(item.Content)+"\n\n"+item.Link, tb.ModeHTML)
				if err != nil && errors.Is(err, tb.ErrTooLongMessage) {
					_, err = b.Send(tb.ChatID(c.chatID), sanitizer.Sanitize(item.Description)+"\n\n"+item.Link, tb.ModeHTML)
					if err != nil {
						lit.Error("Error sending post with title %s: %s", item.Title, err.Error())
					}
				}
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
