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
	StateW8ForTicker	 = "waiting_ticker_name"

	StateValuePosition   = "waiting_value_amount"
	StateLongLastPosition = "how_long_wait_For_profit"
	StateSellAmount		 = "users_amount_sell"
	StateWaitSellCallback= "keyboard_or_text"
	
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
	Add(chatID int64, pos Position)
	List(chatID int64) []string
	Delete(chatID int64, index int) error
	GetTokenBalance(chatID int64, ticker string) (float64, error)
	GetPortfolio(chatID int64) (map[string]float64, error)
}

type Position struct{	
	Ticker string
	Price float64
	Amount float64
	Period string
}

type RiskCalc struct{
	Deposit float64
	RiskPercent float64
	StopLoss float64
}
type CoinCompare struct{
	FirstCoin string
	SecondCoin string
}

type Bot struct {
	Client    *tgbotapi.BotAPI
	Stor      Storage
	States    map[int64]string
	PostgersCache map[int64]*Position
	StatusMap map[string]string
	CommandMap map[string]string
	RiskCache map[int64]*RiskCalc
	CheckCache map[int64]*CoinCompare
}

type BinanceResponse struct{
	Symbol string`json:"symbol"`
	Price string`json:"price"`
}

func (b *Bot) handleStart(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	msg := tgbotapi.NewMessage(chatID, `Привет! Я твой Инвест-Менеджер. 
	Чтобы узнать доступные команды:
	/cmd`)
	b.Client.Send(msg)
}

func (b *Bot) handlerPrice(update *tgbotapi.Message){
	chatID := update.Chat.ID
	msg:= tgbotapi.NewMessage(chatID, "Цену какой монеты мы будем смотреть?")
	b.Client.Send(msg)
	b.States[chatID] = StateFristCoin
}
func (b *Bot) handlerFirstCoin(update tgbotapi.Update){
	chatID:= update.Message.Chat.ID
	b.CheckCache[chatID] = &CoinCompare{}
	b.CheckCache[chatID].FirstCoin = strings.ToUpper(update.Message.Text)
	msg:=tgbotapi.NewMessage(chatID, "Цену к какой монете смотрим (например usdt)")
	b.Client.Send(msg)
	b.States[chatID] = StateSecondCoin
}
func (b *Bot) handlerSecondCoin(update tgbotapi.Update){
	chatID:= update.Message.Chat.ID
	b.CheckCache[chatID].SecondCoin = strings.ToUpper(update.Message.Text)
	strprice , err:= getCryptoPrice(b.CheckCache[chatID].FirstCoin+b.CheckCache[chatID].SecondCoin)
	if err != nil{
		msg:=tgbotapi.NewMessage(chatID, "Такого соотношения монет не существует")
		b.Client.Send(msg)
		b.handleCancel(update.Message)
		return
	}
	msg:= tgbotapi.NewMessage(chatID, strprice)
	b.Client.Send(msg)
	delete(b.PostgersCache, chatID)
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
		"buy":    "Добавить токен к покупке",
		"price":  "Узнать текущий курс криптовалюты",
		"cancel": "Отменить текущее действие",
		"list":   "Показать портфель",
		"check":  "Пройти чек-лист перед сделкой",
		"risk":   "Калькулятор риск-менеджмента",
		"sell":    "Продать актив",
		"cmd":    "Список доступных команд",
		"status": "Узнать текущий статус и активный процесс",
	}

	statuses := map[string]string{
		StateEmp:             "Всё чисто, жду команд",
		StateW8ForTicker:     "Ожидаю ввода названия токена",
		StateValuePosition:   "Ожидаю ввода объема покупки",
		StateLongLastPosition:"Как долго держать токе",
		StateSellAmount:	  "Сколько токенов продаешь численно",
		StateWaitSellCallback:"Сколько токенов продаешь в процентах (по кнопке)",
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
		PostgersCache:  make(map[int64]*Position),
		StatusMap:  statuses,
		CommandMap: cmds,
		RiskCache: make(map[int64]*RiskCalc),
		CheckCache: make(map[int64]*CoinCompare),
	}
}

