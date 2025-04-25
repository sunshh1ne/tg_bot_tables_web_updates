package main

import (
	"config"
	"database/sql"
	"fmt"
	"log"
	"my_database"
	"parser"
	"strconv"
	"strings"
	"sync"
	"tgbot"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	_ "github.com/mattn/go-sqlite3"
)

var DB my_database.DataBaseSites
var bot tgbot.TGBot
var MU sync.Mutex
var cfg config.Config

type Site struct {
	site_id  string
	url      string
	data     string
	users_id string
	ranges   string
}

func GenerateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func AddUrl(user_id int, msg string) string {
	MU.Lock()
	defer MU.Unlock()

	strs := strings.Split(msg, "\n")
	url := strs[0]
	for len(url) > 0 && url[len(url)-1] == ' ' {
		url = url[:len(url)-1]
	}
	ranges := ""
	if len(strs) > 1 {
		ranges = strs[1]
	}

	site_id := GenerateID()
	data, err := parser.ParseSite(url)
	if err != nil {
		data = " Ошибка при получении данных с сайта. Возможно, вы забыли добавить префикс http:// или https:// в начале URL. "
		return data
	}
	_, err = DB.DB.Exec("INSERT INTO sites VALUES (?, ?, ?, ?, ?);", site_id, url, data, user_id, ranges)
	if err != nil {
		log.Fatal(err)
	}

	var sites_id string
	err = DB.DB.QueryRow("SELECT sites FROM users WHERE user_id=?", user_id).Scan(&sites_id)
	if err != nil {
		log.Fatal(err)
	}
	sites := strings.Split(sites_id, ",")
	if len(sites) == 1 && sites[0] == "" {
		sites = make([]string, 0)
	}
	if len(sites) == 15 {
		return "❗ Ошибка. Уже добавлено слишком много сайтов. ❗"
	}
	sites = append(sites, site_id)
	sites_str := strings.Join(sites, ",")

	_, err = DB.DB.Exec("UPDATE users SET sites = ? WHERE user_id = ?;", sites_str, user_id)
	if err != nil {
		log.Fatal(err)
	}

	return "Успешно добавлен [URL](" + url + ") 🔗"
}

func DelUrl(user_id, site_id int, url string) string {
	MU.Lock()
	defer MU.Unlock()

	var sites_str string
	err := DB.DB.QueryRow("SELECT sites FROM users WHERE user_id=?", user_id).Scan(&sites_str)
	if err != nil {
		log.Fatal(err)
		return "❗ Произошла ошибка при получении списка URL пользователя ❗"
	}

	sites := strings.Split(sites_str, ",")
	if len(sites) == 1 && sites[0] == "" {
		sites = make([]string, 0)
	}

	flag := 0
	for i, s := range sites {
		if s == strconv.Itoa(site_id) {
			sites = append(sites[:i], sites[i+1:]...)
			flag = 1
			break
		}
	}

	if flag == 0 {
		return "У вас не добавлен этот сайт :)"
	}

	_, err = DB.DB.Exec("DELETE FROM sites WHERE site_id = ?", site_id)
	if err != nil {
		log.Fatal(err)
		return "❗ Произошла ошибка при удалении URL ❗"
	}

	sites_str = strings.Join(sites, ",")
	_, err = DB.DB.Exec("UPDATE users SET sites = ? WHERE user_id = ?", sites_str, user_id)
	if err != nil {
		log.Fatal(err)
		return "❗ Произошла ошибка при обновлении списка URL пользователя ❗"
	}

	return "[URL](" + url + ") успешно удален ✔️"
}

