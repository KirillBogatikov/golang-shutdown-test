package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func startWorkers(ctx context.Context, g, b int) (Broker, TimeoutWaitGroup) {
	var (
		notifyBroker = NewSimpleBroker()
		endWg        = NewTimeoutWaitGroup()
	)

	for i := 0; i < g+b; i++ {
		time.Sleep(time.Second)
		endWg.Add(1)
		w := NewWorker(ctx, i, i < g, func() {
			endWg.Done()
		})
		notifyBroker.Subscribe(w)
	}

	return notifyBroker, endWg
}

func waitContextWithTimeout(ctx context.Context, to time.Duration) error {
	timer := time.NewTimer(to)
	select {
	case <-timer.C:
		return errors.New("timeout exceed")
	case <-ctx.Done():
		return nil
	}
}

func main() {
	appCtx, cancelAppCtx := context.WithCancel(context.Background())
	bgCtx, cancelBgCtx := context.WithCancel(appCtx)

	server := http.Server{
		Handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			timer := time.NewTimer(time.Second * 3)
			defer timer.Stop()

			select {
			case <-timer.C:
				log.Println("timer over!")
			case <-request.Context().Done():
				log.Println("ctx done!")
			}
		}),
		BaseContext: func(listener net.Listener) context.Context {
			return appCtx
		},
		Addr: ":80",
	}

	workersBroker, workersWg := startWorkers(bgCtx, 5, 3)

	defer shutdown(cancelAppCtx, cancelBgCtx, workersBroker, workersWg, &server)

	go func() {
		log.Printf("сервер перестал слушать порт: %v", server.ListenAndServe())
	}()
	log.Println("started")

	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Interrupt)
	<-sigChan
	log.Println("stop!")
}

func shutdown(cancelAppCtx, cancelBgCtx context.CancelFunc, workersBroker Broker, workersWg TimeoutWaitGroup, server *http.Server) {
	serverDoneCtx, cancelServerDoneCtx := context.WithTimeout(context.Background(), time.Second*5)
	defer cancelServerDoneCtx()

	// Shutdown сначала отключает прослушивание порта, затем убивает пустые коннекты и бесконечно ждет завершения всех выполняющихся запросов
	// serverDoneCtx ограничивает время ожидания завершения, но сами запросы продолжают жить, т.к. привязаны к appCtx
	log.Printf("не остановили сервер: %v", server.Shutdown(serverDoneCtx))

	// Отправляем воркерам требование остановиться
	workersBroker.Broadcast(MessageDeadPlease)
	// Даем 5 секунд на завершение
	log.Printf("не остановили воркеров: %v", workersWg.Wait(time.Second*5))
	log.Printf("выжившие воркеры: %d", workersWg.count.Load())

	// Принудительно убиваем работу оставшимся воркерам
	// хорошие воркеры должны были завершиться при получении запроса, плохие - умрут в неизвестном состоянии
	log.Println("останавливаем bgCtx")
	cancelBgCtx()

	// Принудительно убиваем работу всего оставшегося - подвисших запросов, баз и тд
	log.Println("останавливаем appCtx")
	cancelAppCtx()
}
