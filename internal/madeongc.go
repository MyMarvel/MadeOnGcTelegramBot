package madeongc

import (
	"MyMarvel/MadeOnGcTelegramBot/pkg/sequencedchat"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/chromedp/chromedp"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golang.org/x/net/html"

	"github.com/rs/zerolog/log"
)

type MadeOnGCLogic struct {
	sequencedchat.StepsHandler
}

// STAGE NAMES
const (
	START              = "Первый экран"
	SEND_WEBSITE_LINK  = "Ждем ссылку на сайт"
	SEND_WEBSITE_DESC  = "Ждем описание сайта"
	FINISH             = "Последний экран"
	IS_WEBSITE_YOURS   = "Вы автор?"
	CONFIRM_OWNERSHIP  = "Ждем подтверждения авторства"
	SHOW_PREVIEW       = "Готовим и показываем скриншот"
	SEND_TO_MODERATION = "Отправить на модерирование"
	EDIT_WARNING       = "Предупреждение что все сбросится"
	OWNERSHIP_SUCCESS  = "Успешно проверили владение сайтом"
	OWNERSHIP_FAILED   = "Мета тег не найден"
)

// BUTTONS
const (
	YES                = "Да"
	NO                 = "Нет"
	SEND               = "Отправить"
	EDIT               = "Редактирование"
	YES_EDIT_WARN      = "Да, редактируем"
	NO_EDIT_WARN       = "Отправляем так"
	OWNER_READY        = "Готово"
	OWNER_CANCEL       = "Пропустить"
	OWNER_READY_AGAIN  = "Исправлено"
	OWNER_CANCEL_AGAIN = "Без авторства"
	REDIRECT           = ""
)

func New() *MadeOnGCLogic {
	staticLogic := map[string]map[string]string{
		"": {
			"/start": START,
		},
		START: {
			"Начать": SEND_WEBSITE_LINK,
		},
		SEND_WEBSITE_LINK: {
			"any": SEND_WEBSITE_DESC, //any is a reserved word for any input.
		},
		SEND_WEBSITE_DESC: {
			"any": IS_WEBSITE_YOURS,
		},
		IS_WEBSITE_YOURS: {
			YES: CONFIRM_OWNERSHIP,
			NO:  SHOW_PREVIEW,
		},
		SHOW_PREVIEW: {
			SEND: SEND_TO_MODERATION,
			EDIT: EDIT_WARNING,
		},
		EDIT_WARNING: {
			YES_EDIT_WARN: SEND_WEBSITE_LINK,
			NO_EDIT_WARN:  SEND_TO_MODERATION,
		},
		SEND_TO_MODERATION: {
			"any": FINISH,
		},
		CONFIRM_OWNERSHIP: {
			OWNER_READY:  OWNERSHIP_SUCCESS,
			OWNER_CANCEL: SHOW_PREVIEW,
		},
		OWNERSHIP_FAILED: {
			OWNER_READY_AGAIN:  OWNERSHIP_SUCCESS,
			OWNER_CANCEL_AGAIN: SHOW_PREVIEW,
		},
		OWNERSHIP_SUCCESS: {
			REDIRECT: SHOW_PREVIEW,
		},
		FINISH: {
			"any": FINISH,
		},
	}
	steps := map[string]sequencedchat.LogicStep{
		START: {
			Text: `Привет!

Я — бот Made on GC  👋
Я помогу вам отправить сайт в канал @madeongc. 

После того, как вы нажмете на «Начать», я спрошу ссылку на сайт, спрошу описание сайта и сделаю его скриншот.

Если вы просто нашли классный сайт, я выложу его без пометки контакта разработчика. Если вы разработчик, это можно будет подтвердить вставкой мета-тега на сайте.

После отправки наши модераторы проверят и опубликуют сайт в канале. 

Нажмите «Начать» 👇
`,
			Buttons: []string{"Начать"},
		},
		SEND_WEBSITE_LINK: {
			Text: `Отправьте ссылку на сайт`,
		},
		SEND_WEBSITE_DESC: {
			Text: `Напишите описание о сайте (не более 140 символов)`,
		},
		IS_WEBSITE_YOURS: {
			Text: `Бот ушёл делать скриншот ⏳

А пока давайте установим авторство. Вы разработчик этого сайта?`,
			Buttons: []string{YES, NO},
		},
		SHOW_PREVIEW: {
			Text:    `Отправляем на модерацию?`,
			Buttons: []string{SEND, EDIT},
		},
		EDIT_WARNING: {
			Text:    `Если вы хотите отредактировать пост, надо будет заполнить всё заново, продолжаем?`,
			Buttons: []string{YES_EDIT_WARN, NO_EDIT_WARN},
		},
		SEND_TO_MODERATION: {
			Text: "Забрали на модерацию ❤️ \nБольшое спасибо, что поделились сайтом!",
		},
		CONFIRM_OWNERSHIP: {
			Text:    `Blank message to be replaced`,
			Buttons: []string{OWNER_READY, OWNER_CANCEL},
		},
		OWNERSHIP_FAILED: {
			Text:    "К сожалению, мы не смогли найти такой мета тег, пожалуйста, проверьте его правильность",
			Buttons: []string{OWNER_READY_AGAIN, OWNER_CANCEL_AGAIN},
		},
		OWNERSHIP_SUCCESS: {
			Text:     "Отлично, мы подтвердили авторство сайта, мы добавим ваш телеграм-ник в пост.",
			Redirect: true,
		},
		FINISH: {
			Text: "Спасибо!",
		},
	}

	return &MadeOnGCLogic{
		StepsHandler: sequencedchat.StepsHandler{
			StaticLogic:        staticLogic,
			Steps:              steps,
			DynamicStepActions: dynamicStepActions,
			DeveloperErrorText: "Произошла ошибка. Пожалуйста, сообщите о ней в нашу редакцию.",
			WhenNotFoundText:   "Извините, я вас не поняла.",
		},
	}
}