func CheckUpdateOnSite(site Site) {
	new_data, err := parser.ParseSite(site.url)
	if err != nil {
		new_data = "Произошла ошибка при получении данных с сайта "
	}
	if site.data == new_data {
		return
	}

	before, after := parser.GetDifferences(site.data, new_data, site.ranges)
	before = before[:min(len(before), cfg.Maxlength)]
	flag := false
	if len(before) == cfg.Maxlength {
		flag = true
	}

	after = after[:min(len(after), cfg.Maxlength)]
	if len(after) == cfg.Maxlength {
		flag = true
	}

	changes := ""
	for i := 0; i < len(before); i++ {
		changes += "БЫЛО:\n" + "```html\n" + before[i] + "```\n" + "СТАЛО:\n" + "```html\n" + after[i] + "```\n" + "\n"
	}
	if flag {
		changes += "Все не поместилось\n"
	}

	text := fmt.Sprintf("ИЗМЕНЕНИЕ НА: %s 🔗\n"+changes,
		"[URL]("+site.url+")")

	users := strings.Split(site.users_id, ",")
	if len(users) == 1 && users[0] == "" {
		users = make([]string, 0)
	}
	for i := 0; i < len(users); i++ {
		user_id, err := strconv.Atoi(users[i])
		if err != nil {
			log.Fatal(err)
		}
		bot.SendMessage(user_id, text)
	}

	_, err = DB.DB.Exec("UPDATE sites SET data = ? WHERE site_id = ?", new_data, site.site_id)
	if err != nil {
		log.Fatal(err)
	}
}

func CheckUpdatesOnAllSites() {
	rows, err := DB.DB.Query("SELECT * FROM sites")
	if err != nil {
		return
	}
	defer rows.Close()

	var wg sync.WaitGroup
	for rows.Next() {
		var site Site
		if err := rows.Scan(&site.site_id, &site.url, &site.data, &site.users_id, &site.ranges); err != nil {
			log.Fatal(err)
		}
		wg.Add(1)
		go func(site Site) {
			defer wg.Done()
			CheckUpdateOnSite(site)
		}(site)
	}
	wg.Wait()
}

func CatchCallbackQuery(update tgbotapi.Update) {
	user_id := update.CallbackQuery.From.ID
	var exists bool
	err := DB.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE user_id = ?);", user_id).Scan(&exists)
	if err != nil {
		log.Fatal(err)
	}
	if !exists {
		_, err := DB.DB.Exec("INSERT INTO users(user_id, sites) VALUES (?, '');", user_id)
		if err != nil {
			log.Fatal(err)
		}
	}

	site_id, err := strconv.Atoi(update.CallbackQuery.Data)
	if err != nil {
		log.Fatal(err)
	}

	err = DB.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM sites WHERE site_id = ?);", site_id).Scan(&exists)
	if err != nil || !exists {
		return
	}

	var url string
	err = DB.DB.QueryRow("SELECT url FROM sites WHERE site_id=?", site_id).Scan(&url)
	if err != nil {
		log.Fatal(err)
	}
	bot.SendMessage(user_id, DelUrl(user_id, site_id, url))

	var sites_id string
	err = DB.DB.QueryRow("SELECT sites FROM users WHERE user_id=?", user_id).Scan(&sites_id)
	if err != nil {
		log.Fatal(err)
	}
	sites := strings.Split(sites_id, ",")
	if len(sites) == 1 && sites[0] == "" {
		sites = make([]string, 0)
	}

	if len(sites) == 0 {
		editedMessageText := tgbotapi.NewEditMessageText(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, "Нет добавленных сайтов 😢")
		_, err = bot.Bot.Send(editedMessageText)
		if err != nil {
			log.Println(err)
		}

		keyboard := tgbotapi.NewInlineKeyboardMarkup()
		editedMessageMarkup := tgbotapi.NewEditMessageReplyMarkup(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, keyboard)
		_, err = bot.Bot.Send(editedMessageMarkup)
		if err != nil {
			log.Println(err)
		}
	} else {
		var rows [][]tgbotapi.InlineKeyboardButton
		for _, site_id := range sites {
			var url string
			err = DB.DB.QueryRow("SELECT url FROM sites WHERE site_id=?", site_id).Scan(&url)
			if err != nil {
				log.Fatal(err)
			}

			btn := tgbotapi.NewInlineKeyboardButtonData(url, site_id)
			row := tgbotapi.NewInlineKeyboardRow(btn)
			rows = append(rows, row)
		}

		keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

		editedMessage := tgbotapi.NewEditMessageReplyMarkup(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, keyboard)
		_, err = bot.Bot.Send(editedMessage)
		if err != nil {
			log.Println(err)
		}
	}
}

