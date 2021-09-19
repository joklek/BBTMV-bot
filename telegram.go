package bbtmvbot

import (
	"bbtmvbot/database"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	telebot "gopkg.in/tucnak/telebot.v2"
)

func initTelegramHandlers() {
	tb.Handle("/start", handleCommandInfo)
	tb.Handle("/info", handleCommandInfo)
	tb.Handle("/enable", handleCommandEnable)
	tb.Handle("/disable", handleCommandDisable)
	tb.Handle("/config", handleCommandConfig)
}

func handleCommandInfo(m *telebot.Message) {
	sendTelegram(m.Chat.ID, "BBTMV-noRestrict - 'Butų NE TIK Be Tarpininkavimo Mokesčio Vilniuje' is a project intended to help find flats for a rent in Vilnius, Lithuania. All you have to do is to set config using /config command and wait until bot sends you notifications.\n\n**Fun fact** - if you are couple and looking for a flat, then create group chat and add this bot into that group - enable settings and bot will send notifications to the same chat. :)")
}

func handleCommandEnable(m *telebot.Message) {
	user := db.GetUser(m.Chat.ID)
	if user.PriceFrom == 0 && user.PriceTo == 0 && user.RoomsFrom == 0 && user.RoomsTo == 0 && user.YearFrom == 0 {
		sendTelegram(m.Chat.ID, "You must first use /config command before using /enable or /disable commands!")
		return
	}
	if user.Enabled {
		sendTelegram(m.Chat.ID, "Notifications are already enabled!")
		return
	}
	db.SetEnabled(m.Chat.ID, true)
	sendTelegram(m.Chat.ID, "Notifications enabled!")
}

func handleCommandDisable(m *telebot.Message) {
	user := db.GetUser(m.Chat.ID)
	if user.PriceFrom == 0 && user.PriceTo == 0 && user.RoomsFrom == 0 && user.RoomsTo == 0 && user.YearFrom == 0 {
		sendTelegram(m.Chat.ID, "You must first use `/config` command before using `/enable` or `/disable` commands!")
		return
	}
	if !user.Enabled {
		sendTelegram(m.Chat.ID, "Notifications are already disabled!")
		return
	}
	db.SetEnabled(m.Chat.ID, false)
	sendTelegram(m.Chat.ID, "Notifications disabled!")
}

var reConfigCommand = regexp.MustCompile(`^/config (\d{1,5}) (\d{1,5}) (\d{1,2}) (\d{1,2}) (\d{4}) (\d{1,3}) (yes|no)$`)

const configText = "Use this format:\n\n```\n/config <price_from> <price_to> <rooms_from> <rooms_to> <year_from> <min_flor> <show with fee?(yes/no)>\n```\nExample:\n```\n/config 200 330 1 2 2000 2 yes\n```"

const configErrorText = "Wrong input! " + configText

func handleCommandConfig(m *telebot.Message) {
	msg := strings.ToLower(strings.TrimSpace(m.Text))

	// Remove @<botname> from command if exists
	msg = strings.Split(msg, "@")[0]

	// Check if default
	if msg == "/config" {
		sendTelegram(m.Chat.ID, configText+"\n"+activeSettings(m.Chat.ID))
		return
	}

	if !reConfigCommand.MatchString(msg) {
		sendTelegram(m.Chat.ID, configErrorText)
		return
	}

	// Extract variables from message (using regex)
	match := reConfigCommand.FindStringSubmatch(msg)
	priceFrom, _ := strconv.Atoi(match[1])
	priceTo, _ := strconv.Atoi(match[2])
	roomsFrom, _ := strconv.Atoi(match[3])
	roomsTo, _ := strconv.Atoi(match[4])
	yearFrom, _ := strconv.Atoi(match[5])
	minFloor, _ := strconv.Atoi(match[6])
	showWithFees := strings.ToLower(match[7]) == "yes"

	// Values check
	priceCorrect := priceFrom >= 0 || priceTo <= 100000 && priceTo >= priceFrom
	roomsCorrect := roomsFrom >= 0 || roomsTo <= 100 && roomsTo >= roomsFrom
	yearCorrect := yearFrom <= time.Now().Year()
	minFloorCorrect := minFloor >= 0 && minFloor <= 100

	if !(priceCorrect && roomsCorrect && yearCorrect && minFloorCorrect) {
		sendTelegram(m.Chat.ID, configErrorText)
		return
	}

	user := &database.User{
		TelegramID:   m.Chat.ID,
		Enabled:      true,
		PriceFrom:    priceFrom,
		PriceTo:      priceTo,
		RoomsFrom:    roomsFrom,
		RoomsTo:      roomsTo,
		YearFrom:     yearFrom,
		MinFloor:     minFloor,
		ShowWithFees: showWithFees,
	}
	db.UpdateUser(user)
	sendTelegram(m.Chat.ID, "Config updated!\n\n"+activeSettings(m.Chat.ID))
}

const userSettingsTemplate = `*Your active settings:*
» *Notifications:* %[1]s
» *Price:* %[2]d-%[3]d€
» *Rooms:* %[4]d-%[5]d
» *From construction year:* %[6]d
» *Min floor:* %[7]d
» *Show with extra fees:* %[8]s

Current config:
` + "`/config %[2]d %[3]d %[4]d %[5]d %[6]d %[7]d %[8]s`"

func activeSettings(telegramID int64) string {
	u := db.GetUser(telegramID)

	status := "Disabled"
	if u.Enabled {
		status = "Enabled"
	}
	showWithFee := "yes"
	if !u.ShowWithFees {
		showWithFee = "no"
	}

	msg := fmt.Sprintf(
		userSettingsTemplate,
		status,
		u.PriceFrom,
		u.PriceTo,
		u.RoomsFrom,
		u.RoomsTo,
		u.YearFrom,
		u.MinFloor,
		showWithFee,
	)

	return msg
}

var telegramMux sync.Mutex
var elapsedTime time.Duration

func sendTelegram(chatID int64, msg string) {
	telegramMux.Lock()
	defer telegramMux.Unlock()

	startTime := time.Now()
	tb.Send(&telebot.Chat{ID: chatID}, msg, &telebot.SendOptions{
		ParseMode:             "Markdown",
		DisableWebPagePreview: false,
	})
	elapsedTime = time.Since(startTime)

	// See https://core.telegram.org/bots/faq#my-bot-is-hitting-limits-how-do-i-avoid-this
	time.Sleep(30*time.Millisecond - elapsedTime)
}
