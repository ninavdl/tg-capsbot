package main

import (
	FMT "fmt"
	LOG "log"
	OS "os"
	STRINGS "strings"
	TIME "time"
	UNICODE "unicode"

	TB "github.com/sour-dough/telebot/v2"
)

// ALIAS STUPID LOWERCASE TYPES AND CONSTANTS
type STRING = string
type BOOL = bool
type RUNE = rune

const TRUE = true
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
		Token:                   TOKEN,
		Poller:                  &TB.LongPoller{Timeout: 10 * TIME.Second},
		HandleCommandsForOthers: true,
	})

	if ERR != nil {
		FMT.Println(ERR)
		return
	}

	TEXTHANDLER := FILTER(FILTERTEXT)
	MEDIAHANDLER := FILTER(FILTERMEDIA)
	DOCUMENTHANDLER := FILTER(FILTERDOCUMENT)

	// FILTER TEXT MESSAGES TEXTS
	BOT.Handle(TB.OnText, TEXTHANDLER)
	BOT.Handle(TB.OnEdited, TEXTHANDLER)

	// FILTER MEDIA MESSAGE'S CAPTIONS
	BOT.Handle(TB.OnPhoto, MEDIAHANDLER)
	BOT.Handle(TB.OnAudio, MEDIAHANDLER)
	BOT.Handle(TB.OnVideo, MEDIAHANDLER)

	// FILTER DOCUMENT'S CAPTIONS AND FILE NAMES
	BOT.Handle(TB.OnDocument, DOCUMENTHANDLER)

	BOT.Start()
}

// FILTER GROUP MESSAGES BY THE GIVEN FUNCTION
// IF THE FUNCTION RETURNS TRUE, DELETE THE MESSAGE,
// KICK THE USER AND SEND A WARNING MESSAGE
func FILTER(FILTERFUNC func(*TB.Message) BOOL) func(*TB.Message) {
	return func(MSG *TB.Message) {
		if !MSG.FromGroup() {
			return
		}

		if !FILTERFUNC(MSG) {
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
		//UNBAN COMMAND REMOVES USER FROM GROUPS
		BOT.Unban(MSG.Chat, MSG.Sender)

		LOG.Printf("%s VIOLATED THE NO-LOWER-CASE RULE AND WAS PUNISHED", NAME)
	}
}

// FILTERTEXT RETURNS TRUE IFF A GIVEN TEXT MESSAGE CONTAINS LOWER-CASE LETTERS
func FILTERTEXT(MSG *TB.Message) BOOL {
	return !ISUPPERCASE(MSG.Text)
}

// FILTERDOCUMENT RETURNS TRUE IFF A MESSAGE'S ATTACHED FILE'S NAME OR CAPTION CONTAINS LOWER-CASE LETTERS
func FILTERDOCUMENT(MSG *TB.Message) BOOL {
	if MSG.Document != nil {
		if !ISUPPERCASE(MSG.Document.FileName) {
			return true
		}
	}
	return !ISUPPERCASE(MSG.Caption)
}

// FILTERMEDIA RETURNS TRUE IFF A GIVEN MEDIA MESSAGE'S CAPTION CONTAINS LOWER-CASE LETTERS
func FILTERMEDIA(MSG *TB.Message) BOOL {
	return !ISUPPERCASE(MSG.Caption)
}
