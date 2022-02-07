package bbtmvbot

import (
	"bbtmvbot/config"
	"bbtmvbot/database"
	"bbtmvbot/website"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/go-co-op/gocron"
	telebot "gopkg.in/tucnak/telebot.v2"
)

var (
	db *database.Database
	tb *telebot.Bot
)

func Start(c *config.Config, dbPath *string) {

	// Open DB
	var err error
	db, err = database.Open(*dbPath)
	if err != nil {
		log.Fatalln(err)
	}

	// Connect to Telegram
	poller := &telebot.LongPoller{Timeout: 10 * time.Second}
	middlewarePoller := telebot.NewMiddlewarePoller(poller, func(upd *telebot.Update) bool {
		if upd.Message != nil {
			db.EnsureUserInDB(upd.Message.Chat.ID) // This ensures that user is always in DB
		}
		if upd.Callback != nil && upd.Callback.Message != nil {
			db.EnsureUserInDB(upd.Callback.Message.Chat.ID)
		}
		return true
	})
	tb, err = telebot.NewBot(telebot.Settings{Token: c.Telegram.ApiKey, Poller: middlewarePoller})
	if err != nil {
		log.Fatalln(err)
	}
	initTelegramHandlers()

	// Start telegram bot
	go tb.Start()

	// Setup cronjob
	location, _ := time.LoadLocation("Europe/Vilnius")
	s := gocron.NewScheduler(location)

	go func() {
		for {
			now := time.Now().In(location)
			if now.Hour() > 1 && now.Hour() < 7 {
				log.Println("It's night, let's sleep and not scan websites")
				time.Sleep(time.Duration(1) * time.Hour)
				continue
			}
			refreshWebsites()
			minimumWaitSeconds := 3 * 60
			maxDelaySeconds := 3 * 60
			randomDelaySeconds := rand.Intn(maxDelaySeconds)
			time.Sleep(time.Duration(minimumWaitSeconds+randomDelaySeconds) * time.Second)
		}
	}()

	s.Every("3m").Do(refreshWebsites) // Retrieve new posts, send to users // TODO randomize
	s.Every("24h").Do(cleanup)        // Cleanup (remove posts that are not seen in the last 30 days)

	// Start cronjob and block execution
	s.StartBlocking()
}

func refreshWebsites() {
	for title, site := range website.Websites {

		go func(title string, site website.Website) {
			posts := site.Retrieve(db)
			for _, post := range posts {
				go processPost(post)
			}
		}(title, site)
	}
}

func processPost(post *website.Post) {
	if post.IsExcludable() {
		db.AddPost(post.Link)
		return
	}

	insertedPostID := db.AddPost(post.Link)

	telegramIDs := db.GetInterestedTelegramIDs(post.Price, post.Rooms, post.Year, post.Floor, post.District, post.IsWithFee())
	for _, telegramID := range telegramIDs {
		sendTelegram(telegramID, post.FormatTelegramMessage(insertedPostID))
	}

	log.Println(fmt.Sprintf(
		"\tID:%d Tel:%s Desc:%d Addr:%d Dist:%s Heat:%d Fl:%d FlTot:%d Area:%d Price:%d Room:%d Year:%d WithFees:%t Link:%s",
		insertedPostID, post.Phone, len(post.Description), len(post.Address()), post.District, len(post.Heating), post.Floor, post.FloorTotal, post.Area, post.Price, post.Rooms, post.Year, post.IsWithFee(), post.Link,
	))
}

func cleanup() {
	db.DeleteOldPosts() // Older than 30 days
}
