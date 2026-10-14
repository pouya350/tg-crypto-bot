package storage

import "fmt"

type MemoryStorage struct {
	tasks map[int64][]string
}

func NewMemoryMap() *MemoryStorage {
	return &MemoryStorage{map[int64][]string{}}
}

func (m *MemoryStorage) Add(chatID int64, task string) {
	m.tasks[chatID] = append(m.tasks[chatID], task)

}

func (m *MemoryStorage) List(chatID int64) []string {
	return m.tasks[chatID]
}

func (m *MemoryStorage) Delete(chatID int64, index int) error {
	task := m.tasks[chatID]
	normindex := index - 1
	if normindex < 0 || normindex >= len(task) {
		return fmt.Errorf("задача с таким номером не найдена")
	}
	m.tasks[chatID] = append(task[:normindex], task[normindex+1:]...)
	return nil
}

