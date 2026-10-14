package telegram

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	StateEmp             = ""
	StateW8For           = "waiting_task_name"
	StateTimeRunOut      = "waiting_deadline"
	StateTheasisW8       = "waiting_thesis"
	StateCheck1          = "check_1"
	StateCheck2          = "check_2"
	StateCheck3          = "check_3"
	StateCheckFinal      = "check_final"
	StateDeposit         = "waiting_deposit"
	StateRickProcent     = "waiting_risk_percent"
	StateStopLossProcent = "waiting_stoploss_percent"
	StateFristCoin       = "waiting_first_coin"
	StateSecondCoin      = "waiting_second_coin"
)

type Storage interface {
	Add(chatID int64, task string)
	List(chatID int64) []string
	Delete(chatID int64, index int) error
}
type Bot struct {
	Client    *tgbotapi.BotAPI
	Stor      Storage
	States    map[int64]string
	TaskCache map[int64]string
	StatusMap map[string]string
	CommandMap map[string]string
}

type BinanceResponse struct{
	Symbol string`json:"symbol"`
	Price string`json:"price"`
}

func (b *Bot) handlerPrice(update *tgbotapi.Message){
	chatID := update.Chat.ID
	msg:= tgbotapi.NewMessage(chatID, "Цену какой монеты мы будем смотреть?")
	b.Client.Send(msg)
	b.States[chatID] = StateFristCoin
}
func (b *Bot) handlerFirstCoin(update tgbotapi.Update){
	chatID:= update.Message.Chat.ID
	b.TaskCache[chatID]+=update.Message.Text
	msg:=tgbotapi.NewMessage(chatID, "Цену к какой монете смотрим (например usdt)")
	b.Client.Send(msg)
	b.States[chatID] = StateSecondCoin
}
func (b *Bot) handlerSecondCoin(update tgbotapi.Update){
	chatID:= update.Message.Chat.ID
	b.TaskCache[chatID]+=update.Message.Text
	bigSymbols := strings.ToUpper(b.TaskCache[chatID])
	strprice , err:= getCryptoPrice(bigSymbols)
	if err != nil{
		msg:=tgbotapi.NewMessage(chatID, "Такого соотношения монет не существует")
		b.Client.Send(msg)
		b.handleCancel(update.Message)
		return
	}
	msg:= tgbotapi.NewMessage(chatID, strprice)
	b.Client.Send(msg)
	delete(b.TaskCache, chatID)
	b.States[chatID] = StateEmp
}
func  getCryptoPrice(symbols string) (string, error){
	url:="https://api.binance.com/api/v3/ticker/price?symbol=" + symbols
	resp, err := http.Get(url)
	if err !=nil {
		return "Не удалось обработать запрос", fmt.Errorf("%w", err)
	}
	defer resp.Body.Close()
	var bresp BinanceResponse
	err2 :=  json.NewDecoder(resp.Body).Decode(&bresp)
	if err2!=nil{
		return "Не удалось распарсить данные", fmt.Errorf("%w", err)
	}
	return bresp.Price, nil

}

func BotConstruct(client *tgbotapi.BotAPI, stor Storage) *Bot {
	cmds := map[string]string{
		"add":    "Добавить новую задачу",
		"price":  "Узнать текущий курс криптовалюты",
		"cancel": "Отменить текущее действие",
		"list":   "Показать список всех задач",
		"check":  "Пройти чек-лист перед сделкой",
		"risk":   "Калькулятор риск-менеджмента",
		"del":    "Удалить задачу",
		"cmd":    "Список доступных команд",
		"status": "Узнать текущий статус и активный процесс",
	}

	statuses := map[string]string{
		StateEmp:             "Всё чисто, жду команд",
		StateW8For:           "Ожидаю ввода названия задачи",
		StateTimeRunOut:      "Ожидаю ввода дедлайна",
		StateTheasisW8:       "Ожидаю ввода тезиса",
		StateCheck1:          "Прохожу чек-лист перед покупкой (вопрос 1)",
		StateCheck2:          "Прохожу чек-лист перед покупкой (вопрос 2)",
		StateCheck3:          "Прохожу чек-лист перед покупкой (вопрос 3)",
		StateCheckFinal:      "Ожидаю финального подтверждения чек-листа",
		StateDeposit:         "Ожидаю ввода размера депозита",
		StateRickProcent:     "Ожидаю ввода процента риска",
		StateStopLossProcent: "Ожидаю ввода процента стоп-лосса",
		StateFristCoin:       "Ожидаю тикер первой монеты (например, BTC)",
		StateSecondCoin:      "Ожидаю тикер второй монеты (например, USDT)",
	}

	return &Bot{
		Client:     client,
		Stor:       stor,
		States:     make(map[int64]string),
		TaskCache:  make(map[int64]string),
		StatusMap:  statuses,
		CommandMap: cmds,
	}
}
func (b *Bot) handleAdd(message *tgbotapi.Message) {
	chatID := message.Chat.ID

	b.States[chatID] = StateW8For
	msg := tgbotapi.NewMessage(message.Chat.ID, "Название: ")
	b.Client.Send(msg)
}

