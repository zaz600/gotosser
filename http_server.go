package main

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"time"

	"github.com/hhkbp2/go-strftime"
)

const template = `<html><head><title>Тоссер: статистика</title>
	<meta http-equiv="refresh" content="30">
	<link rel="shortcut icon" href="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAYAAAAf8/9hAAAABHNCSVQICAgIfAhkiAAAAAlwSFlzAAALEwAACxMBAJqcGAAAANtJREFUOI290rFKgzEUhuEHWlBHZ8FL0KWLQ9cOjqWCCN6DY+/AwdXFxb1LwZuwFLsoKAjODoIgpUOxVh3+UMLfpC2IfeGQwJfzJTnn8A/s4AJPGOMTjzjHHu7RzyWfYISfFWKOY3wHsYMDbGIDTUwXGexGN58lzB+WveAqujlFr5R8Wz7wHoT9jEGWalhf8KWo8FoYWNDKZdQU331LucaF6mUMboJ+WRbKbbpLJLeDNlRMbNZgiiNshaijG2mt1NNWGd9hMJ5RifaHeEUDH9gOMcEzrnHqD9VP8gs5H1UaDpmL+AAAAABJRU5ErkJggg=="/>
	</head>
	<style type="text/css">
	body {
		 font-family: Helvetica, Sans-Serif; 
	}
   TABLE {
    width: 800px; /* Ширина таблицы */
    border-collapse: collapse; /* Убираем двойные линии между ячейками */
   }
   TD, TH {
    padding: 3px; /* Поля вокруг содержимого таблицы */
    border: 1px solid black; /* Параметры рамки */
   }
   TH {
    background: #b0e0e6; /* Цвет фона */
   }
   ul {
    list-style: square;
	padding: 10px; 
   }
  </style>
	<body><h2>Статистика тоссера за %s</h2>
	<div> Версия: %s от %s</div>
	<table>%s</table>
	<ul>%s</ul>
	</body>
	</html>`

func showstat(w http.ResponseWriter, r *http.Request) {

	now := strftime.Format("%Y-%m-%d", time.Now())
	//получаем статистику за дату
	val, _ := tosserstat.Dates[now]

	//сортируем папки по имени
	var keys []string
	for k := range val {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	trs := `<tr><td align="center">Каталог</td>
	<td align="center" style="width: 100px;">Кол-во переданных файлов</td>
	<td align="center" style="width: 100px;">Суммарный размер</td>
	<td align="center" style="width: 100px;">Последняя передача</td>
	</tr>`

	for _, dir := range keys {
		//статистика для каталога-источника
		dirStat, ok := val[dir]
		LastProcessingDateStr := "-"
		if ok {
			LastProcessingDateStr = strftime.Format("%H:%M:%S", time.Unix(dirStat.LastProcessingDate, 0))
		}
		sizeStr, err := convertSize(dirStat.TotalSize)
		if err != nil {
			sizeStr = "-"
		}
		trs += fmt.Sprintf("<tr><td>%s</td><td align=\"right\">%d</td><td align=\"right\">%s</td><td align=\"right\">%s</td></tr>\n", dir, dirStat.Count, sizeStr, LastProcessingDateStr)
	}

	li := ""
	for _, e := range errorHistory {
		li += "<li>" + e + "</li>"
	}
	fmt.Fprintf(w, template, now, version, buildtime, trs, li)
}

func runHTTP(cfg *Config) {
	Info.Println("Запуск веб-сервера на", cfg.Listen)
	http.HandleFunc("/", showstat)
	err := http.ListenAndServe(cfg.Listen, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func convertSize(size int64) (string, error) {
	if size == 0 {
		return "0B", nil
	}
	sizeName := []string{"B", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"}
	i := int(math.Log2(float64(size)) / 10)
	humanSize := fmt.Sprintf("%d%s", size/int64(math.Pow(1024, float64(i))), sizeName[i])
	return humanSize, nil
}
