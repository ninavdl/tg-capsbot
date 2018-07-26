package main

import (
	FMT "fmt"
	LOG "log"
	OS "os"
	STRINGS "strings"
	TIME "time"
	UNICODE "unicode"

	TB "gopkg.in/tucnak/telebot.v2"
)

// STRING IS A STRING
type STRING = string

// BOOL IS A BOOL
type BOOL = bool

// RUNE IS A RUNE
type RUNE = rune

// TRUE IS TRUE
const TRUE = true

// FALSE IS FALSE
const FALSE = false

// ISUPPERCASE CHECKS WHETHER THE STRING CONTAINS JUST UPPERCASE RUNES
func ISUPPERCASE(TEXT STRING) BOOL {
	for _, R := range []RUNE(TEXT) {
		if UNICODE.IsLetter(R) && !UNICODE.IsUpper(R) {
			return FALSE
		}
	}
	return TRUE
}

// BOT IS THE BOT
var BOT *TB.Bot

func main() {
	TOKEN := OS.Getenv("CAPSBOT_TOKEN")

	var ERR error
	BOT, ERR = TB.NewBot(TB.Settings{
		Token:  TOKEN,
		Poller: &TB.LongPoller{Timeout: 10 * TIME.Second},
	})

	if ERR != nil {
		FMT.Println(ERR)
		return
	}

	BOT.Handle(TB.OnText, TEXTHANDLER)
	BOT.Handle(TB.OnEdited, TEXTHANDLER)

	BOT.Start()
}

// TEXTHANDLER HANDLES TELEGRAM TEXT MESSAGES
func TEXTHANDLER(MSG *TB.Message) {
	if !MSG.FromGroup() || ISUPPERCASE(MSG.Text) {
		return
	}

	MEMBER, ERR := BOT.ChatMemberOf(MSG.Chat, MSG.Sender)
	if ERR != nil {
		LOG.Println(ERR)
		return
	}

	BOT.Delete(MSG)
	var NAME STRING
	if MSG.Sender.Username != "" {
		NAME = "@" + MSG.Sender.Username
	} else {
		NAME = MSG.Sender.FirstName + " " + MSG.Sender.LastName
	}

	BOT.Send(MSG.Chat, STRINGS.ToUpper(NAME)+" RAUS")
	// BANNING AND UNBANNING SHOULD BE EQUIVALENT TO KICKING
	BOT.Ban(MSG.Chat, MEMBER)
	BOT.Unban(MSG.Chat, MSG.Sender)

	LOG.Printf("%s VIOLATED THE NO-LOWER-CASE RULE AND WAS PUNISHED", NAME)
}
