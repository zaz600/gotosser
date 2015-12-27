package main

import (
	"html/template"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/hhkbp2/go-strftime"
)

var (
	//шаблон вывода статистики работы
	tmplStat *template.Template
	loadOnce sync.Once
)

//для передачи в шаблон
type dirStat struct {
	Dir  string
	Stat *dirStatInfo
}

func showstat(w http.ResponseWriter, r *http.Request) {
	loadOnce.Do(func() {
		var err error
		tmplStat, err = template.ParseFiles("templates/stat.tmpl")
		if err != nil {
			log.Fatal(err)
		}
	})

	now := strftime.Format("%Y-%m-%d", time.Now())
	//получаем статистику за дату
	dayStat, _ := tosserstat.Dates[now]

	//сортируем папки по имени
	var dirs []string
	for dir := range dayStat {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)

	//заполняем отсортированнымми данными слайс
	var dirStatInfoList []*dirStat
	for _, dir := range dirs {
		dirStatInfoList = append(dirStatInfoList, &dirStat{dir, dayStat[dir]})
	}
	tmplStat.Execute(w, map[string]interface{}{"StatDate": now, "Version": version, "VersionDate": buildtime, "dirStatInfoList": dirStatInfoList, "errorHistory": errorHistory})
}

func runHTTP(cfg *Config) {
	Info.Println("Запуск веб-сервера на", cfg.Listen)
	http.HandleFunc("/", showstat)
	err := http.ListenAndServe(cfg.Listen, nil)
	if err != nil {
		log.Fatal(err)
	}
}
