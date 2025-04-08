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

}
