// Copyright 2020 Changkun Ou. All rights reserved.
// Use of this source code is governed by a GNU GPL-3.0
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net/smtp"
	"os"
	"os/signal"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// build info
var (
	Version     = "v0.1.0"
	GoVersion   = runtime.Version()
	errCanceled = errors.New("action canceled")
	homedir     string
	pathConf    = ".tli_config"
	pathHist    = ".tli_history"
)

var tli tliConf

func checkhome() {
	user, err := user.Current()
	if err != nil {
		log.Fatalf(`cannot find home directory, err: %v`, err)
	}
	homedir = user.HomeDir
	if len(homedir) == 0 {
		log.Fatalf(`cannot find home directory, err: %v`, err)
	}
}

// mconf contains all necessary information for sending a thing
// to the things' inbox.
type tliConf struct {
	SMTPHost   string `yaml:"smtp_host"`
	SMTPPort   string `yaml:"smtp_port"`
	Avatar     string `yaml:"avatar"`
	EmailAddr  string `yaml:"email_addr"`
	Username   string `yaml:"username"`
	Password   string `yaml:"password"`
	ThingsAddr string `yaml:"things_addr"`
}

func (c *tliConf) parse() {
	f := os.Getenv("TLI_CONF")
	d, err := ioutil.ReadFile(f)
	if err != nil {
		// try again with default setting
		d, err = ioutil.ReadFile(homedir + "/" + pathConf)
		if err != nil {
			log.Fatalf(`cannot find tli config, err: %v
try: tli init`, err)
		}
	}
	err = yaml.Unmarshal(d, c)
	if err != nil {
		log.Fatalf("cannot parse tli config, err: %v", err)
	}
}

func (c *tliConf) save() {
	checkhome()
	data, err := yaml.Marshal(c)
	if err != nil {
		log.Fatalf("cannot save your data, err: %v", err)
	}

	f, err := os.OpenFile(homedir+"/"+pathConf,
		os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return
	}
	defer f.Close()

	all := []byte("---\n")
	all = append(all, data...)
	if _, err := f.Write(all); err != nil {
		return
	}
}

func (c *tliConf) sendInbox(title, body string) error {
	// Text in an encoded-word in a display-name must not contain certain
	// characters like quotes or parentheses (see RFC 2047 section 5.3).
	// When this is the case encode the title using base64 encoding.
	if strings.ContainsAny(title, "\"#$%&'(),.:;<>@[]^`{|}~") {
		title = mime.BEncoding.Encode("utf-8", title)
	} else {
		title = mime.QEncoding.Encode("utf-8", title)
	}

	err := smtp.SendMail(
		c.SMTPHost+":"+c.SMTPPort,
		smtp.PlainAuth("", c.Username, c.Password, c.SMTPHost),
		c.EmailAddr, []string{c.ThingsAddr},
		// rfc822format, see:
		// https://docs.microsoft.com/en-us/previous-versions/office/developer/exchange-server-2010/aa493918(v=exchg.140)
		[]byte(fmt.Sprintf("Subject: %s\r\nFrom: %s <%s>\r\nTo: %s\r\nContent-Type: text/plain; charset=\"UTF-8\"\r\n%s",
			// Content-Type: text/plain; charset=utf-8; format=flowed
			// Content-Transfer-Encoding: 7bit
			// Content-Language: en-US
			title,
			c.Avatar, c.EmailAddr, c.ThingsAddr,
			body,
		)))
	if err != nil {
		return err
	}
	return nil
}

type tliTODO struct {
	// title/body should shorter than 72 for a line
	title string
	body  []string
}

func newtliTODO(title string) (*tliTODO, error) {
	a := &tliTODO{
		title: title,
	}
	if !a.waitBody() {
		return nil, errCanceled
	}
	return a, nil
}

func (a *tliTODO) waitBody() bool {
	s := bufio.NewScanner(os.Stdin)
	fmt.Println("(Enter an empty line to complete; Ctrl+C/Ctrl+D to cancel)")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	line := make(chan string, 1)
	go func() {
		for {
			fmt.Print("> ")
			if !s.Scan() {
				sigCh <- os.Interrupt
				return
			}
			l := s.Text()
			if len(l) == 0 {
				line <- ""
				return
			}
			line <- l
		}
	}()

	for {
		select {
		case <-sigCh:
			return false
		case l := <-line:
			if len(l) == 0 {
				return true
			}
			a.body = append(a.body, l)
		}
	}
}

const maxlen = 2000

