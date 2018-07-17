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

func ISUPPERCASE(text string) bool {
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


	bot.Handle(tb.OnText, TEXTHANDLER)
	bot.Handle(tb.OnEdited, TEXTHANDLER)

	bot.Start()
}

func TEXTHANDLER(msg *tb.Message) {
	if !msg.FromGroup() || ISUPPERCASE(msg.Text) {
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
	// BANNING AND UNBANNING SHOULD BE EQUIVALENT TO KICKING
	bot.Ban(msg.Chat, member)
	bot.Unban(msg.Chat, msg.Sender)
	
	log.Printf("%s VIOLATED THE NO-LOWER-CASE RULE AND WAS PUNISHED", name)
}
