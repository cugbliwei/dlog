package dlog

import (
	"fmt"
	"io"
	"log"
	"net/smtp"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Logger struct {
	mu        sync.Mutex
	date      string
	filename  string
	hourAgo   string
	hFilename string
	file      *os.File
	out       io.Writer
}

func time33(s string) int64 {
	var ret int64
	for _, c := range []byte(s) {
		ret *= 33
		ret += int64(c)
	}
	if ret > 0 {
		return ret
	}
	return -1 * ret
}

func (l *Logger) header(tm time.Time, file string, line int, s string) string {
	newPath := ""
	paths := strings.Split(file, "/")
	length := len(paths)
	if length > 1 {
		newPath = paths[length-2] + "/" + paths[length-1]
	} else if length == 1 {
		newPath = paths[0]
	}
	return fmt.Sprintf("%s %s line %d ", tm.Format("2006-01-02 15:04:05"), newPath, line)
}

func (l *Logger) Output(calldepth int, s string, email bool) error {
	now := time.Now() // get this early.
	var file string
	var line int
	l.mu.Lock()
	defer l.mu.Unlock()
	var ok bool
	_, file, line, ok = runtime.Caller(calldepth)
	if !ok {
		file = "???"
		line = 0
	}

	head := l.header(now, file, line, s)
	buf := make([]byte, 0, len(head))
	buf = append(buf, head...)
	for _, c := range []byte(s) {
		if c != '\n' {
			buf = append(buf, c)
		} else {
			buf = append(buf, '\n')
			_, err := l.out.Write(buf)
			if err != nil {
				return err
			}
			_, _ = l.file.Write(buf)

			buf = buf[:0]
			buf = append(buf, head...)
		}
	}
	if len(buf) > len(head) {
		buf = append(buf, '\n')
		_, err := l.out.Write(buf)
		if err != nil {
			return err
		}
		_, _ = l.file.Write(buf)
	}
	if email {
		target := head + s
		emailCache.lock.Lock()
		flag := true
		for _, cache := range emailCache.cache {
			rate := Similarity(target, cache)
			if rate > 0.9 {
				flag = false
			}
		}

		emailCache.cache = append(emailCache.cache, target)
		if flag {
			SendMail("程序实时报警", target)
		}

		emailCache.lock.Unlock()
	}
	return nil
}

func SendMail(subject, body string) {
	sendToMail("host:port", "email@xxx", "", "email@xxx;email@xxx", "plain", subject, body)
}

//smtp服务发送邮件, mailType表示邮件格式是普通文件还是html或其他
func sendToMail(host, user, password, to, mailType, subject, body string) {
	hp := strings.Split(host, ":")
	auth := smtp.PlainAuth("", user, password, hp[0])
	content_type := "Content-Type: text/" + mailType + "; charset=UTF-8"
	msg := []byte("To: " + to + "\r\nFrom: " + user + "\r\nSubject: " + subject + "\r\n" + content_type + "\r\n\r\n" + body)
	send_to := strings.Split(to, ";")
	err := smtp.SendMail(host, auth, user, send_to, msg)
	if err != nil {
		log.Println("SendMail error: ", err)
	}
}
