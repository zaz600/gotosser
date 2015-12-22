package main

import (
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

var (
	//контролируем число потоков processScanGroup
	tokens = make(chan struct{}, 8)
)

func processItems(items []os.FileInfo, fullSrcDir string) {
	for _, item := range items {
		// обрабатываем только файлы. Не каталоги, символические ссылки и т.п.
		if !item.Mode().IsRegular() {
			continue
		}
		srcFile := item.Name()
		fullSrcFilePath := filepath.Join(fullSrcDir, srcFile)
		log.Println(fullSrcFilePath)
	}
}

func processScanGroup(scangroup ScanGroup) {
	//освобождаем токен после завершения процедуры
	defer func() { <-tokens }()

	for _, srcDir := range scangroup.SrcDirs {
		//разворачиваем маску времени (%Y%m%d и т.п.), если есть в пути
		fullSrcDir := strftime.Format(srcDir, time.Now())
		abspath, err := filepath.Abs(fullSrcDir)
		if err != nil {
			log.Println("Ошибка вычисления абсолютного пути", srcDir, err)
			continue
		}
		fullSrcDir = abspath
		log.Println("Сканируем каталог", fullSrcDir)
		//читаем содержимое каталога
		items, err := ioutil.ReadDir(fullSrcDir)
		if err != nil {
			log.Println(err)
			log.Printf("Обработка каталога завершена %s", fullSrcDir)
			continue
		}
		//обрабатываем файлы
		processItems(items, fullSrcDir)
	}
}

// scanLoop просматривает конфиг и для каждого каталога-источника
// запускает горутину processScanGroup
func scanLoop(cfg *Config) {
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
	cfg, err := reloadConfig(configFileName)
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
