package main

import (
	BYTES "bytes"
	JSON "encoding/json"
	FMT "fmt"
	IO "io"
	IOUTIL "io/ioutil"
	LOG "log"
	HTTP "net/http"
	OS "os"
	STRINGS "strings"
	TIME "time"
	UNICODE "unicode"

	MIMETYPE "github.com/gabriel-vasile/mimetype"

	GOSSERACT "github.com/otiai10/gosseract/v2"

	TB "github.com/sour-dough/telebot/v2"

	PDF "github.com/ledongthuc/pdf"
)

// ALIAS STUPID LOWERCASE TYPES AND CONSTANTS
type STRING = string
type BOOL = bool
type RUNE = rune
type INT = int

type RESULT struct {
	FILE_ID        STRING
	FILE_UNIQUE_ID STRING
	FILE_SIZE      INT
	FILE_PATH      STRING
}

type JSON_RESP struct {
	OK     BOOL
	RESULT RESULT
}

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

var HTTP_CLIENT = &HTTP.Client{Timeout: 5 * TIME.Second}

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
	STICKERHANDLER := FILTER(FILTERSTICKER)

	// FILTER TEXT MESSAGES TEXTS
	BOT.Handle(TB.OnText, TEXTHANDLER)
	BOT.Handle(TB.OnEdited, TEXTHANDLER)

	// FILTER MEDIA MESSAGE'S CAPTIONS
	BOT.Handle(TB.OnPhoto, MEDIAHANDLER)
	BOT.Handle(TB.OnAudio, MEDIAHANDLER)
	BOT.Handle(TB.OnVideo, MEDIAHANDLER)

	// FILTER STICKER TEXT
	BOT.Handle(TB.OnSticker, STICKERHANDLER)

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
			NAME = "@" + STRINGS.ToUpper(MSG.Sender.Username)
		} else {
			NAME = STRINGS.ToUpper(MSG.Sender.FirstName) + " " + STRINGS.ToUpper(MSG.Sender.LastName)
		}

		BOT.Send(MSG.Chat, STRINGS.ToUpper(NAME)+" RAUS")
		// BANNING AND UNBANNING SHOULD BE EQUIVALENT TO KICKING
		BOT.Ban(MSG.Chat, MEMBER)
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
			return TRUE
		}
		URL := "https://api.telegram.org/bot" + BOT.Token + "/getFile?file_id=" + MSG.Document.FileID

		J := GETJSON(URL)

		DOCUMENT_URL := "https://api.telegram.org/file/bot" + BOT.Token + "/" + J.RESULT.FILE_PATH

		REQ, ERR := HTTP.NewRequest("GET", DOCUMENT_URL, nil)

		if ERR != nil {
			LOG.Println(ERR)
		}

		RESP, ERR := HTTP_CLIENT.Do(REQ)
		if ERR != nil {
			LOG.Println(ERR)
		}

		_ = OS.Mkdir("DOCUMENTS/", 0700)

		OUT, ERR := OS.Create(STRINGS.ToUpper(J.RESULT.FILE_PATH))

		if ERR != nil {
			LOG.Println(ERR)
		}

		defer OUT.Close()

		_, ERR = IO.Copy(OUT, RESP.Body)

		if ERR != nil {
			LOG.Println(ERR)
		}

		MIMETYPE.SetLimit(0)
		MTYPE, ERR := MIMETYPE.DetectFile("./" + STRINGS.ToUpper(J.RESULT.FILE_PATH))

		defer OS.Remove(STRINGS.ToUpper(J.RESULT.FILE_PATH))
		if MTYPE.Is("text/plain") {
			CONTENT, ERR := OS.ReadFile("./" + STRINGS.ToUpper(J.RESULT.FILE_PATH))

			if ERR != nil {
				LOG.Println(ERR)
			}
			return !ISUPPERCASE(STRING(CONTENT))

		} else {
			// FUTURE CONDITIONAL CODE GOES HERE

			if MTYPE.Is("application/pdf") {
				PDF.DebugOn = TRUE

				FILE, READ, ERR := PDF.Open(STRINGS.ToUpper(J.RESULT.FILE_PATH))

				defer FILE.Close()

				if ERR != nil {
					LOG.Println(ERR)
				}
				var BUFF BYTES.Buffer

				B, ERR := READ.GetPlainText()
				if ERR != nil {
					LOG.Println(ERR)
				}
				BUFF.ReadFrom(B)

				return !ISUPPERCASE(BUFF.String())

			}
		}

	}
	return !ISUPPERCASE(MSG.Caption)
}

