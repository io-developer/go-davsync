package log

type ThreadLog struct {
	Id       int
	TaskId   int
	Complete bool
	Level    Level
	Msg      string
}

type ThreadLogger struct {
	logger   *Logger
	logs     <-chan ThreadLog
	capacity uint
}

func NewThreadLogger(logs <-chan ThreadLog, capacity uint) *ThreadLogger {
	return &ThreadLogger{
		logger:   DefaultLogger,
		logs:     logs,
		capacity: capacity,
	}
}

func (l *ThreadLogger) SetLogger(logger *Logger) {
	l.logger = logger
}

func (l *ThreadLogger) Listen() {
	buf := make([]ThreadLog, l.capacity)
	lastCount := int64(0)
	printBuf := func() {
		for _, item := range buf {
			if !item.Complete && item.Level >= DebugLevel {
				l.logger.Log(item.Level, "*", item.Msg)
			}
		}
		lastCount = l.logger.GetCounter()
	}
	clearBuf := func() {
		for _, item := range buf {
			if !item.Complete && item.Level >= DebugLevel {
				l.logger.Logf(item.Level, "\033[A\033[K")
			}
		}
	}
	for {
		select {
		case item, ok := <-l.logs:
			if !ok {
				return
			}
			if lastCount == l.logger.GetCounter() {
				clearBuf()
			}
			bufItem := buf[item.Id]
			isDelimer := bufItem.TaskId != item.TaskId || item.Complete
			if isDelimer && !bufItem.Complete {
				l.logger.Info(bufItem.Msg)
			}
			buf[item.Id] = item
			printBuf()
		}
	}
}
