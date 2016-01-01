[![Build Status](https://travis-ci.org/zaz600/gotosser.svg?branch=master)](https://travis-ci.org/zaz600/gotosser)

# Описание
Исходный текст к статье [Пишем "тоссер" на Go](https://goo.gl/2fP3bO)    
Программа предназначена для копирования файлов из одних каталогов в другие.

Возможности: 

* Хранение настроек в файле.
* Перемещение файлов между локальными каталогами и/или сетевыми дисками. 
* Одновременная обработка более одной пары каталогов источник-назначение, то есть работа в несколько потоков.
* Наличие в настройках вариантов выбора действия, если файл в конечной папке уже существует. Например, перезапись, пропуск.
* Возможность задать для одной сканируемой папки несколько правил: какие файлы искать и в какой каталог их перемещать.
* Возможность задавать списки исключений для файлов.
* Просмотр статистики работы через веб-браузер.

# Настройка
Чтобы настроить программу, необходимо переименовать файл gotosser.yaml.example в gotosser.yaml и там выставить нужные значения.  
Параметры там описаны.

# Сборка
Чтобы сделать сборку с номером версии, необходимо скопировать файлы build-debug.bat, build-release.bat из папки build\win в корень и настроить эти файлы там под себя.