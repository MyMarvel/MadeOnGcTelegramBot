package sequencedchat

import (
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	BUTTONS_KEYBOARD = 0
	BUTTONS_INLINE   = 1
)

type Chat struct {
	ChatId          int64
	CurrentStage    string
	ChatData        map[string]interface{}
	UserName        string
	UserFName       string
	UserLName       string
	LastMessageDate time.Time
}

type SequencedChat struct {
	bot         *tgbotapi.BotAPI
	activeChats sync.Map
	logic       IChatLogic
	buttonsType uint8
}

type LogicStep struct {
	Id       string
	Text     string
	Buttons  []string
	Action   func(c *Chat, userInput string) (bool, string, string)
	Redirect bool
}

type IChatLogic interface {
	GenerateAnswer(*Chat, string, *tgbotapi.BotAPI, *tgbotapi.Message) LogicStep
}

func New(bot *tgbotapi.BotAPI, logic IChatLogic, buttonsType uint8) *SequencedChat {
	return &SequencedChat{
		bot:         bot,
		logic:       logic,
		buttonsType: buttonsType,
	}
}

func (s *SequencedChat) NewMessage(update tgbotapi.Update) {
	if update.Message != nil {
		s.generateAnswer(update.Message.Chat.ID, update.Message.Text, update.Message.From, update.Message)
	} else if update.CallbackQuery != nil {
		// Tell telegram we got the button click, it will show the button name in a nice gray popup for a few seconds
		callback := tgbotapi.NewCallback(update.CallbackQuery.ID, update.CallbackQuery.Data)
		if _, err := s.bot.Request(callback); err != nil {
			log.Error().Err(err)
		}
		// Write an answer into chat
		s.generateAnswer(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Data, update.CallbackQuery.From, update.CallbackQuery.Message)
	}
}

func (s *SequencedChat) generateAnswer(chatId int64, userInput string, fromUser *tgbotapi.User, message *tgbotapi.Message) {
	cha, _ := s.activeChats.LoadOrStore(chatId, Chat{
		ChatId:    chatId,
		UserName:  fromUser.UserName,
		UserFName: fromUser.FirstName,
		UserLName: fromUser.LastName,
		ChatData:  make(map[string]interface{}),
	})
	c := cha.(Chat)

	step := s.logic.GenerateAnswer(&c, userInput, s.bot, message)
	msg := tgbotapi.NewMessage(chatId, step.Text)
	msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
	if len(step.Buttons) > 0 {
		// TODO: Add an ability to use multiple rows
		if s.buttonsType == BUTTONS_INLINE {
			btns := []tgbotapi.InlineKeyboardButton{}
			for _, button := range step.Buttons {
				btns = append(btns, tgbotapi.NewInlineKeyboardButtonData(button, button))
			}
			msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(btns...),
			)
		}
		if s.buttonsType == BUTTONS_KEYBOARD {
			btns := []tgbotapi.KeyboardButton{}
			for _, button := range step.Buttons {
				btns = append(btns, tgbotapi.NewKeyboardButton(button))
			}
			msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(btns...),
			)
		}
	}

	c.CurrentStage = step.Id
	c.LastMessageDate = time.Now()
	s.activeChats.Store(chatId, c)

	if _, err := s.bot.Send(msg); err != nil {
		log.Error().Err(err)
	}

	if step.Redirect {
		s.generateAnswer(chatId, "", fromUser, message)
	}
}