func (b *Bot) handleBuyToken(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	b.States[chatID] = StateW8ForTicker
	msg := tgbotapi.NewMessage(message.Chat.ID, "Название: ")
	b.Client.Send(msg)
	
}

func (b *Bot) handleTokenPriceInput(update tgbotapi.Update) {
	chatID := update.Message.Chat.ID
	tokenPrice, err := getCryptoPrice(update.Message.Text+"USDT")
	if err != nil {
		msg:= tgbotapi.NewMessage(chatID, "Не удалось узнать цены!!!!")
		b.Client.Send(msg)
		return
	}
	b.PostgersCache[chatID] = &Position{}
	b.PostgersCache[chatID].Ticker = strings.ToUpper(update.Message.Text) + "USDT"
	numb, _ := strconv.ParseFloat(tokenPrice, 64)
	b.PostgersCache[chatID].Price = numb
	text:= fmt.Sprintf("Сейчас стоит %.2f$", numb)
	msg:=tgbotapi.NewMessage(chatID, text)
	b.Client.Send(msg)
	msg2 := tgbotapi.NewMessage(chatID, "Сколько токенов покупаешь?")
	b.Client.Send(msg2)
	b.States[chatID] = StateValuePosition

}


func (b *Bot) handleLongLastInput(update tgbotapi.Update) {
	chatID := update.Message.Chat.ID
	numb, err := strconv.ParseFloat(update.Message.Text, 64)
	if err != nil{
		msg1 := tgbotapi.NewMessage(chatID, "Это не число, введи цену цифрами")
		b.Client.Send(msg1)
		return
	}
	b.PostgersCache[chatID].Amount = numb
	msg := tgbotapi.NewMessage(chatID, "Период покупки")
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Краткосрок", "Краткосрок"),
			tgbotapi.NewInlineKeyboardButtonData("Среднесрок", "Среднесрок"),
			tgbotapi.NewInlineKeyboardButtonData("Долгосрок", "Долгосрок"),
		),
	)
	
	msg.ReplyMarkup = keyboard
	b.Client.Send(msg)
	b.States[chatID] = StateLongLastPosition
}



func (b *Bot) handleList(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	tasks := b.Stor.List(chatID)
	if len(tasks) == 0 {
		msg := tgbotapi.NewMessage(chatID, "ТВОЙ ПОРТФЕЛЬ ПУСТ, ТЫ brokeBoi")
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



func (b *Bot) handleSell(message *tgbotapi.Message) {
	arg := message.CommandArguments()
	chatID := message.Chat.ID
	if arg == "" {
		text := fmt.Sprintln("Укажи  сразу токен который хочешь продать")
		msg := tgbotapi.NewMessage(chatID, text)
		b.Client.Send(msg)
		return
	}
	bdamount, err := b.Stor.GetTokenBalance(chatID, strings.ToUpper(arg) )
	if err != nil{
		msg:= tgbotapi.NewMessage(chatID, "Не удалось уточнить баланс")
		b.Client.Send(msg)
		b.handleCancel(message)
		return
	}
	if bdamount == 0 {
		msg:=tgbotapi.NewMessage(chatID, "У тебя и так нет этого токена! Нечего продавать")
		b.Client.Send(msg)
		b.handleCancel(message)
		return
	} 
	text:= fmt.Sprintf("Для %v, доступно к продаже : %g", strings.ToUpper(arg), bdamount )
	msg:= tgbotapi.NewMessage(chatID, text)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("10%", "sell_10"),
			tgbotapi.NewInlineKeyboardButtonData("25%", "sell_25"),
			tgbotapi.NewInlineKeyboardButtonData("50%", "sell_50"),
			tgbotapi.NewInlineKeyboardButtonData("100%", "sell_100"),
			tgbotapi.NewInlineKeyboardButtonData("Другая сумма", "sell_users_sum"),
		),
	)
	msg.ReplyMarkup = keyboard
	b.Client.Send(msg)
	b.PostgersCache[chatID] = &Position{Ticker: strings.ToUpper(arg)}
	b.States[chatID] = StateWaitSellCallback
}

