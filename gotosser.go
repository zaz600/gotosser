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
	statfile       = "tmp/stat.json"
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
	tokens         chan struct{}
	processingchan = make(chan processingItem, 1000)
	processing     = NewProcessingCache()
	cfg            *Config
	tosserstat     = NewTosserStat(statfile)
	savestatchan   chan processingItem

	//эти переменные заполняются линкером.
	//чтобы их передать надо компилировать программу с ключами
	//go build -ldflags "-X main.buildtime '2015-12-22' -X main.version 'v1.0'"
	version   = "debug build"
	buildtime = "n/a"
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

//getAbsPath принимает имя папки(путь) и имя файла
//возвращает абсолютный полный путь к файлу и/или ошибку
//при этом раскрывает в пути переменные времени в формате strftime
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

//проверка файла. если по каким-либо условиям он не подходит,
//возвращаем false
func needProcess(item processingItem, rule CopyRule) bool {
	//Проверяем маски
	if matched, _ := rule.match(item.srcFile); !matched {
		return false
	}
	//проверяем исключения
	if needExclude(item.srcFile, item.scangroup, rule) {
		return false
	}
	return true
}

//копирует или перемещает конкретный файл
//в зависимости от заданных правил
func processItem() {
	for item := range processingchan {
		//Проверяем правила
		fileProcessed := false
		for _, k := range item.scangroup.getRuleKeys() {
			rule := item.scangroup.Rules[k]

			//проверяем файл
			if !needProcess(item, rule) {
				continue
			}
			//файл прошел проверки
			fullDstFilePath, err := getAbsPath(rule.DstDir, item.srcFile)
			if err != nil {
				errorln("Ошибка вычисления абсолютного пути", rule.DstDir, err)
				continue
			}

			//создаем каталоги
			fullDstFileDir := filepath.Dir(fullDstFilePath)
			if err := os.MkdirAll(fullDstFileDir, os.ModeDir); err != nil {
				errorln("Ошибка создания каталога", fullDstFileDir, err)
				continue
			}

			//если файл уже существует
			if _, err := os.Stat(fullDstFilePath); err == nil {
				switch rule.IfExists {
				case "replace":
					Info.Printf("Файл существует. %s ifexists=%s. Удаляем файл в конечном каталоге", fullDstFilePath, rule.IfExists)
					if err := os.Remove(fullDstFilePath); err != nil {
						errorln(err)
						continue
					}
				case "skip":
					Info.Printf("Файл существует. %s ifexists=%s. Пропускаем файл", fullDstFilePath, rule.IfExists)
					continue
				default:
					Info.Printf("Файл существует. %s Неизвестное значение ifexists=%s. Пропускаем файл", fullDstFilePath, rule.IfExists)
					continue
				}
			}

			//Обработка файла
			fileMoved := false
			switch rule.Mode {
			case "move":
				if err := moveFile(item.fullSrcFilePath, fullDstFilePath); err == nil {
					fileMoved = true
					fileProcessed = true
					FileLog.Printf("%s -> %s", item.fullSrcFilePath, fullDstFilePath)
				}
			case "copy":
				if err := copyFile(item.fullSrcFilePath, fullDstFilePath); err == nil {
					fileProcessed = true
				}
			default:
				errorln("Неизвестный режим", rule.Mode)
			}

			//если файл перемещён, то другие правила проверять нет смысла
			if fileMoved {
				break
			}
		}
		if fileProcessed {
			//файл обработан сохраняем статистику
			savestatchan <- item
		}
		//после обработки всеми правилами удаляем файл из кэша
		processing.del(item.fullSrcFilePath)
	}
}

//processItems обрабатывает список файлов
func processItems(items []os.FileInfo, fullSrcDir string, scangroup ScanGroup) {
	for _, item := range items {
		// обрабатываем только файлы. Не каталоги, символические ссылки и т.п.
		if !item.Mode().IsRegular() {
			continue
		}
		srcFile := item.Name()

		//пропускаем файлы, попадающе под глобальный список исключений
		//или список исключений группы
		if needExclude(srcFile, scangroup, CopyRule{}) {
			continue
		}

		fullSrcFilePath := filepath.Join(fullSrcDir, srcFile)
		if processing.check(fullSrcFilePath) == true {
			Debug.Println("Файл уже обрабатывается", fullSrcFilePath)
			continue
		}

		//добавляем файл в кэш
		processing.add(fullSrcFilePath)
		processingchan <- processingItem{srcFile, fullSrcFilePath, scangroup, item.Size()}
		Info.Println(fullSrcFilePath)
	}
}

func processScanGroup(scangroup ScanGroup) {
	//освобождаем токен после завершения процедуры
	defer func() { <-tokens }()

	for _, srcDir := range scangroup.SrcDirs {
		//разворачиваем маску времени (%Y%m%d и т.п.), если есть в пути
		fullSrcDir, err := getAbsPath(srcDir, "")
		if err != nil {
			errorln("Ошибка вычисления абсолютного пути", srcDir, err)
			continue
		}
		Debug.Println("Сканируем каталог", fullSrcDir)

		//если каталог уже сканируется, пропускаем его
		if processing.check(fullSrcDir) == true {
			Debug.Println("Каталог уже сканируется", fullSrcDir)
			continue
		}

		//создаем каталог-источник, если не существует и СreateSrc = true
		if _, err := os.Stat(fullSrcDir); err != nil {
			if os.IsNotExist(err) {
				if scangroup.СreateSrc == true {
					Info.Println("Создаём каталог(и) источник: ", fullSrcDir)
					err := os.MkdirAll(fullSrcDir, os.ModeDir)
					if err != nil {
						errorf("Не удалось создать каталог %s", fullSrcDir)
						Debug.Printf("Обработка каталога завершена %s", fullSrcDir)
						continue
					}
				} else {
					Debug.Printf("Каталог источник не существует %s. СreateSrc = false. Пропускаем каталог", fullSrcDir)
					Debug.Printf("Обработка каталога завершена %s", fullSrcDir)
					continue
				}
			} else {
				errorln(err)
				Debug.Printf("Обработка каталога завершена %s", fullSrcDir)
				continue
			}
		}

		//читаем содержимое каталога
		items, err := ioutil.ReadDir(fullSrcDir)
		if err != nil {
			errorln(err)
			Debug.Printf("Обработка каталога завершена %s", fullSrcDir)
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
		// Перезагружаем конфиг
		cfgTmp, err := reloadConfig(configFileName)
		if err != nil {
			if err != errNotModified {
				errorln("readconfig:", err)
			}
		} else {
			Info.Println("Перезагружаем конфигурационный файл")
			cfg = cfgTmp
			initLogger(cfg)
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
	//инициализируем логгеры
	if err := initLogger(cfg); err != nil {
		log.Fatalln(err)
	}
	Info.Printf("Версия: %s от %s\n", version, buildtime)
	tokens = make(chan struct{}, cfg.MaxScanThreads)
	//запускаем цикл сканирования каталогов
	go scanLoop(cfg)

	//запускаем горутину, которая сохраняет статистику в файл
	savestatchan = SaveStatLoop(tosserstat)

	if cfg.EnableHTTP {
		go runHTTP(cfg)
	}

	//ожидаем завершение программы по Ctrl-C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	signal.Notify(sigChan, syscall.SIGTERM)

	for {
		select {
		case <-sigChan:
			errorln("CTRL-C: Завершаю работу.")
			tosserstat.save(statfile)
			return
		}
	}
}