func CatchCommand(update tgbotapi.Update) {
	user_id := update.Message.From.ID
	command := update.Message.Command()
	switch command {
	case "start":
		bot.SendMessage(user_id, "Привет!\nЯ помогу отслеживать изменения на онлайн-таблице.\n\nДля добавления таблицы просто отправьте мне URL нужной вам таблицы\nДобавлять с префиксом 'http://' или 'https://'.\n\nДля удаления страницы введите команду /del.\nДля поиска уточняющей информации введите команду /help.")
	case "del":
		var sites_id string
		err := DB.DB.QueryRow("SELECT sites FROM users WHERE user_id=?", user_id).Scan(&sites_id)
		if err != nil {
			log.Fatal(err)
		}
		sites := strings.Split(sites_id, ",")
		if len(sites) == 1 && sites[0] == "" {
			sites = make([]string, 0)
		}

		if len(sites) == 0 {
			bot.SendMessage(user_id, "Нет добавленных сайтов 😢")
		} else {
			var rows [][]tgbotapi.InlineKeyboardButton
			for _, site_id := range sites {
				var url string
				err = DB.DB.QueryRow("SELECT url FROM sites WHERE site_id=?", site_id).Scan(&url)
				if err != nil {
					log.Fatal(err)
				}

				btn := tgbotapi.NewInlineKeyboardButtonData(url, site_id)
				row := tgbotapi.NewInlineKeyboardRow(btn)
				rows = append(rows, row)
			}

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Нажмите на кнопку для удаления:")
			keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
			msg.ReplyMarkup = keyboard

			_, err = bot.Bot.Send(msg)
			if err != nil {
				panic(err)
			}
		}
	case "help":
		bot.SendMessage(user_id, "1. Для добавления онлайн-таблицы просто пришлите мне ссылку на нее. Если вас интересует конкретный диапазон данных, то введите его в формате x1:y1-x2:y2 (без пробелов, такой формат задает прямоугольник с левым верхним углом в ячейке x1:y1 и правым нижним в ячейке x2:y2), в одном сообщении после ссылки на таблицу.\n\n2. Всего вы можете добавить не более 15 онлайн таблиц.\n\n3.Если вы ввели неверный диапазон данных, то автоматически будете получать уведомления обо всех изменениях в таблице.\n\n4. Таблицы обновляются каждую 1 минуту и, соответственно, об изменениях вы узнаете сразу же после обновления.")
	}
}

func CatchMessage(update tgbotapi.Update) {
	user_id := update.Message.From.ID
	var exists bool
	err := DB.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE user_id = ?);", user_id).Scan(&exists)
	if err != nil {
		log.Fatal(err)
	}
	if !exists {
		_, err := DB.DB.Exec("INSERT INTO users(user_id, sites) VALUES (?, '');", user_id)
		if err != nil {
			log.Fatal(err)
		}
	}
	if update.Message.IsCommand() {
		CatchCommand(update)
	} else {
		bot.SendMessage(user_id, AddUrl(user_id, update.Message.Text))
	}
}

func main() {
	cfg = config.LoadConfig("config.json")

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

	go func() {
		for {
			CheckUpdatesOnAllSites()
			time.Sleep(time.Minute * time.Duration(cfg.Check_period))
		}
	}()

	for update := range updates {
		if update.Message != nil {
			CatchMessage(update)
		} else if update.CallbackQuery != nil {
			CatchCallbackQuery(update)
		}
	}
}