func (b *Bot) handleSellAmount(update tgbotapi.Update){
	chatID := update.Message.Chat.ID
	sellamount, err := strconv.ParseFloat(update.Message.Text, 64)
	if err != nil{
		msg1 := tgbotapi.NewMessage(chatID, "Это не число, введи цену цифрами")
		b.Client.Send(msg1)
		return
	}
	bdamount, err := b.Stor.GetTokenBalance(chatID, strings.ToUpper(b.PostgersCache[chatID].Ticker))
	if err != nil{
		msg:=tgbotapi.NewMessage(chatID, "Я забыл что мы там хотели продать")
		b.Client.Send(msg)
		b.handleCancel(update.Message)
		return
	}
	howLast:=fmt.Sprintf("Твой оставшийся баланс %s: %.2f", b.PostgersCache[chatID].Ticker ,bdamount - sellamount)
	b.PostgersCache[chatID].Amount = -sellamount
	sellPriceText, err3:= getCryptoPrice(b.PostgersCache[chatID].Ticker+"USDT")
	if err3!=nil{
		msg:=tgbotapi.NewMessage(chatID, "Не удалось получить актуальную цену")
		b.Client.Send(msg)
		b.handleCancel(update.Message)
		return
	}
	text:= fmt.Sprintf("Токен успешно продана за %s. %s ", sellPriceText, howLast)
	sellPriceReal, _ := strconv.ParseFloat(sellPriceText, 64)
	b.PostgersCache[chatID].Price = sellPriceReal
	msg:=tgbotapi.NewMessage(chatID, text)
	b.Client.Send(msg)
	b.Stor.Add(chatID, *b.PostgersCache[chatID])
	delete(b.PostgersCache, chatID)
	b.States[chatID] = StateEmp
}







func (b *Bot) handleCallback(cal_back tgbotapi.CallbackQuery) {
	chatID := cal_back.Message.Chat.ID
	text := cal_back.Data
	anws := tgbotapi.NewCallback(cal_back.ID, text)
	b.Client.Request(anws)
	switch b.States[chatID]{
		case  StateLongLastPosition: 
			b.PostgersCache[chatID].Period = text
			b.Stor.Add(chatID,*b.PostgersCache[chatID])
			done_work := tgbotapi.NewMessage(chatID, "Позиция успешно добавлена в портфель!")
			b.Client.Send(done_work)
			delete(b.PostgersCache, chatID)
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

		case StateWaitSellCallback:
			switch text{
			case "sell_users_sum":
				msg:= tgbotapi.NewMessage(chatID, "Введи количество монет для продажи числом : ")
				b.Client.Send(msg)
				b.States[chatID] = StateSellAmount
			case "sell_10", "sell_25", "sell_50", "sell_100":

				bdamount, err := b.Stor.GetTokenBalance(chatID, strings.ToUpper(b.PostgersCache[chatID].Ticker))
				if err != nil{
					msg:=tgbotapi.NewMessage(chatID, "Я забыл что мы там хотели продать")
					b.Client.Send(msg)
					b.handleCancel(cal_back.Message)
					return
				}
				parts := strings.Split(text, "_")
				pescent, err2 := strconv.ParseFloat(parts[1], 64)
				if err2 != nil{
					msg:=tgbotapi.NewMessage(chatID, "ВВеди число в процентах")
					b.Client.Send(msg)
					return
				} 
				sellamount:= (pescent/100.0) * bdamount
				b.PostgersCache[chatID].Amount =- sellamount
				howLast:=fmt.Sprintf("Твой оставшийся баланс %s: %.2f", b.PostgersCache[chatID].Ticker ,bdamount - sellamount)
				newprice, err4:= getCryptoPrice(b.PostgersCache[chatID].Ticker+"USDT")
				if err4 != nil{
					msg2:= tgbotapi.NewMessage(chatID, " Не удалось получить актуальную цену этой монеты -> не продаем")
					b.Client.Send(msg2)
					b.handleCancel(cal_back.Message)
					return 
				}
				newfloatprice, _ := strconv.ParseFloat(newprice, 64)
				b.PostgersCache[chatID].Price = newfloatprice
				b.Stor.Add(chatID, *b.PostgersCache[chatID])
				text:= fmt.Sprintf("Позиция успешно продана за %s. %s", newprice, howLast)
				msg:=tgbotapi.NewMessage(chatID, text)
				b.Client.Send(msg)
				delete(b.PostgersCache, chatID)
				b.States[chatID] = StateEmp
			}
		default:
			b.States[chatID] = StateEmp
	}
	
}


