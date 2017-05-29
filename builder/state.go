package builder

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"sync"
	"time"
)

// Будем хранить в структуре State все данные, необходимые для конкурентного выполнения.
// 1. Задаем канал maxWorkers такой емкости, сколько задач одновременно мы готовы обрабатывать.
// 2. Также задаем список задач, поступивших на обработку, и список обрабатываемых в настоящий момент пользователей.
// Для этих списков ставим мьютексы для того, чтобы конкурентно брать задачи и учитывать пользователей.
// 3. ProcessPool: Одновременная обработка одного и того же репозитория для одного и того же пользователя невозможна (нельзя
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
	logger       logrus.FieldLogger
	processor    Processor
	maxWorkers   chan struct{}
	progressLock *sync.RWMutex
	progress     map[string]string
	tasksLock    *sync.RWMutex
	tasks        []Task
	shutdown     bool
}

// NewState creates new State instance with the parameters:
// - processor to process a task (use builder.Process for it)
// - logger to log important information
// - special channel to be able to process only maxWorkers goroutines in the same time
// - list of processing tasks and a mutex to deal with them
// - list of 'to do' tasks and a mutex to deal with them
// - shutdown channel to mark that service is not available for tasks anymore
func NewState(processor Processor, logger logrus.FieldLogger, maxWorkers int) *State {
	state := &State{
		processor:    processor,
		logger:       logger,
		maxWorkers:   make(chan struct{}, maxWorkers),
		progressLock: &sync.RWMutex{},
		progress:     make(map[string]string),
		tasksLock:    &sync.RWMutex{},
		shutdown:     false,
	}

	return state
}

// AddTask add Task to the pool.
// If shutdown command is sent, the Task will nod be added and the function will return an error.
func (state *State) AddTask(task, prefix, user, repo, commit string) error {
	if state.shutdown {
		return fmt.Errorf("Couldn't process a task. The service is shutting down, please try agiain later.")
	}

	state.logger.Info("Add task")
	state.tasksLock.Lock()
	t := Task{
		task:   task,
		prefix: prefix,
		user:   user,
		repo:   repo,
		commit: commit,
	}
	state.tasks = append(state.tasks, t)
	state.tasksLock.Unlock()
	state.logger.Info("Task added")

	return nil
}

// ProcessPool gets Task from the pool.
// You can call this function concurrently, state.maxWorkers parameter decides if it is possible to add one more worker.
func (state *State) ProcessPool() {
	if len(state.tasks) == 0 {
		return
	}

	state.logger.Info("Waiting to process task...")
	state.maxWorkers <- struct{}{}
	state.logger.Info("Process task...")

	state.tasksLock.Lock()
	state.progressLock.Lock()

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

	state.progressLock.Unlock()
	state.tasksLock.Unlock()

	state.processor(state.logger, t)

	// task was processed, delete it from 'in progress' list
	state.progressLock.Lock()
	delete(state.progress, t.user)
	state.progressLock.Unlock()

	<-state.maxWorkers
	state.logger.Info("Task processed.")
}

// Shutdown function uses state.shutdown channel, so AddTask function will not be able to add Task to the pool.
func (state *State) Shutdown() {
	state.shutdown = true

	// Wait when all tasks will be processed
	for {
		time.Sleep(5 * time.Second)
		state.logger.Info(state.tasks)
		state.logger.Info(state.progress)

		if len(state.tasks) == 0 && len(state.progress) == 0 {
			return
		}
	}
}