func (b *Bot) handleList(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	tasks := b.Stor.List(chatID)
	if len(tasks) == 0 {
		msg := tgbotapi.NewMessage(chatID, "ТВОЙ СПИСОК ЗАДАЧ ПУСТ, ИДИ СПИ")
		b.Client.Send(msg)
		return
	}
	var text string
	for i, t := range tasks {
		text += fmt.Sprintf("%d. %s\n", i+1, t)
	}
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	b.Client.Send(msg)
}

func (b *Bot) handleStart(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	msg := tgbotapi.NewMessage(chatID, `Привет! Я твой таск-менеджер. 
	Доступные команды:
	/add <задача> - добавить
	/list - посмотреть дела
	/del <номер> - удалить задачу`)
	b.Client.Send(msg)
}

func (b *Bot) handleDel(message *tgbotapi.Message) {
	arg := message.CommandArguments()
	chatID := message.Chat.ID
	if arg == "" {
		text := fmt.Sprintln("Укажи номер задачи, например: /del 1")
		msg := tgbotapi.NewMessage(chatID, text)
		b.Client.Send(msg)
		return
	}
	tasknumber, err := strconv.Atoi(arg)
	if err != nil {
		text := fmt.Sprintln("Номер задачи должен быть числом!")
		msg := tgbotapi.NewMessage(chatID, text)
		b.Client.Send(msg)
		return
	}
	erre := b.Stor.Delete(chatID, tasknumber)
	if erre != nil {
		text := erre.Error()
		msg := tgbotapi.NewMessage(message.Chat.ID, text)
		b.Client.Send(msg)
		return
	}
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintln("Задача удалена!"))
	b.Client.Send(msg)
}

func (b *Bot) handleTaskNameInput(update tgbotapi.Update) {
	chatID := update.Message.Chat.ID
	msg := tgbotapi.NewMessage(chatID, "Готово, всё назвали")
	b.Client.Send(msg)
	b.TaskCache[chatID] = update.Message.Text
	msg2 := tgbotapi.NewMessage(chatID, "Принято. Опиши краткий инвест-тезис (почему берем, какая цель или триггер)")
	b.Client.Send(msg2)
	b.States[chatID] = StateTheasisW8
}
func (b *Bot) handleThesisInput(update tgbotapi.Update) {
	chatID := update.Message.Chat.ID
	b.TaskCache[chatID] = b.TaskCache[chatID] + " | " + update.Message.Text
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Краткосрок", "Краткосрок"),
			tgbotapi.NewInlineKeyboardButtonData("Среднесрок", "Среднесрок"),
			tgbotapi.NewInlineKeyboardButtonData("Долгосрок", "Долгосрок"),
		),
	)
	msg := tgbotapi.NewMessage(chatID, "YUOR DEADLINE IS")
	msg.ReplyMarkup = keyboard
	b.Client.Send(msg)
	b.States[chatID] = StateTimeRunOut
}

func (b *Bot) handleDeadlineInput(update tgbotapi.Update) {
	chatID := update.Message.Chat.ID
	task_name := b.TaskCache[chatID]
	fullTask := task_name + "| DEADLINE: " + update.Message.Text
	b.Stor.Add(chatID, fullTask)
	msg := tgbotapi.NewMessage(chatID, "Всё иди занимайся я добавил")
	b.Client.Send(msg)
	b.States[chatID] = StateEmp
	delete(b.TaskCache, chatID)
}

