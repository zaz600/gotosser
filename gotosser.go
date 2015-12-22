package main

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/hhkbp2/go-strftime"
)

const (
	configFileName = "gotosser.yaml"
)

type processingItem struct {
	srcFile         string
	fullSrcFilePath string
	scangroup       ScanGroup
	//для подсчета размера переданных файлов
	size int64
}

var (
	//контролируем число потоков processScanGroup
	tokens         = make(chan struct{}, 8)
	processingchan = make(chan processingItem, 1000)
	processing     = NewProcessingCache()
	cfg            *Config
)

//перемещаем файл
func moveFile(src, dst string) error {
	err := os.Rename(src, dst)
	if err != nil {
		return err
	}
	return nil
}

//копируем файл
func copyFile(src string, dst string) (err error) {
	sourcefile, err := os.Open(src)
	if err != nil {
		return err
	}

	defer sourcefile.Close()

	destfile, err := os.Create(dst)
	if err != nil {
		return err
	}
	//копируем содержимое и проверяем коды ошибок
	_, err = io.Copy(destfile, sourcefile)
	if closeErr := destfile.Close(); err == nil {
		//если ошибки в io.Copy нет, то берем ошибку от destfile.Close(), если она была
		err = closeErr
	}
	if err != nil {
		return err
	}
	sourceinfo, err := os.Stat(src)
	if err == nil {
		err = os.Chmod(dst, sourceinfo.Mode())
	}
	return err
}

func getAbsPath(dir, file string) (string, error) {
	filePath := filepath.Join(strftime.Format(dir, time.Now()), file)
	abspath, err := filepath.Abs(filePath)
	if err != nil {
		return "", err
	}
	return abspath, nil
}

//проверка на исключения из правил
func needExclude(file string, scangroup ScanGroup, rule CopyRule) bool {
	//пропуск файлов по маскам, заданным в настройках группы
	if cfg.matchExclude(file) {
		return true
	}
	//пропуск файлов по маскам, заданным в настройках группы
	if scangroup.matchExclude(file) {
		return true
	}
	//пропуск файлов по маскам, заданным в настройках правила
	if rule.matchExclude(file) {
		return true
	}
	return false
}

//копирует или перемещает конкретный файл
//в зависимости от заданных правил
func processItem() {
	for item := range processingchan {
		//Проверяем правила
		for _, k := range item.scangroup.getRuleKeys() {
			rule := item.scangroup.Rules[k]
			//Проверяем маски
			if matched, _ := rule.match(item.srcFile); !matched {
				continue
			}

			//проверяем исключения
			if needExclude(item.srcFile, item.scangroup, rule) {
				continue
			}

			//файл подошел под маски правила
			fullDstFilePath, err := getAbsPath(rule.DstDir, item.srcFile)
			if err != nil {
				log.Println("Ошибка вычисления абсолютного пути", err)
				continue
			}

			//создаем каталоги
			fullDstFileDir := filepath.Dir(fullDstFilePath)
			if err := os.MkdirAll(fullDstFileDir, os.ModeDir); err != nil {
				log.Println("Ошибка создания каталога", fullDstFileDir, err)
				continue
			}

			//если файл уже существует
			if _, err := os.Stat(fullDstFilePath); err == nil {
				switch rule.IfExists {
				case "replace":
					log.Printf("Файл существует. %s ifexists=%s. Удаляем файл в конечном каталоге", fullDstFilePath, rule.IfExists)
					if err := os.Remove(fullDstFilePath); err != nil {
						log.Println(err)
						continue
					}
				case "skip":
					log.Printf("Файл существует. %s ifexists=%s. Пропускаем файл", fullDstFilePath, rule.IfExists)
					continue
				default:
					log.Printf("Файл существует. %s Неизвестное значение ifexists=%s. Пропускаем файл", fullDstFilePath, rule.IfExists)
					continue
				}
			}

			switch rule.Mode {
			case "move":
				moveFile(item.fullSrcFilePath, fullDstFilePath)
				//тут надо обработать возможные ошибки
			case "copy":
				copyFile(item.fullSrcFilePath, fullDstFilePath)
				//тут надо обработать возможные ошибки
			default:
				log.Println("Неизвестный режим", rule.Mode)
			}
			//тут надо проверить, если файл перемещён, то другие правила проверять нет смысла
		}
		//после обработки всеми правилами удаляем файл из кэша
		processing.del(item.fullSrcFilePath)
	}
}