func (b *Bot) handleCancel(update *tgbotapi.Message) {
	chatID := update.Chat.ID
	b.States[chatID] = StateEmp
	delete(b.PostgersCache, chatID)
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
	b.RiskCache[chatID] = &RiskCalc{}
	var err error
	b.RiskCache[chatID].Deposit, err = strconv.ParseFloat(update.Message.Text, 64)
	if err !=nil{
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
	var err error
	b.RiskCache[chatID].RiskPercent, err = strconv.ParseFloat(update.Message.Text, 64)
	if err!=nil{
		msg:=tgbotapi.NewMessage(chatID, "Это не, число введи заново")
		b.Client.Send(msg)
		return 
	} 

	msg:=tgbotapi.NewMessage(chatID, "Окей, твой риск процент у меня, какой стоп лос процент?")
	b.Client.Send(msg)
	b.States[chatID] = StateStopLossProcent
}
func (b *Bot) handleProccesOfUorDep(update tgbotapi.Update){
	chatID:= update.Message.Chat.ID
	
	var err error
	b.RiskCache[chatID].StopLoss, err = strconv.ParseFloat(update.Message.Text, 64)
	if err!=nil{
		msg:=tgbotapi.NewMessage(chatID, "Это не, число введи заново")
		b.Client.Send(msg)
		return 
	}
	form:= b.RiskCache[chatID].Deposit * b.RiskCache[chatID].RiskPercent  /  b.RiskCache[chatID].StopLoss
	str := strconv.FormatFloat(form, 'f', -1, 64)
	msg:=tgbotapi.NewMessage(chatID,str )
	b.Client.Send(msg)
	delete(b.RiskCache, chatID)
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
			case "buy":
				b.handleBuyToken(update.Message)
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
			case "sell":
				b.handleSell(update.Message)
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


			case StateW8ForTicker:
				b.handleTokenPriceInput(update)

			case StateValuePosition:
				b.handleLongLastInput(update)
			case StateLongLastPosition:
				if update.CallbackQuery != nil{
					b.handleCallback(*update.CallbackQuery)
				}else{
					msg:=tgbotapi.NewMessage(chatID, "Зря ты это сделал, надо было нажимать на кнопки пальцами!")
					b.Client.Send(msg)
					b.handleCancel(update.Message)
					return
				}
				
			case StateWaitSellCallback:
				if update.CallbackQuery != nil{
					b.handleCallback(*update.CallbackQuery)
				}else{
					msg:=tgbotapi.NewMessage(chatID, "Зря ты это сделал, надо было нажимать на кнопки пальцами!")
					b.Client.Send(msg)
					b.handleCancel(update.Message)
					return
				}
			case StateSellAmount:
				b.handleSellAmount(update)


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