func (b *Bot) handleCallback(cal_back tgbotapi.CallbackQuery) {
	chatID := cal_back.Message.Chat.ID
	text := cal_back.Data
	anws := tgbotapi.NewCallback(cal_back.ID, text)
	b.Client.Request(anws)
	switch b.States[chatID]{
		case  StateTimeRunOut: 
			full_text := b.TaskCache[chatID] + " | DEADLINE: " + text
			b.Stor.Add(chatID, full_text)
			done_work := tgbotapi.NewMessage(chatID, "Готово, есть в листе")
			b.Client.Send(done_work)
			delete(b.TaskCache, chatID)
			b.States[chatID] = StateEmp
		case StateCheck1:
			if text == "КЛЯНУСЬ"{
				msg2 := tgbotapi.NewMessage(chatID, "Это самое взвешанное решение в твоей жизни")
				msg2.ReplyMarkup = b.theThreeDaKeyboard()
				b.Client.Send(msg2)
				b.States[chatID] = StateCheck2
			}else{
				b.handleCancel(cal_back.Message)
				
			}
		case StateCheck2:
			if text == "КЛЯНУСЬ"{
				msg3 := tgbotapi.NewMessage(chatID, "Ты осознаешь что это реально твои честно заработанные кровные деньги и ты ими рискуешь?")
				msg3.ReplyMarkup = b.theThreeDaKeyboard()
				b.Client.Send(msg3)
				b.States[chatID] = StateCheck3
			}else{
				b.handleCancel(cal_back.Message)
			}
		case StateCheck3:
			if text == "КЛЯНУСЬ"{
				msglast := tgbotapi.NewMessage(chatID, "Ну без этой сделки ну вот ну никак не сидишь на месте и с  ней 100% разбогатеешь!?")
				msglast.ReplyMarkup = b.theThreeDaKeyboard()
				b.Client.Send(msglast)
				b.States[chatID] = StateCheckFinal
			}else{
				b.handleCancel(cal_back.Message)
				
			}
		case StateCheckFinal:
			if text == "КЛЯНУСЬ"{
				msg:=tgbotapi.NewMessage(chatID, "Окей выбор за тобой, вперед!")
				b.Client.Send(msg)
				b.States[chatID] = StateEmp
			}else{
				b.handleCancel(cal_back.Message)
				
			}
		default:
			b.States[chatID] = StateEmp
	}
	
}


func (b *Bot) handleCancel(update *tgbotapi.Message) {
	chatID := update.Chat.ID
	b.States[chatID] = StateEmp
	delete(b.TaskCache, chatID)
	msg := tgbotapi.NewMessage(chatID, "да не очкуй, отменили тебе всё")
	b.Client.Send(msg)
}
func (b *Bot) handleStatus(update tgbotapi.Update){
	chatID:= update.Message.Chat.ID
	current := b.States[chatID]
	msg:= tgbotapi.NewMessage(chatID, b.StatusMap[current])
	b.Client.Send(msg)
}

func (b *Bot) theThreeDaKeyboard() tgbotapi.InlineKeyboardMarkup{
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("КЛЯНУСЬ", "КЛЯНУСЬ"),
			tgbotapi.NewInlineKeyboardButtonData("НЕТ", "НЕТ"),
		),
	)
	return  keyboard
}
func (b *Bot) theThreeDa(update *tgbotapi.Message) {
	chatID := update.Chat.ID
	msg := tgbotapi.NewMessage(chatID, "А теперь ответь себе честно на три простых вопроса перед тем как совершить это")
	b.Client.Send(msg)
	msg1 := tgbotapi.NewMessage(chatID, "Сутки прошли с момента твоей гениальной идеи?")
	msg1.ReplyMarkup = b.theThreeDaKeyboard()
	b.Client.Send(msg1)
	b.States[chatID] = 	StateCheck1
}