/**
 * This function allow to override some steps and their messages.
 * In case a stage name is matched and user entered/clicked on a correct button, a special stage will be returned.
 * If we do not want to change a stage but just change a returned message, we return true, theSameStepName, and a new message.
 * We can send multiple messages to chat and only then return some stage - in this case the stage message will appear in the end.
 * When we return empty overrideText (""), we'll use the default stage text from the steps map.
 */
var dynamicStepActions = func(c *sequencedchat.Chat, userInput string, bot *tgbotapi.BotAPI) (doOverride bool, newStepName string, overrideText string) {
	theSameStepName := c.CurrentStage

	switch true {

	case SEND_WEBSITE_LINK == c.CurrentStage:
		errMsg := "Мы не можем найти такой сайт, проверьте корректность, возможно ошиблись с http или спец. символами, попробуйте снова"
		u, err := url.Parse(userInput)
		if err == nil && u.Scheme != "" && u.Host != "" {
			res, err := http.Get(userInput)
			if err != nil {
				log.Error().Err(err).Msg("error making http request")
				return true, theSameStepName, errMsg
			}

			if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusPermanentRedirect && res.StatusCode != http.StatusMovedPermanently {
				return true, theSameStepName, errMsg
			}

			c.ChatData["link"] = userInput
		} else {
			return true, theSameStepName, errMsg
		}

	case SEND_WEBSITE_DESC == c.CurrentStage:
		if utf8.RuneCountInString(userInput) <= 140 {
			c.ChatData["desc"] = userInput
		} else {
			return true, theSameStepName, "Ваш текст слишком длинный, попробуйте ещё раз."
		}

	case CONFIRM_OWNERSHIP == c.CurrentStage && userInput == OWNER_CANCEL:
		fallthrough
	case OWNERSHIP_FAILED == c.CurrentStage && userInput == OWNER_CANCEL_AGAIN:
		fallthrough
	case OWNERSHIP_SUCCESS == c.CurrentStage && userInput == REDIRECT:
		fallthrough
	case IS_WEBSITE_YOURS == c.CurrentStage && userInput == NO:
		for k, v := range c.ChatData {
			log.Trace().Msgf("ChatData[%s] = %s", k, v)
		}

		msg := tgbotapi.NewMessage(c.ChatId, "Подготавливаем предварительный просмотр ⏳")
		if _, err := bot.Send(msg); err != nil {
			log.Error().Err(err)
		}

		ctx, cancel := chromedp.NewContext(context.Background())
		defer cancel()

		var buf []byte
		if err := chromedp.Run(ctx,
			chromedp.EmulateViewport(1920, 1080),
			chromedp.Navigate(c.ChatData["link"].(string)),
			chromedp.CaptureScreenshot(&buf),
		); err != nil {
			log.Fatal().Err(err)
			return
		}

		filename := time.Now().Format("2006-01-02-15-04-05-00") + ".jpg"
		filepath := os.Getenv("SCREENSHOTS_FOLDER") + "/" + filename
		if err := ioutil.WriteFile(filepath, buf, 0644); err != nil {
			log.Fatal().Err(err)
		}

		c.ChatData["screen"] = filename
		sendCompletedWebsite(c.ChatId, c, bot)

		return true, SHOW_PREVIEW, ""

	case SHOW_PREVIEW == c.CurrentStage && userInput == SEND:
		fallthrough
	case EDIT_WARNING == c.CurrentStage && userInput == NO_EDIT_WARN:
		// TODO: Save this to our database
		// Send it to our moderation bot
		moderationChat, err := strconv.ParseInt(os.Getenv("MODERATION_CHAT_ID"), 10, 64)
		if err != nil {
			log.Error().Err(err).Msg("cannot convert MODERATION_CHAT_ID env var to int64")
			return
		}
		sendCompletedWebsite(moderationChat, c, bot)

	case IS_WEBSITE_YOURS == c.CurrentStage && userInput == YES:
		msg := `Отлично!

Чтобы подтвердить, что вы разработчик, пожалуйста, вставьте мета-тег с вашим ником в телеграме в head сайта.
Бот проверит его и подтвердит авторство.

Код для добавления в head:
<meta author="@{username}">`
		msg = strings.ReplaceAll(msg, "{username}", c.UserName)
		return true, CONFIRM_OWNERSHIP, msg

	case CONFIRM_OWNERSHIP == c.CurrentStage && userInput == OWNER_READY:
		fallthrough
	case OWNERSHIP_FAILED == c.CurrentStage && userInput == OWNER_READY_AGAIN:
		found, err := findMetaTag(c.ChatData["link"].(string), "author", "@"+c.UserName)
		if err != nil {
			log.Error().Err(err).Msg("error making http request")
			return true, theSameStepName, "Произошла ошибка, пожалуйста, обратитесь к модератору."
		}

		if !found {
			return true, OWNERSHIP_FAILED, ""
		} else {
			c.ChatData["dev"] = c.UserName
		}
	}

	return

}

func findMetaTag(url, tagName, tagValue string) (bool, error) {
	// Do check the meta tag
	response, err := http.Get(url)
	if err != nil {
		return false, err
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
					if attr.Key == tagName && attr.Val == tagValue {
						log.Debug().Msgf("we found an author %s", tr)
						return true, nil
					}
				}
			}
		}
	}
	return false, nil
}

func sendCompletedWebsite(chatId int64, c *sequencedchat.Chat, bot *tgbotapi.BotAPI) {
	// send as a file
	filepath := os.Getenv("SCREENSHOTS_FOLDER") + "/" + c.ChatData["screen"].(string)
	photo := tgbotapi.NewPhoto(chatId, tgbotapi.FilePath(filepath))
	if _, err := bot.Send(photo); err != nil {
		log.Fatal().Err(err)
	}

	// Send description
	desc := fmt.Sprintf("%s\n%s", c.ChatData["desc"], c.ChatData["link"])
	dev, ok := c.ChatData["dev"]
	if ok {
		desc += fmt.Sprintf("\nРазработчик: @%s", dev.(string))
	}
	msg := tgbotapi.NewMessage(chatId, desc)
	if _, err := bot.Send(msg); err != nil {
		log.Error().Err(err)
	}
}
