package main

import (
	"log"
	
	"tgdobot/internal/config"
	"tgdobot/internal/storage"
	"tgdobot/internal/telegram"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {

	cfg, err := config.Load()

	if err != nil {
		log.Fatalf("Ошибка конфига: %v", err)
	}

	bot, errnewbot := tgbotapi.NewBotAPI(cfg.TgToken)
	if errnewbot != nil {
		log.Fatalf("Ошибка инициализации бота: %v", errnewbot)
	}

	log.Printf("Бот успешно авторизован! Юзернейм: %s", bot.Self.UserName)
	strmypostgers:= cfg.DbURL
	NewPostger, erro:= storage.NewPostgresStorage(strmypostgers)
	if erro != nil{
		log.Fatalf("Ошибка подключения к БД: %v", erro)
	}
	NewBot := telegram.BotConstruct(bot, NewPostger)
	NewBot.Start()
}