func (b *Bot) handleRisk(update *tgbotapi.Message){
	chatID:= update.Chat.ID
	msg:=tgbotapi.NewMessage(chatID, "Какой депозит мы считаем ?")
	b.Client.Send(msg)
	b.States[chatID] = StateDeposit
}
func (b *Bot) handleRiskSumHowMuch(update tgbotapi.Update){
	chatID:= update.Message.Chat.ID
	b.TaskCache[chatID] = update.Message.Text
	_, err := strconv.Atoi(update.Message.Text)
	if err!=nil{
		msg:=tgbotapi.NewMessage(chatID, "Это не, число введи заново")
		b.Client.Send(msg)
		return 
	}
	msg:=tgbotapi.NewMessage(chatID, "Окей твой жеп я записал, какой процент риска?")
	b.Client.Send(msg)
	b.States[chatID] = StateRickProcent
}
func (b *Bot) handleRiskProccent(update tgbotapi.Update){
	chatID:= update.Message.Chat.ID
	_, err := strconv.Atoi(update.Message.Text)
	if err!=nil{
		msg:=tgbotapi.NewMessage(chatID, "Это не, число введи заново")
		b.Client.Send(msg)
		return 
	} 
	b.TaskCache[chatID]+="|" + update.Message.Text
	msg:=tgbotapi.NewMessage(chatID, "Окей, твой риск процент у меня, какой стоп лос процент?")
	b.Client.Send(msg)
	b.States[chatID] = StateStopLossProcent
}
func (b *Bot) handleProccesOfUorDep(update tgbotapi.Update){
	chatID:= update.Message.Chat.ID
	casheText:= strings.Split(b.TaskCache[chatID], "|")
	los, errlos:= strconv.ParseFloat(update.Message.Text, 64)
	risk, _:= strconv.ParseFloat(casheText[1], 64)
	dep, _:= strconv.ParseFloat(casheText[0], 64)
	if errlos!=nil{
		msg:=tgbotapi.NewMessage(chatID, "Это не, число введи заново")
		b.Client.Send(msg)
		return 
	}
	form:= dep*risk/los
	str := strconv.FormatFloat(form, 'f', -1, 64)
	msg:=tgbotapi.NewMessage(chatID,str )
	b.Client.Send(msg)
	delete(b.TaskCache, chatID)
	b.States[chatID] = StateEmp
}

func (b *Bot) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 10
	updates := b.Client.GetUpdatesChan(u)
	for update := range updates {
		if update.CallbackQuery != nil {
			b.handleCallback(*update.CallbackQuery)
			continue
		}
		if update.Message == nil {
			continue
		}
		chatID := update.Message.Chat.ID
		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "add":
				b.handleAdd(update.Message)
			case "price":
				b.handlerPrice(update.Message)
			case "cancel":
				b.handleCancel(update.Message)
			case "list":
				b.handleList(update.Message)
			case "check":
				b.theThreeDa(update.Message)
			case "risk":
				b.handleRisk(update.Message)
			case "del":
				b.handleDel(update.Message)
			case "cmd":
				var text string
				for k,v := range b.CommandMap{
					text+= fmt.Sprintf("/%s — %s\n", k, v)
				}
				msg:=tgbotapi.NewMessage(chatID, text)
				b.Client.Send(msg)
			case "status":
				b.handleStatus(update)
			default:
				tetx := fmt.Sprintln("Я не знаю такую команду. Чтобы узнать какие комманды у меня есть введи /cmd")
				msg := tgbotapi.NewMessage(chatID, tetx)
				b.Client.Send(msg)
			}
		} else {
			currentstate := b.States[chatID]
			switch currentstate {
			case StateW8For:
				b.handleTaskNameInput(update)
			case StateTimeRunOut:
				b.handleDeadlineInput(update)
			case StateTheasisW8:
				b.handleThesisInput(update)
			case StateDeposit:
				b.handleRiskSumHowMuch(update)
			case StateFristCoin:
				b.handlerFirstCoin(update)
			case StateSecondCoin:
				b.handlerSecondCoin(update)
			case StateRickProcent:
				b.handleRiskProccent(update)
			case StateStopLossProcent:
				b.handleProccesOfUorDep(update)
			default:
				msg := tgbotapi.NewMessage(chatID, "Что ты по приколу пишешь сюда, возьмись за голову")
				b.Client.Send(msg)
			}
		}

	}
}
