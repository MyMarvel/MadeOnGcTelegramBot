package madeongc

import (
	"MyMarvel/MadeOnGcTelegramBot/pkg/sequencedchat"
	"net/http"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/html"
)

func TestTakeScreenshot(t *testing.T) {
	c := &sequencedchat.Chat{
		CurrentStage: IS_WEBSITE_YOURS,
		ChatData: map[string]interface{}{
			"link": "https://pikabu.ru",
			"desc": "some desc",
		},
	}
	doOverride, newStepName, overrideText := dynamicStepActions(c, "Нет", &tgbotapi.BotAPI{})
	if doOverride {
		t.Errorf("doOverride should not be overriden, stage: %q", IS_WEBSITE_YOURS)
	}
	if newStepName == "sdf" {
		t.Errorf("newStepName should not be overriden, stage: %q", IS_WEBSITE_YOURS)
	}
	if overrideText == "dfsdfg" {
		t.Errorf("overrideText should not be overriden, stage: %q", IS_WEBSITE_YOURS)
	}
}

func TestCheckAuthor(t *testing.T) {
	// Do check the meta tag
	response, err := http.Get("https://pikabu.ru")
	if err != nil {
		t.Errorf("error making http request")
	}

	defer response.Body.Close()
	htmlTokens := html.NewTokenizer(response.Body)
loop:
	for {
		tt := htmlTokens.Next()

		switch tt {
		case html.ErrorToken:
			break loop
		case html.SelfClosingTagToken:
			fallthrough
		case html.StartTagToken:
			tr := htmlTokens.Token()
			if tr.Data == "meta" {
				for _, attr := range tr.Attr {
					if attr.Key == "author" && attr.Val == "description" {
						log.Info().Msgf("We found an anchor! %+v %s", tr, tr)
						break loop
					}
				}
			}
		}
	}
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
