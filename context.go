package julla

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/glog"
	"net/http"
	"net/url"
	"redigo/redis"
	"regexp"
	"strings"
	"text/template"
)

type (

	DataBase struct {
		postgresql *sql.DB
		redis  redis.Conn
	}

	Context struct {

		request  *http.Request

		responseWriter  http.ResponseWriter

		Pattern string

		mimicry map[string]string

		DB      DataBase

	}
	
	)

func (c *Context) UrlLink() string {

	return c.request.URL.String()
}

func (c *Context) UrlPath() string {

	return c.request.URL.Path
}

func (c *Context) WebSite() (website string) {

	scheme := "http://"

	if c.request.TLS != nil {
		scheme = "https://"
	}

	website = strings.Join([]string{ scheme, c.request.Host, c.request.RequestURI }, "")

	return
}

func (c *Context) Param(name string) string {

	return c.mimicry[name]
}

func (c *Context) ParamNames() (names []string) {

	for name, _ := range c.mimicry {

		names = append(names, name)
	}

	return
}

func (c *Context) QueryString(name string) string {

	urlForm, err := url.ParseQuery(c.request.URL.RawQuery)

	if err == nil && len(urlForm[name]) > 0 {

		return urlForm[name][0]
	}

	return ""
}

func (c *Context) FormString(name string) string {

	c.request.ParseForm()

	return c.request.PostFormValue(name)
}

func (c *Context) Validate(s, rex string) bool {

	re := regexp.MustCompile(rex)

	whether := re.MatchString(s)

	return whether

}

func (c *Context) Examine(e error, whether bool, code int, args ...string) {

	if whether == true {

		for _, v := range args{

			glog.Error(v)

		}
	}

	c.responseWriter.WriteHeader(code)

	c.responseWriter.Write([]byte(e.Error()))

}

func (c *Context) Error(code int, err error) error {

	if code < 400 || code > 600 {

		return errors.New("invalid error status code")

	}

	c.responseWriter.WriteHeader(code)

	c.responseWriter.Write([]byte(err.Error()))

	return nil
}

func (c *Context) String(code int, s string) error {

	c.responseWriter.Header().Set("Content-Type", "text/plain; charset=UTF-8")

	c.responseWriter.WriteHeader(code)

	c.responseWriter.Write([]byte(s))

	return nil
}

func (c *Context) HTML(code int, html string) error {

	c.responseWriter.Header().Set("Content-Type", "text/html; charset=UTF-8")

	c.responseWriter.WriteHeader(code)

	c.responseWriter.Write([]byte(html))

	return nil
}

func (c *Context) JSON(code int, data H) error {

	c.responseWriter.Header().Set("Content-Type", "application/json; charset=UTF-8")

	c.responseWriter.WriteHeader(code)

	err := json.NewEncoder(c.responseWriter).Encode(data)

	if  err != nil {

		return err
	}

	return nil
}

func (c *Context) Render(data interface{}, filenames ...string) error {

	tpl, err := template.ParseFiles(filenames...)

	if err != nil {

		return err
	}

	tpl.Execute(c.responseWriter, data)

	return nil

}

func (c *Context) RenderTPL(name string, data interface{}, filenames ...string) error {

	tpl, err := template.ParseFiles(filenames...)

	if err != nil {

		return err
	}

	tpl.ExecuteTemplate(c.responseWriter, name, data)

	return nil

}

func (c *Context) NoContent(code int) error {

	c.responseWriter.WriteHeader(code)

	return nil

}

func (c *Context) Redirect(code int, url string) error {

	if code < 300 || code > 308 {

		return errors.New("invalid redirect status code")

	}

	c.responseWriter.Header().Set("Location", url)

	c.responseWriter.WriteHeader(code)

	return nil
}


func (c *Context) Equipment() string {

	var utensils = make(map[string][]string)

	var (
		mobile  = []string{
			//iPhone
			`(?U)Mozilla.*(?:5\.0|4\.0).*iPhone.*(?:CPU|CPU iPhone OS).*like Mac OS X`,
		}
		computer = []string{
			//Windows xp -- 10
			`Mozilla.*(?:5\.0|4\.0).*Windows NT.*(?:5\.1|5\.2|6\.0|6\.1|6\.2|6\.3|6\.4|10\.0)`,
		}
		tabletPC = []string{
			//iPad
			`(?U)Mozilla.*(?:5\.0|4\.0).*iPad.*CPU OS.*like Mac OS X.*Mobile`,
		}
	)

	utensils["Mobile"] = mobile
	
	utensils["Computer"] = computer
	
	utensils["TabletPC"] = tabletPC

	index := len(mobile) + len(tabletPC) + len(computer)

	ch := make(chan string, index)

	for name, res := range utensils {

		for _, value := range res{

			go equipment(c.request.Header.Get("User-Agent") ,name, value, ch)

		}

	}

	eName := "Computer"

	for i := 0; i < index; i ++ {

		if i == index {

			close(ch)

		}

		e := <- ch

		if e == "nil" {

			continue

		}else {

			fmt.Println(e)

			eName = e
		}
	}

	return eName
}

func (c *Context) System() string {

	return ""
}

func (c *Context) Device() string {

	return ""
}

func (c *Context) Browser() string {

	return ""
}

func equipment(ua, name, re string, c chan string)  {

	rex := regexp.MustCompile(re)

	whether := rex.MatchString(ua)

	if whether {

		c <- name

	}else {

		c <- "nil"
	}

}