func processItems(items []os.FileInfo, fullSrcDir string, scangroup ScanGroup) {
	for _, item := range items {
		// обрабатываем только файлы. Не каталоги, символические ссылки и т.п.
		if !item.Mode().IsRegular() {
			continue
		}
		srcFile := item.Name()
		fullSrcFilePath := filepath.Join(fullSrcDir, srcFile)
		if processing.check(fullSrcFilePath) == true {
			log.Println("файл уже обрабатывается", fullSrcFilePath)
			continue
		}
		//тут надо проверить маски исключения

		//добавляем файл в кэш
		processing.add(fullSrcFilePath)
		processingchan <- processingItem{srcFile, fullSrcFilePath, scangroup, item.Size()}
		log.Println(fullSrcFilePath)
	}
}

func processScanGroup(scangroup ScanGroup) {
	//освобождаем токен после завершения процедуры
	defer func() { <-tokens }()

	for _, srcDir := range scangroup.SrcDirs {
		//разворачиваем маску времени (%Y%m%d и т.п.), если есть в пути
		fullSrcDir, err := getAbsPath(srcDir, "")
		if err != nil {
			log.Println("Ошибка вычисления абсолютного пути", srcDir, err)
			continue
		}
		//создаем каталоги, если необходимо
		if scangroup.СreateSrc {
			if err := os.MkdirAll(fullSrcDir, os.ModeDir); err != nil {
				log.Println("Ошибка создания каталога", fullSrcDir, err)
				continue
			}
		}

		//если каталог уже сканируется, пропускаем его
		if processing.check(fullSrcDir) == true {
			log.Println("каталог уже сканируется", fullSrcDir)
			continue
		}

		log.Println("Сканируем каталог", fullSrcDir)
		//читаем содержимое каталога
		items, err := ioutil.ReadDir(fullSrcDir)
		if err != nil {
			log.Println(err)
			log.Printf("Обработка каталога завершена %s", fullSrcDir)
			continue
		}
		//обрабатываем файлы
		processing.add(fullSrcDir)
		//сделать горутиной
		processItems(items, fullSrcDir, scangroup)
		processing.del(fullSrcDir)
	}
}

// scanLoop просматривает конфиг и для каждого каталога-источника
// запускает горутину processScanGroup
func scanLoop(cfg *Config) {
	for i := 1; i <= cfg.MaxCopyThreads; i++ {
		go processItem()
	}

	for {
		for _, scangroup := range cfg.ScanGroups {
			if scangroup.Enabled != true {
				continue
			}
			//захватываем токен.
			//в этом месте будет пауза, если окажется,
			//что число запущенных горутин processScanDir больше,
			//чем вместимость tokens
			tokens <- struct{}{}
			go processScanGroup(scangroup)
		}
		time.Sleep(time.Duration(cfg.RescanInterval) * time.Second)
	}
}

func main() {
	//загружаем конфиг
	var err error
	cfg, err = reloadConfig(configFileName)
	if err != nil {
		if err != errNotModified {
			log.Fatalf("Не удалось загрузить %s: %s", configFileName, err)
		}
	}
	//log.Printf("%#v", cfg)

	//запускаем цикл сканирования каталогов
	go scanLoop(cfg)

	//ожидаем завершение программы по Ctrl-C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	signal.Notify(sigChan, syscall.SIGTERM)

	for {
		select {
		case <-sigChan:
			log.Println("CTRL-C: Завершаю работу.")
			return
		}
	}
}
