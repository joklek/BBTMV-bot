package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	tb "gopkg.in/tucnak/telebot.v2"
)

type StatusChangeType struct {
	value   int
	message string
}

var (
	DisableMessages = StatusChangeType{value: 0, message: "Pranešimai išjungti! Naudokite komandą /enable kad juos įjungti."}
	EnableMessages  = StatusChangeType{value: 1, message: "Pranešimai įjungti! Naudokite komandą /disable kad juos išjungti."}
)

func handleCommandStats(m *tb.Message) {
	s, err := getStats()
	if err != nil {
		sendTo(m.Sender, errorText)
		return
	}

	msg := fmt.Sprintf(statsTemplate, s.usersCount, s.enabledUsersCount,
		s.usersCount-s.usersWithFee, s.postsCount, s.averagePriceFrom,
		s.averagePriceTo, s.averageRoomsFrom, s.averageRoomsTo)

	sendTo(m.Sender, msg)
}

func handleCommandEnable(m *tb.Message) {
	handleUserStatusChange(m, EnableMessages)
}

func handleCommandDisable(m *tb.Message) {
	handleUserStatusChange(m, DisableMessages)
}

func handleUserStatusChange(m *tb.Message, stateStatus StatusChangeType) {
	message := stateStatus.message
	query := "UPDATE users SET enabled=? WHERE id=?"
	_, err := db.Exec(query, stateStatus.value, m.Sender.ID)
	if err != nil {
		sendTo(m.Sender, errorText)
		return
	}

	ActiveSettingsText, err := getActiveSettingsText(m.Sender)
	if err != nil {
		sendTo(m.Sender, errorText)
		return
	}
	sendTo(m.Sender, message+"\n\n"+ActiveSettingsText)
}

var validConfig = regexp.MustCompile(`^/config (\d{1,7}) (\d{1,7}) (\d{1,2}) (\d{1,2}) (\d{4}) (\d{1,3}) (taip|ne)$`)

func handleCommandConfig(m *tb.Message) {
	msg := strings.ToLower(strings.TrimSpace(m.Text))

	// Check if default:
	if msg == "/config" {
		sendTo(m.Sender, configText)
		return
	}

	// Check if input is valid (using regex)
	if !validConfig.MatchString(msg) {
		sendTo(m.Sender, configErrorText)
		return
	}

	// Extract variables from message (using regex)
	extracted := validConfig.FindStringSubmatch(msg)
	priceFrom, _ := strconv.Atoi(extracted[1])
	priceTo, _ := strconv.Atoi(extracted[2])
	roomsFrom, _ := strconv.Atoi(extracted[3])
	roomsTo, _ := strconv.Atoi(extracted[4])
	yearFrom, _ := strconv.Atoi(extracted[5])
	minFloor, _ := strconv.Atoi(extracted[6])
	showWithFee := strings.ToLower(extracted[7]) == "taip"

	// Values check
	priceCorrect := priceFrom >= 0 || priceTo <= 1000000 && priceTo >= priceFrom
	roomsCorrect := roomsFrom >= 0 || roomsTo <= 100 && roomsTo >= roomsFrom
	yearCorrect := yearFrom <= time.Now().Year()
	minFloorCorrect := minFloor >= 0 && minFloor <= 100

	if !(priceCorrect && roomsCorrect && yearCorrect && minFloorCorrect) {
		sendTo(m.Sender, configErrorText)
		return
	}

	// Update in DB
	query := "UPDATE users SET enabled=1, price_from=?, price_to=?, rooms_from=?, rooms_to=?, year_from=?, min_floor=?, show_with_fee=? WHERE id=?"
	_, err := db.Exec(query, priceFrom, priceTo, roomsFrom, roomsTo, yearFrom, minFloor, showWithFee, m.Sender.ID)
	if err != nil {
		sendTo(m.Sender, errorText)
		return
	}

	ActiveSettingsText, err := getActiveSettingsText(m.Sender)
	if err != nil {
		sendTo(m.Sender, errorText)
		return
	}
	sendTo(m.Sender, "Nustatymai atnaujinti ir pranešimai įjungti!\n\n"+ActiveSettingsText)
}

func handleCommandHelp(m *tb.Message) {
	ActiveSettingsText, err := getActiveSettingsText(m.Sender)
	if err != nil {
		sendTo(m.Sender, errorText)
		return
	}
	sendTo(m.Sender, helpText+"\n\n"+ActiveSettingsText)
}
