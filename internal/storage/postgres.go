package storage

import (
	"database/sql"
	"fmt"
	"log"
	"tgdobot/internal/telegram"
	_ "github.com/lib/pq"
)

type PostgresStorage struct {
	db *sql.DB
}


func NewPostgresStorage(path string) (*PostgresStorage ,error){
	baza, err:=sql.Open("postgres", path)
	if err != nil {
    return nil, fmt.Errorf("SQL opening failed: %v", err)
}
	err2:= baza.Ping()
	if err2 != nil{
		fmt.Errorf("No Ping -> not working baza")
	}
	issue:= "CREATE TABLE IF NOT EXISTS portfolio (id serial primary key, chat_id bigint not null, ticker varchar, price double precision, amount double precision, period varchar)"
	_, err3 := baza.Exec(issue)
	if err3!= nil{
		return nil, fmt.Errorf("Issue wrond %v", err3)
	}
	return &PostgresStorage{db: baza}, nil
}

func (p *PostgresStorage) Add(chatID int64, pos telegram.Position) {
	str:= "insert into portfolio (chat_id, ticker, price, amount, period) values ($1, $2, $3, $4, $5)"
	_, err:= p.db.Exec(str, chatID, pos.Ticker, pos.Price, pos.Amount, pos.Period )
	if err != nil {
    	log.Printf("Ошибка сохранения задачи в БД: %v", err)
	}
}

func (p *PostgresStorage) List(chatID int64) []string {
	var tasks []string
	str:= "select task_text from tasks where chat_id = $1 order by id"
	row, err := p.db.Query(str, chatID)
	if err!=nil {
		log.Printf("Без листа сегодня братик либо пробуй ещё: %v", err)
		return tasks
	}
	defer row.Close()
	for row.Next(){
		var task string
		err:= row.Scan(&task)
		if err != nil {
			log.Printf("Ошибка при чтении строки: %v", err)
			continue
		}
		tasks = append(tasks, task)
	}
	return tasks
}

func (p *PostgresStorage) Delete(chatID int64, index int) error {
	offset := index - 1
	var realID int 
	query := "SELECT id FROM tasks WHERE chat_id = $1 ORDER BY id LIMIT 1 OFFSET $2"
	err := p.db.QueryRow(query, chatID, offset).Scan(&realID)
	if err!=nil {
		return fmt.Errorf("задача с таким номером не найдена")
	}
	query2 := "DELETE FROM tasks WHERE id = $1"
	_, err2:= p.db.Exec(query2, realID)
	if err2 != nil {
		return fmt.Errorf("задача не удалена ")
	}
	return nil
}