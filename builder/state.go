package builder

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"sync"
	"time"
)

// Будем хранить в структуре State все данные, необходимые для конкурентного выполнения.
// 1. Задаем канал workers такой емкости, сколько задач одновременно мы готовы обрабатывать.
// 2. Также задаем список задач, поступивших на обработку, и список обрабатываемых в настоящий момент пользователей.
// Для этих списков ставим мьютексы для того, чтобы конкурентно брать задачи и учитывать пользователей.
// 3. processPool: Одновременная обработка одного и того же репозитория для одного и того же пользователя невозможна (нельзя
// сделать два make build параллельно на разные коммиты), поэтому в случае, если на обработку попадает задача
// для такого пользователя и репозитория, который уже обрабатывается, эту задачу перенесем в конец списка задач.
// 4. Shutdown: В случае, если сервису был отправлен сигнал завершения работы, прием новых задач прекращается
// (AddTask возвращает ошибку). Сервис будет завершен после того, как будут обработаны текущие задачи.

// Task represents a Task for CI/CD.
type Task struct {
	task   string
	prefix string
	user   string
	repo   string
	commit string
}

// Processor is a function to process tasks
type Processor func(logger logrus.FieldLogger, task Task)

// State is a state of building process, it must live during the service is working
type State struct {
	logger     logrus.FieldLogger
	processor  Processor
	maxWorkers chan Task

	mutex     *sync.RWMutex
	progress  map[string]string
	tasks     []Task
}

// NewState creates new State instance with the parameters:
// - processor to process a task (use builder.Process for it)
// - logger to log important information
// - special channel to be able to process only workers goroutines in the same time
// - list of processing tasks and a mutex to deal with them
// - list of 'to do' tasks and a mutex to deal with them
// - shutdown channel to mark that service is not available for tasks anymore
func NewState(processor Processor, logger logrus.FieldLogger, maxWorkers int) *State {
	state := &State{
		processor:  processor,
		logger:     logger,
		maxWorkers: make(chan Task, maxWorkers),
		mutex:      &sync.RWMutex{},
		progress:   make(map[string]string),
	}

	for i:=0; i<maxWorkers; i++ {
		go state.worker()
	}

	go state.processPool()

	return state
}

func (state *State) worker() {
	for {
		select {
		case t := <-state.maxWorkers:
			state.processor(state.logger, t)

			state.mutex.Lock()
			delete(state.progress, t.user)
			state.mutex.Unlock()
		}
	}
}

// AddTask add Task to the pool.
// If shutdown command is sent, the Task will nod be added and the function will return an error.
func (state *State) AddTask(task, prefix, user, repo, commit string) error {
	state.logger.Info("Add task")
	t := Task{
		task:   task,
		prefix: prefix,
		user:   user,
		repo:   repo,
		commit: commit,
	}
	state.setTask(t)
	state.logger.Info("Task added")

	return nil
}

// processPool gets Task from the pool.
// You can call this function concurrently, state.workers parameter decides if it is possible to add one more worker.
func (state *State) processPool() {
	for {
		if len(state.tasks) == 0 {
			continue
		}

		state.mutex.Lock()

		t := state.tasks[0]
		repo, ok := state.progress[t.user]

		// if this user is already in progress, move him to the end of the queue
		// (couldn't process the same user at the same time)
		if ok && (repo == t.repo) {
			state.tasks = append(state.tasks, t)
		} else {
			// otherwise mark user as 'in progress'
			state.progress[t.user] = t.repo
		}

		state.tasks = state.tasks[1:]

		state.mutex.Unlock()

		state.maxWorkers <- t
	}
}

func (state *State) getTask() Task {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	return state.tasks[0]
}

func (state *State) setTask(t Task) {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	state.tasks = append(state.tasks, t)
}

func (state *State) deleteTask() {
	state.mutex.Lock()
	defer state.mutex.Unlock()

}

func (state *State) getCurrent(user string) (string, bool) {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	repo, ok := state.progress[user]
	return repo, ok
}

func (state *State) setCurrent(user, repo string) {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	state.progress[user] = repo
}

func (state *State) deleteCurrent() {

}