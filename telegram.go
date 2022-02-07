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
	configureDistricts(tb)
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
» *Filter by district:* %[9]s

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
	filterByDistrict := "yes"
	if !u.FilterByDistrict {
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
		filterByDistrict,
	)

	return msg
}

func configureDistricts(bot *telebot.Bot) {
	tb.Handle("/districts", showInitialDistricts)
}

func showInitialDistricts(m *telebot.Message) {
	user := db.GetUser(m.Chat.ID)
	var message string
	var content *telebot.ReplyMarkup

	if !user.FilterByDistrict {
		message = "There is a possibility to filter listings by district. Listings without any district will always be shown. Please note that some sites have differet district classifications or names."
		content = showTurnedOffPage(user)
	} else {
		message = "Please select your wanted districts. If none are selected all listings will be shown. Listings without any district will always be shown. Please note that some sites have differet district classifications or names."
		content = showPagedDistricts(m, 0, user)
	}

	tb.Send(m.Sender, message, content)
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func showTurnedOffPage(user *database.User) *telebot.ReplyMarkup {
	var (
		selector  = &telebot.ReplyMarkup{}
		btnTurnOn = selector.Data("✅ Turn on", "on")
	)
	selector.Inline(
		selector.Row(btnTurnOn),
	)
	tb.Handle(&btnTurnOn, func(c *telebot.Callback) {
		db.ToggleFilteringDistricts(user.TelegramID, true)
		tb.Edit(c.Message, showPagedDistricts(c.Message, 0, user))
		tb.Respond(c, &telebot.CallbackResponse{})
	})
	return selector
}

func showPagedDistricts(m *telebot.Message, page int, user *database.User) *telebot.ReplyMarkup {
	districts := db.GetAllDistrictsForUser(user.TelegramID)
	pageSize := 6 // TODO multi row selection
	pageCount := len(districts) / pageSize
	nextPage := min(page+1, pageCount)
	prevPage := max(page-1, 0)

	var (
		selector   = &telebot.ReplyMarkup{}
		btnReset   = selector.Data("🔄", "reset")
		btnPrev    = selector.Data("⬅", "prev", strconv.Itoa(prevPage))
		btnNext    = selector.Data("➡", "next", strconv.Itoa(nextPage))
		btnTurnOff = selector.Data("❌", "off")
	)

	from_i := page * pageSize
	to_i := min(from_i+pageSize, len(districts))
	var btns []telebot.Btn
	for i := from_i; i < to_i; i++ {
		id := strconv.FormatInt(districts[i].Id, 10)
		displayName := districts[i].Name
		if districts[i].Enabled {
			displayName = "✅" + displayName
		}
		btns = append(btns, selector.Data(displayName, id, id))
	}

	var row1 = selector.Row(btns[:min(3, len(btns))]...)
	var row2 telebot.Row
	if len(btns) > 3 {
		row2 = selector.Row(btns[3:]...)
	}

	selector.Inline(
		row1,
		row2,
		selector.Row(btnPrev, btnNext, btnReset, btnTurnOff),
	)

	for _, element := range btns {
		// fmt.Println(index, "=>", element)
		tb.Handle(&element, func(c *telebot.Callback) {
			// Always respond!
			// fmt.Printf("Atradimui: %s \n", c.Data)
			id, err := strconv.Atoi(c.Data)
			if err != nil {
				tb.Respond(c, &telebot.CallbackResponse{ShowAlert: true, Text: "Error while changing pages"})
				return
			}
			added := db.ToggleDistrictForUser(id, user.TelegramID)
			tb.Edit(c.Message, showPagedDistricts(c.Message, page, user))
			if added {
				tb.Respond(c, &telebot.CallbackResponse{ShowAlert: false, Text: "Added to list"})
			} else {
				tb.Respond(c, &telebot.CallbackResponse{ShowAlert: false, Text: "Removed from list"})
			}
		})
	}

	tb.Handle(&btnPrev, changePage)
	tb.Handle(&btnNext, changePage)
	tb.Handle(&btnReset, func(c *telebot.Callback) {
		db.ClearDistricts(user.TelegramID)
		tb.Edit(c.Message, showPagedDistricts(c.Message, 0, user))
		tb.Respond(c, &telebot.CallbackResponse{ShowAlert: false, Text: "List cleared"})
	})
	tb.Handle(&btnTurnOff, func(c *telebot.Callback) {
		db.ToggleFilteringDistricts(user.TelegramID, false)
		tb.Edit(c.Message, showTurnedOffPage(user))
		tb.Respond(c, &telebot.CallbackResponse{})
	})

	return selector
}

func changePage(c *telebot.Callback) {
	user := db.GetUser(c.Message.Chat.ID)
	wantedPage, e := strconv.Atoi(c.Data)
	if e != nil {
		tb.Respond(c, &telebot.CallbackResponse{ShowAlert: true, Text: "Error while changing pages"})
		return
	}
	tb.Edit(c.Message, showPagedDistricts(c.Message, wantedPage, user))
	tb.Respond(c, &telebot.CallbackResponse{ShowAlert: false})
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
