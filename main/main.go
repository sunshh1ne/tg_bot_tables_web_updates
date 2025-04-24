package main

import (
	"config"
	"database/sql"
	"log"
	"my_database"
	"tgbot"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	_ "github.com/mattn/go-sqlite3"
)

var DB my_database.DataBaseSites
var bot tgbot.TGBot

type Site struct {
	site_id  string
	url      string
	data     string
	users_id string
}

func CatchCallbackQuery(update tgbotapi.Update) {
	//CallbackQuery := update.CallbackQuery
	//user_id := CallbackQuery.From.ID
	//	TODO
}

func CatchMessage(update tgbotapi.Update) {
	//	TODO
}

func main() {
	cfg := config.LoadConfig("config.json")

	DB.Init()
	defer func(DB *sql.DB) {
		err := DB.Close()
		if err != nil {
			log.Println("Error closing DB")
		}
	}(DB.DB)
	log.Println("Connected to database")

	bot.Init(cfg.TGBotKey)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = cfg.Timeout

	updates, err := bot.Bot.GetUpdatesChan(u)
	if err != nil {
		log.Println("Error getting updates")
	}

	for update := range updates {
		if update.Message != nil {
			CatchMessage(update)
		} else if update.CallbackQuery != nil {
			CatchCallbackQuery(update)
		}
	}

}
