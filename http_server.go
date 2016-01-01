package main

import (
	"html/template"
	"net/http"
	"sort"
	"time"

	"github.com/hhkbp2/go-strftime"
)

var (
	//шаблон вывода статистики работы
	tmplStat *template.Template
)

//для передачи в шаблон
type dirStat struct {
	Dir  string
	Stat *dirStatInfo
}

func showstat(w http.ResponseWriter, r *http.Request) {
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

	tmplStat.Execute(w, &struct {
		StatDate        string
		Version         string
		VersionDate     string
		DirStatInfoList []*dirStat
		ErrorHistory    []errorHistoryItem
	}{now, version, buildtime, dirStatInfoList, errorHistory.Get()})
}

func runHTTP(cfg *Config) {
	log.Infoln("Запуск веб-сервера на", cfg.Listen)
	//загружаем шаблон
	var err error
	tmplStat, err = template.ParseFiles("templates/stat.tmpl")
	if err != nil {
		log.Fatal(err)
	}
	//запускаем сервер
	http.HandleFunc("/", showstat)
	err = http.ListenAndServe(cfg.Listen, nil)
	if err != nil {
		log.Fatal(err)
	}
}
