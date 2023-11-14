package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Ttype - структура задания
type Ttype struct {
	id         int64     // изменил тип на int64 для поддержки UnixNano
	cT         time.Time // время создания задачи
	fT         time.Time // время завершения задачи
	taskRESULT []byte    // результат выполнения задачи
}

// TaskResult - структура результата выполнения задания
type TaskResult struct {
	task Ttype // данuые задания
	err  error // возможная ошибка
}

func main() {
	// Добавил context для корректного завершения всех горутин
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Функция создаёт задания и помещает их в канал a
	taskCreator := func(ctx context.Context, a chan<- Ttype) {
		for {
			select {
			case <-ctx.Done():
				close(a) // Закрытие канала при завершении контекста
				return
			default:
				isSuccess := time.Now().Nanosecond()%2 == 0 // Условие успешности создания задания
				now := time.Now()
				task := Ttype{cT: now, id: now.UnixNano()} // добавил уникальность id

				if !isSuccess {
					task.cT = time.Time{} // случай ошибочного таска
				}

				a <- task // Добавляем задания в канал
			}
		}
	}

	// Создаем канал для заданий
	superChan := make(chan Ttype)

	// Запускаем горутину создания заданий
	go taskCreator(ctx, superChan)

	workersCount := 5
	// Канал для сбора выполнененых заданий
	doneTasks := make(chan TaskResult)

	wg := &sync.WaitGroup{} // sync.WaitGroup для ожидания обработки всех задач
	wg.Add(workersCount)

	// Запускаем пять рабочих горутин для обработки заданий
	for i := 0; i < workersCount; i++ {
		go func() {
			defer wg.Done() // Уменьшаем счетчик ожидания при завершении горутины

			for t := range superChan { // обработка заданий из канала
				workerStartTime := time.Now()

				// Проверка времени создания задания и генерация ошибки
				if t.cT.IsZero() || t.cT.Before(workerStartTime.Add(-20*time.Second)) {
					doneTasks <- TaskResult{task: t, err: fmt.Errorf("something went wrong")}
					continue
				}

				t.fT = time.Now()
				t.taskRESULT = []byte("task has been successed")

				// Имитация работы над заданием
				time.Sleep(time.Millisecond * 150)
				// Отправка исполненного задания в канал
				doneTasks <- TaskResult{task: t}
			}
		}()
	}

	// Горутина ожидает окончания всех рабочих горутин и закрывает канал doneTasks
	go func() {
		wg.Wait()
		close(doneTasks) // Закрываем канал только после завершения всех worker горутин
	}()

	// Обработка полученных заданий и ошибок
	results, errors := processResults(doneTasks)
	printResults(results, errors) // Вывод результатов и ошибок
}

// Функция обрабатывает результаты и возвращает отдельно результаты и ошибки
func processResults(doneTasks <-chan TaskResult) (map[int64]Ttype, []string) {
	results := map[int64]Ttype{}
	errors := []string{}

	// Обход по каналу с результатом
	for taskResult := range doneTasks {
		if taskResult.err != nil {
			errors = append(errors, fmt.Sprintf("Task id: %d error: %v", taskResult.task.id, taskResult.err))
		} else {
			results[taskResult.task.id] = taskResult.task
		}
	}

	return results, errors
}

// Функция для вывода результатов и ошибок
func printResults(results map[int64]Ttype, errors []string) {
	if len(errors) > 0 {
		fmt.Println("Errors:")
		for _, err := range errors {
			fmt.Println(err)
		}
	}

	if len(results) > 0 {
		fmt.Println("Done tasks:")
		for _, task := range results {
			fmt.Printf("ID: %d, cT: %s, fT: %s, result: %s\n", task.id, task.cT, task.fT, string(task.taskRESULT))
		}
	}
}
