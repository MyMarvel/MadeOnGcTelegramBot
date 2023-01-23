package main

import (
	madeongc "MyMarvel/MadeOnGcTelegramBot/internal"
	"MyMarvel/MadeOnGcTelegramBot/pkg/sequencedchat"
	"os"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	loggerSetup()

	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_APITOKEN"))
	if err != nil {
		log.Fatal().Err(err)
	}
	log.Info().Msgf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)

	updates := bot.GetUpdatesChan(u)
	logic := madeongc.New()
	s := sequencedchat.New(bot, logic)

	for update := range updates {
		go s.NewMessage(update)
	}
}

func loggerSetup() {
	level, err := strconv.Atoi(os.Getenv("LOG_LEVEL"))
	if err != nil {
		panic(err)
	}
	zerolog.SetGlobalLevel(zerolog.Level(level))
	multi := zerolog.ConsoleWriter{Out: os.Stdout}
	log.Logger = zerolog.New(multi).With().Timestamp().Logger()
}
