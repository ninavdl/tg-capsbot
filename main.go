package main

import (
	tb "gopkg.in/tucnak/telebot.v2" 
	"os"
	"unicode"
	"strings"
	"fmt"
	"log"
	"time"
)

func isUpperCase(text string) bool {
	for _, r := range []rune(text) {
		if (unicode.IsLetter(r) && !unicode.IsUpper(r)) {
			return false
		}
	}
	return true
}

var bot *tb.Bot

func main() {
	token := os.Getenv("CAPSBOT_TOKEN")

	var err error
	bot, err = tb.NewBot(tb.Settings{
		Token: token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})

	if err != nil {
		fmt.Println(err)
		return
	}


	bot.Handle(tb.OnText, textHandler)
	bot.Handle(tb.OnEdited, textHandler)

	bot.Start()
}

func textHandler(msg *tb.Message) {
	if !msg.FromGroup() || isUpperCase(msg.Text) {
		return
	}

	member, err := bot.ChatMemberOf(msg.Chat, msg.Sender)
	if err != nil {
		log.Println(err)
		return
	}

	bot.Delete(msg)
	var name string
	if msg.Sender.Username != "" {
		name = "@" + msg.Sender.Username
	} else {
		name = msg.Sender.FirstName + " " + msg.Sender.LastName
	}

	bot.Send(msg.Chat, strings.ToUpper(name) + " RAUS")
	// banning and unbanning should be equivalent to kicking
	bot.Ban(msg.Chat, member)
	bot.Unban(msg.Chat, msg.Sender)
	
	log.Printf("%s violated the no-lower-case rule and was punished", name)
}