// FILTERMEDIA RETURNS TRUE IFF A GIVEN MEDIA MESSAGE'S CAPTION CONTAINS LOWER-CASE LETTERS
func FILTERMEDIA(MSG *TB.Message) BOOL {

	// TESSERACT WITH ENG AND DEU PACKAGES AND LECTONICA PACKAGE NEED TO BE INSTALLED IN ORDER FOR OCR TO WORK PROPERLY
	OCR_CLIENT := GOSSERACT.NewClient()
	defer OCR_CLIENT.Close()

	URL := "https://api.telegram.org/bot" + BOT.Token + "/getFile?file_id=" + MSG.Photo.FileID

	J := GETJSON(URL)

	PHOTO_URL := "https://api.telegram.org/file/bot" + BOT.Token + "/" + J.RESULT.FILE_PATH

	REQ, ERR := HTTP.NewRequest("GET", PHOTO_URL, nil)

	if ERR != nil {
		LOG.Println(ERR)
	}

	RESP, ERR := HTTP_CLIENT.Do(REQ)
	if ERR != nil {
		LOG.Println(ERR)
	}

	_ = OS.Mkdir("PHOTOS/", 0700)

	OUT, ERR := OS.Create(STRINGS.ToUpper(J.RESULT.FILE_PATH))

	if ERR != nil {
		LOG.Println(ERR)
	}

	defer OUT.Close()

	_, ERR = IO.Copy(OUT, RESP.Body)

	if ERR != nil {
		LOG.Println(ERR)
	}

	OCR_CLIENT.SetPageSegMode(6)
	OCR_CLIENT.SetImage("./" + STRINGS.ToUpper(J.RESULT.FILE_PATH))
	// OCR_TEXT, _ := OCR_CLIENT.Text()

	ERR = OS.Remove(STRINGS.ToUpper(J.RESULT.FILE_PATH))

	if ERR != nil {
		LOG.Println(ERR)
	}

	_OUT, ERR := OCR_CLIENT.GetBoundingBoxes(GOSSERACT.RIL_SYMBOL)

	if ERR != nil {
		LOG.Println(ERR)
	}

	MAX := 0.0
	for _, BOX := range _OUT {
		if BOX.Confidence > MAX {
			MAX = BOX.Confidence
		}
		if BOX.Confidence > 95 && !ISUPPERCASE(BOX.Word) {
			return TRUE
		}
	}

	if MAX < 95 {
		LOG.Printf("IMAGE OF "+STRINGS.ToUpper(MSG.Sender.FirstName)+" PASSED BECAUSE OF TOO LOW MAX CONFIDENCE OF %f.", MAX)
		return FALSE
	} else if ISUPPERCASE(MSG.Caption) {
		LOG.Printf("IMAGE OF "+STRINGS.ToUpper(MSG.Sender.FirstName)+" PASSED BECAUSE NO RECOGNISED LETTER WAS LOWERCASE. MAX RECOGNITION CONFIDENCE: %f", MAX)
		return FALSE
	}

	return !ISUPPERCASE(MSG.Caption)
}

func FILTERSTICKER(MSG *TB.Message) BOOL {
	// TESSERACT WITH ENG AND DEU PACKAGES AND LECTONICA PACKAGE NEED TO BE INSTALLED IN ORDER FOR OCR TO WORK PROPERLY
	OCR_CLIENT := GOSSERACT.NewClient()
	defer OCR_CLIENT.Close()

	URL := "https://api.telegram.org/bot" + BOT.Token + "/getFile?file_id=" + MSG.Sticker.FileID

	J := GETJSON(URL)

	// MAKE SURE TO FILTER OUT ANIMATED STICKERS
	if !STRINGS.HasSuffix(J.RESULT.FILE_PATH, ".webp") {
		return FALSE
	}

	STICKER_URL := "https://api.telegram.org/file/bot" + BOT.Token + "/" + J.RESULT.FILE_PATH

	REQ, ERR := HTTP.NewRequest("GET", STICKER_URL, nil)

	if ERR != nil {
		LOG.Println(ERR)
	}

	RESP, ERR := HTTP_CLIENT.Do(REQ)
	if ERR != nil {
		LOG.Println(ERR)
	}

	_ = OS.Mkdir("STICKERS/", 0700)

	OUT, ERR := OS.Create(STRINGS.ToUpper(J.RESULT.FILE_PATH))

	if ERR != nil {
		LOG.Println(ERR)
	}

	defer OUT.Close()

	_, ERR = IO.Copy(OUT, RESP.Body)

	if ERR != nil {
		LOG.Println(ERR)
	}

	OCR_CLIENT.SetPageSegMode(6)
	OCR_CLIENT.SetImage("./" + STRINGS.ToUpper(J.RESULT.FILE_PATH))
	// OCR_TEXT, _ := OCR_CLIENT.Text()

	ERR = OS.Remove(STRINGS.ToUpper(J.RESULT.FILE_PATH))

	if ERR != nil {
		LOG.Println(ERR)
	}

	_OUT, ERR := OCR_CLIENT.GetBoundingBoxes(GOSSERACT.RIL_SYMBOL)

	if ERR != nil {
		LOG.Println(ERR)
	}

	MAX := 0.0
	for _, BOX := range _OUT {
		if BOX.Confidence > MAX {
			MAX = BOX.Confidence
		}
		if BOX.Confidence > 95 && !ISUPPERCASE(BOX.Word) {
			return TRUE
		}
	}

	if MAX < 95 {
		LOG.Printf("STICKER OF "+STRINGS.ToUpper(MSG.Sender.FirstName)+" PASSED BECAUSE OF TOO LOW MAX CONFIDENCE OF %f.", MAX)
		return FALSE
	} else {
		LOG.Printf("STICKER OF "+STRINGS.ToUpper(MSG.Sender.FirstName)+" PASSED BECAUSE NO RECOGNISED LETTER WAS LOWERCASE. MAX RECOGNITION CONFIDENCE: %f", MAX)
		return FALSE
	}
}

func GETJSON(URL STRING) JSON_RESP {
	RESP, ERR := HTTP_CLIENT.Get(URL)
	if ERR != nil {
		LOG.Println(ERR)
	}
	defer RESP.Body.Close()

	BODY, ERR := IOUTIL.ReadAll(RESP.Body)

	if ERR != nil {
		LOG.Println(ERR)
	}

	var J JSON_RESP

	JSON.Unmarshal(BODY, &J)

	return J
}