func (a *tliTODO) Range(f func(string, string)) {
	whole := strings.Join(a.body, "\n")

	if len(whole) < maxlen {
		f(a.title, whole)
		return
	}

	count := 1
	for i := 0; i < len(whole); i += maxlen {
		f(a.title+fmt.Sprintf(" (%d)", count), whole[i:min(i+maxlen, len(whole))])
		count++
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type record struct {
	Time  time.Time `yaml:"time"`
	Title string    `yaml:"title"`
	Body  string    `yaml:"body"`
}

func (a *tliTODO) Save() {
	var err error
	defer func() {
		if err != nil {
			log.Fatalf("cannot save your TODO, err: %v", err)
		}
	}()

	r := record{
		Time:  time.Now().UTC(),
		Title: a.title,
		Body:  strings.Join(a.body, "\n"),
	}

	data, err := yaml.Marshal(r)
	if err != nil {
		return
	}

	f, err := os.OpenFile(homedir+"/"+pathHist,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()

	all := []byte("---\n")
	all = append(all, data...)
	all = append(all, []byte("\n")...)
	if _, err := f.Write(all); err != nil {
		return
	}
}

func main() {
	log.SetPrefix("tli: ")
	log.SetFlags(log.Ldate | log.Ltime)

	var cmdInit = &cobra.Command{
		Use:   "init",
		Short: "initialize tli settings",
		Long:  `init will ask you several informations for setting up the configuration.`,
		Args:  cobra.ExactArgs(0),
		Run:   initCmd,
	}

	var cmdLog = &cobra.Command{
		Use:   "log [number]",
		Short: "print logs",
		Long:  `log will print the specified number of items`,
		Args:  cobra.MinimumNArgs(0),
		Run:   logCmd,
	}

	var cmdTodo = &cobra.Command{
		Use:                   "todo [title]",
		Short:                 "create a todo and send it to the Things' Inbox",
		Long:                  "create a todo and send it to the Things' Inbox.",
		Args:                  cobra.MinimumNArgs(1),
		DisableFlagsInUseLine: true,
		Run:                   todoCmd,
	}

	var rootCmd = &cobra.Command{
		Use:   "tli",
		Short: "A Things CLI for Linux support.",
		Long: fmt.Sprintf(`
tli is a Linux CLI that supports send items to the Things' Inbox safely.
Specifically, it will save the sent TODO log to prevent if you send too
much to the Things' server. tli also checks your content to make sure your
inputs won't be too large so that the content is not silently truncated
by Things.

Version:   %v
GoVersion: %v
`, Version, GoVersion),
	}
	rootCmd.AddCommand(cmdInit)
	rootCmd.AddCommand(cmdLog)
	rootCmd.AddCommand(cmdTodo)
	rootCmd.Execute()
}

func initCmd(cmd *cobra.Command, args []string) {
	s := bufio.NewScanner(os.Stdin)
	log.Printf("SMTP Host Address: ")
	if !s.Scan() {
		log.Println("init was canceled.")
		return
	}
	info := s.Text()
	tli.SMTPHost = info

	log.Printf("SMTP Host Port: ")
	if !s.Scan() {
		log.Println("init was canceled.")
		return
	}
	info = s.Text()
	tli.SMTPPort = info

	log.Printf("Avatar (Your Name): ")
	if !s.Scan() {
		log.Println("init was canceled.")
		return
	}
	info = s.Text()
	tli.Avatar = info

	log.Printf("Email (Your Email): ")
	if !s.Scan() {
		log.Println("init was canceled.")
		return
	}
	info = s.Text()
	tli.EmailAddr = info

	log.Printf("Username (Your Email's Username): ")
	if !s.Scan() {
		log.Println("init was canceled.")
		return
	}
	info = s.Text()
	tli.Username = info

	log.Printf("Password (Your Email's Password): ")
	if !s.Scan() {
		log.Println("init was canceled.")
		return
	}
	info = s.Text()
	tli.Password = info

	log.Printf("Things 3 Email Address: ")
	if !s.Scan() {
		log.Println("init was canceled.")
		return
	}
	info = s.Text()
	tli.ThingsAddr = info
	tli.save()
	log.Println("You can start using tli now :)")
}

func logCmd(cmd *cobra.Command, args []string) {
	checkhome()
	tli.parse()

	var (
		n   int
		err error
	)
	if len(args) != 0 {
		n, err = strconv.Atoi(args[0])
		if err != nil {
			log.Fatalf("invalid argument, please input a number.")
		}
	}

	data, err := ioutil.ReadFile(homedir + "/" + pathHist)
	if err != nil {
		log.Fatalf("cannot read ~/.tli_history, try store something first.")
	}

	d := yaml.NewDecoder(bytes.NewReader(data))

	rs := []record{}
	for {
		var r record
		err = d.Decode(&r)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalf("corrupted ~/.tli_history file, err: %v", err)
		}
		rs = append(rs, r)
	}

	if n == 0 {
		n = len(rs)
	}
	for i := 1; i <= n; i++ {
		out, _ := yaml.Marshal(rs[len(rs)-i])
		log.Println(string(out))
	}
}

func todoCmd(cmd *cobra.Command, args []string) {
	checkhome()
	tli.parse()

	title := strings.Join(args, " ")
	a, err := newtliTODO(title)
	if errors.Is(err, errCanceled) {
		log.Println("TODO is canceled.")
		return
	}
	a.Save()
	a.Range(func(title, body string) {
		// TODO: think more about here, send email is considered slow
		// we could submit a task to a deamon process and let it send
		// the email in a background. The daemon process can also help
		// us retry the send.
		for i := 0; i < 5; i++ {
			err := tli.sendInbox(title, body)
			if err == nil {
				return
			}
			log.Printf("failed to send inbox, err: %v. Retry...", err)
		}
	})
	log.Println("DONE!")
}
