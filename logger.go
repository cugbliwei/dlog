package dlog

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wsxiaoys/terminal/color"
)

const (
	FATAL = 0
	PANIC = 1
	ERROR = 2
	WARN  = 3
	INFO  = 4
	DEBUG = 5
)

var Level int64 = INFO
var lg *Logger

type EmailCache struct {
	lock  *sync.RWMutex
	cache []string
}

var emailCache *EmailCache

func init() {
	lg = &Logger{out: os.Stdout}
	emailCache = &EmailCache{lock: &sync.RWMutex{}}
	initFile()

	go func() {
		if strings.Contains(runtime.GOARCH, "arm") || strings.Contains(runtime.GOOS, "android") {
			return
		}

		ticker := time.NewTicker(10 * time.Minute)
		last := 100
		for now := range ticker.C {
			sendEmail()

			_, minute, _ := now.Clock()
			if last != 100 && last != minute {
				continue
			}

			if minute < 20 {
				last = minute
				initFile()

				//upload 1 hour ago log file to hdfs
				tmpFile, err := os.Open(lg.hFilename)
				if err != nil {
					log.Println("open 1 hour ago log file error:", err)
					continue
				}

				for i := 0; i < 3; i++ {
					body, err := Upload("http://10.130.64.140:8088/hdfs/put", tmpFile, lg.hFilename, lg.hourAgo)
					if err != nil || string(body) == "false" {
						log.Println("fail to upload file to hdfs")
						continue
					} else {
						break
					}
				}
				tmpFile.Close()

			}
		}
	}()
}

func sendEmail() {
	var body string
	emailCache.lock.Lock()
	if len(emailCache.cache) > 0 {
		for _, email := range emailCache.cache {
			body += email + "\n\n"
		}

		emailCache.cache = []string{}
	}
	emailCache.lock.Unlock()

	if len(body) > 0 {
		SendMail("程序缓存报警", body) //如果body太大，可能会发送失败
	}
}

func initFile() {
	lg.mu.Lock()
	defer lg.mu.Unlock()

	lg.file.Close()

	lg.date = time.Now().Format("2006010215")
	h, _ := time.ParseDuration("-1h")
	lg.hourAgo = time.Now().Add(h).Format("2006010215")
	removeHour, _ := time.ParseDuration("-6h")
	removeHourAgo := time.Now().Add(removeHour).Format("2006010215")
	path, err := os.Getwd()
	if err != nil {
		log.Println("get path err: %v", err)
		return
	}

	paths := strings.Split(path, "/")
	filename := paths[len(paths)-1]
	if len(filename) == 0 {
		if len(paths)-2 >= 0 && len(paths[len(paths)-2]) > 0 {
			filename = paths[len(paths)-2]
		} else {
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			filename = "default" + strconv.Itoa(r.Intn(10000))
		}
	}

	lg.filename = filename + "_" + lg.date + ".log"
	lg.hFilename = filename + "_" + lg.hourAgo + ".log"
	removeFilename := filename + "_" + removeHourAgo + ".log"
	os.Remove(removeFilename)

	lg.file, err = os.OpenFile(filename+"_"+lg.date+".log", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		log.Println("open file error: %v", err)
		return
	}
}

func Upload(link string, file *os.File, filename, date string) ([]byte, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		log.Println("create form file error:", err)
		return nil, err
	}
	_, err = io.Copy(part, file)
	if err != nil {
		log.Println("copy file to part error:", err)
		return nil, err
	}

	writer.WriteField("filename", filename)
	writer.WriteField("date", date)
	writer.Close()

	req, err := http.NewRequest("POST", link, body)
	if err != nil {
		log.Println("new post request error:", err)
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	c := &http.Client{}
	resp, err := c.Do(req)
	if err != nil {
		log.Println("do request error:", err)
		return nil, err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("read all from resp body error:", err)
		return nil, err
	}
	return b, nil
}

func SetLogFile(filename string) {
	lg.file, _ = os.OpenFile(filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
}

func CloseLogFile() {
	lg.file.Close()
}

func Info(format string, v ...interface{}) {
	if Level >= INFO {
		lg.Output(2, fmt.Sprintf("[INFO] "+format, v...), false)
	}
}

func Println(v ...interface{}) {
	lg.Output(2, fmt.Sprint(v...), false)
}

func Warn(format string, v ...interface{}) {
	if Level >= WARN {
		escapeCode := color.Colorize("y")
		io.WriteString(lg.out, escapeCode)
		io.WriteString(lg.file, escapeCode)
		lg.Output(2, color.Sprintf("[WARN] "+format, v...), false)
	}
}

func Error(format string, v ...interface{}) {
	if Level >= ERROR {
		escapeCode := color.Colorize("r")
		io.WriteString(lg.out, escapeCode)
		io.WriteString(lg.file, escapeCode)
		lg.Output(2, color.Sprintf("[ERROR] "+format, v...), true)
	}
}

func ErrorN(n int, format string, v ...interface{}) {
	if Level >= ERROR {
		lg.Output(2+n, fmt.Sprintf("[ERROR] "+format, v...), true)
	}
}

func Debug(format string, v ...interface{}) {
	if Level >= DEBUG {
		lg.Output(2, fmt.Sprintf("[DEBUG] "+format, v...), false)
	}
}

func Fatal(format string, v ...interface{}) {
	if Level >= FATAL {
		lg.Output(2, fmt.Sprintf("[FATAL] "+format, v...), true)
		os.Exit(1)
	}
}

func Fatalln(v ...interface{}) {
	if Level >= FATAL {
		lg.Output(2, fmt.Sprint(v...), true)
		os.Exit(1)
	}
}

func Panic(format string, v ...interface{}) {
	if Level >= PANIC {
		s := fmt.Sprintf("[PANIC] "+format, v...)
		lg.Output(2, s, true)
		panic(s)
	}
}
