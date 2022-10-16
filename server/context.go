package server

import (
	"encoding/json"
	"fmt"
	"github.com/Nixson/annotation"
	"github.com/Nixson/environment"
	"github.com/Nixson/http/session"
	"github.com/Nixson/logger"
	"github.com/google/uuid"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type ContextInterface interface {
	SetContext(c *Context)
}

type Context struct {
	Request  *http.Request
	Response http.ResponseWriter
	Session  *session.Session
	Path     string
	Params   map[string]string
	Method   string
	Data     io.ReadCloser
	Query    url.Values
	Handle   *Info
}

func (c *Context) Access(access uint) bool {
	return access <= c.Session.Access
}
func getInfo(method, path string) (*Info, error) {
	nf, ok := contextMap[method+"."+path]
	if ok {
		return &nf, nil
	}
	nf, ok = contextMap["ALL."+path]
	if ok {
		return &nf, nil
	}
	return nil, fmt.Errorf("404")
}
func getRegInfo(method, path string) (*Info, map[string]string, error) {
	for _, info := range contextList {
		if !info.IsReg {
			continue
		}
		if info.Method != "ALL" && info.Method != method {
			continue
		}
		if path[len(path)-1:] != "/" {
			path = path + "/"
		}
		if info.Reg.MatchString(path) {
			find := info.Reg.FindAllStringSubmatch(path, -1)
			paramMap := make(map[string]string)
			for i := 0; i < len(find); i++ {
				paramMap[info.Params[i]] = find[i][1]
			}
			return &info, paramMap, nil
		}
	}
	return nil, nil, fmt.Errorf("404")
}
func (c *Context) IsGranted() bool {
	inf, err := getInfo(c.Method, c.Path)
	if err != nil {
		var mp map[string]string
		inf, mp, err = getRegInfo(c.Method, c.Path)
		if err != nil {
			c.Error(http.StatusNotFound, "URL not found")
			return false
		}
		for ks, vs := range mp {
			c.Params[ks] = vs
		}
	}
	c.Handle = inf
	if environment.GetEnv().GetBool("security.enable") {
		if len(c.Request.Header["Authorization"]) < 1 {
			if inf.Access == "all" {
				c.Session = &session.Session{
					User:  session.User{Access: 1000},
					Hash:  uuid.New().String(),
					Dtime: time.Now().Unix() + session.Dtime,
				}
				return true
			}
			c.Error(http.StatusUnauthorized, "failed header Authorization")
			return false
		}
		authorizationHeader := c.Request.Header["Authorization"][0]
		token := strings.TrimPrefix(authorizationHeader, "Bearer ")
		sess := session.GetSession(token)
		if sess != nil {
			c.Session = sess
			return true
		}
		c.Error(http.StatusUnauthorized, "failed header Authorization")
		return false
	}
	c.Session = &session.Session{
		User:  session.User{Access: 1000},
		Hash:  uuid.New().String(),
		Dtime: time.Now().Unix() + session.Dtime,
	}
	return true
}

type Info struct {
	Index   int
	Method  string
	Access  string
	Path    string
	Context *ContextInterface
	Params  []string
	Reg     *regexp.Regexp
	IsReg   bool
}

var contextMap = make(map[string]Info)
var contextList = make([]Info, 0)

func InitController(name string, controller *ContextInterface) {
	annotationList := annotation.Get("controller")
	annotationMap := make(map[string]annotation.Element)
	for _, annotationMapEl := range annotationList {
		if annotationMapEl.StructName == name {
			for _, child := range annotationMapEl.Children {
				annotationMap[child.StructName] = child
			}
		}
	}
	_struct := reflect.TypeOf(*controller)
	for index := 0; index < _struct.NumMethod(); index++ {
		_method := _struct.Method(index)
		annotationMapEl, ok := annotationMap[_method.Name]
		if !ok {
			continue
		}
		access, ok := annotationMapEl.Parameters["access"]
		if !ok {
			access = "auth"
		}
		aType := "ALL"
		switch annotationMapEl.Type {
		case "GetRequest":
			aType = "GET"
		case "PutRequest":
			aType = "PUT"
		case "PostRequest":
			aType = "POST"
		case "DeleteRequest":
			aType = "DELETE"
		}
		inf := Info{
			Index:   _method.Index,
			Method:  aType,
			Access:  access,
			Context: controller,
			IsReg:   false,
		}
		path, hasPath := annotationMapEl.Parameters["path"]
		if !hasPath {
			path = _method.Name
		}
		inf.Path = path
		if strings.Contains(path, "{") {
			find := isEnv.FindAllStringSubmatch(path, -1)
			regPath := path
			regParams := make([]string, 0)
			if len(find) > 0 {
				for i := 0; i < len(find); i++ {
					regParams = append(regParams, find[i][1])
					regPath = strings.ReplaceAll(regPath, "{"+find[i][1]+"}/", `(.*?)\/`)
					regPath = strings.ReplaceAll(regPath, "{"+find[i][1]+"}", `(.*?)\/`)
				}
				inf.Params = regParams
				inf.Reg = regexp.MustCompile(regPath)
				inf.IsReg = true
			}
		}
		contextMap[aType+"."+path] = inf
		contextList = append(contextList, inf)
	}
	sort.SliceStable(contextList, func(i, j int) bool {
		return len(contextList[i].Path) > len(contextList[j].Path)
	})
}

var isEnv = regexp.MustCompile(`\{(.*?)\}`)

func (c *Context) Call() {
	in := make([]reflect.Value, 0)
	info := c.Handle
	var hdl ContextInterface
	hdl = *info.Context
	hdl.SetContext(c)
	reflect.ValueOf(hdl).Method(info.Index).Call(in)
}
func (c *Context) Write(iface interface{}) {
	marshal, _ := json.Marshal(iface)
	_, err := c.Response.Write(marshal)
	if err != nil {
		logger.Println(err.Error())
	}
}
func (c *Context) Error(status int, iface interface{}) {
	c.Response.WriteHeader(status)
	c.Write(iface)
}

func (c *Context) ParseUrl() {
	lst := strings.Split(c.Path, "/")
	serviceName := environment.GetEnv().Get("service.name")
	if lst[1] == serviceName {
		lst = lst[2:]
		c.Path = "/" + strings.Join(lst, "/")
		lst = strings.Split(c.Path, "/")
	}
	ignore := environment.GetEnv().Get("server.ignore")
	if len(ignore) < 1 {
		return
	}
	ignoreList := strings.Split(ignore, ",")
	find := false
	for _, sub := range ignoreList {
		sub = strings.TrimSpace(sub)
		if lst[1] == sub {
			lst = lst[2:]
			c.Path = "/" + strings.Join(lst, "/")
			lst = strings.Split(c.Path, "/")
			c.Params["element"] = sub
			find = true
		}
	}
	if !find {
		c.Params["element"] = serviceName
	}
}

func (c *Context) CheckStatic(env *environment.Env, path string) (string, bool) {
	if path == "" || path == "/" || path == "/index.html" {
		return "/" + env.GetString("template.main"), true
	}
	if path[0:1] == "/" {
		path = path[1:]
	}
	subs := strings.Split(path, "/")
	if subs[0] == "api" {
		return "", false
	}
	if _, err := strconv.Atoi(subs[0]); err == nil {
		subs = subs[1:]
		path = strings.Join(subs, "/")
		return "/" + path, true
	}
	return "/" + path, true
}

type TokenException struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func (t *TokenException) Marshal() string {
	marshal, err := json.Marshal(t)
	if err != nil {
		return ""
	}
	return string(marshal)
}
