# golang-shutdown-test
Тестируем механизмы остановки сервиса на golang

### Приложение:
Сервер на 80 порту + 8 воркеров (фоновых задач)

### Воркеры
Воркеры есть:
- плохие (читай говнокод), которые игнорируют требования прекратить работу и 
- хорошие, которые делят свою логику на крупные неделимые блоки и перед выполнением 
- каждого проверяют что там с состоянием

### Брокер
`sub.go` содержит интерфейс консьюмера (потребитель сигналов) и брокера - объекта, рассылающего
сигналы подписанным консьюмерам. Почти как `rabbitmq` только в вашем приложении

### Остановка
`main.go` через `signal.Notify` слушает сигналы от ОС и по запросу об остановке (подробнее про сигналы ниже) 
начинает операцию остановки:
- останавливает сервер (новые запросы) с таймаутом в 5 секунд
- отправляет в брокер сообщение с требованием остановиться всем воркерам
- ожидает 5 секунд на остановку всех воркеров
- силой отключает все контексты, прерывая работу всех оставшихся воркеров и запросов

### Проблемы 
1. Прерывание `appCtx` и `bgCtx` без ожидания завершения прямо-всех задач грозит нарушением
согласованности данных
2. Время остановки сервиса в худшем случае растягивается до 10+ секунд (5 на сервер + 5 на воркеры + оверхед)
3. Порой невозможно разделить работу воркера на независимые сегменты
4. pizdec как сложно

### Преимущества
1. При корректном написании воркеров (хорошие воркеры) доступна полная остановка без потери состояния
2. "Безопасное" отключение от БД - appCtx отключается последним

### Проектирование
Брокер и оперирование состоянием заставляет разработчика проектировать каждый метод так,
чтобы вначале выполнялись легкие операции - маппинги, получение данных, тд, а сохранение данных в БД было сгруппировано в один блок 
в конце выполнения метода

### Сигналы ОС
Для захвата в приложении доступны только "нежесткие" сигналы - например, `Interrupt` / `SIGINT`. Его посылают ОС, когда хотят 
чтобы процесс завершился, но жестких ограничений по времени нет. `SIGKILL` - самый жесткий сигнал, отправляемый, например,
командой `kill -9 pid`, это серийный убийца, которому без разницы что происходит в приложении и сколько детей (контекстов-детей) он убьет. 
Для него все поставленные задачи (смерть приложения в кратчайшие сроки) должны быть выполнены на 100%
