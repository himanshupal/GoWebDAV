package main

import (
	_ "embed"
	"fmt"
	"golang.org/x/net/context"
	"golang.org/x/net/webdav"
	"net/http"
	"net/url"
	"os"
	"strings"

	"GoWebDAV/model"

	"github.com/spf13/viper"
)

//go:embed static/index.html
var indexHTML string

func main() {
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.SetConfigName("config")

	AppConfig.Load()
	fmt.Print("AppConfig.dav ")
	fmt.Println(AppConfig.dav)
	davConfigs := strings.Split(AppConfig.dav, ";")

	WebDAVConfigs := make([]*model.WebDAVConfig, 0)

	for _, davConfig := range davConfigs {
		WebDAVConfig := &model.WebDAVConfig{}
		WebDAVConfig.InitByConfigStr(davConfig)

		WebDAVConfigs = append(WebDAVConfigs, WebDAVConfig)
	}

	w := &model.WebDAVConfig{}
	WebDAVConfigs = append(WebDAVConfigs, w)

	sMux := http.NewServeMux()
	sMux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {

		webDAVConfig := model.WebDAVConfigFindOneByPrefix(WebDAVConfigs, parsePrefixFromURL(req.URL))

		if webDAVConfig == nil {

			w.Header().Set("Content-Type", "text/html; charset=utf-8")

			// index

			_, err := fmt.Fprintf(w, "<pre>\n")
			if err != nil {
				fmt.Println(err)
			}

			for _, config := range WebDAVConfigs {
				_, err = fmt.Fprintf(w, "<a href=\"%s\" >%s</a>\n", config.Prefix+"/", config.Prefix)
				if err != nil {
					fmt.Println(err)
				}
			}

			_, err = fmt.Fprintf(w, "<pre>\n")
			if err != nil {
				fmt.Println(err)
			}

			return
		}

		if webDAVConfig.Username != "null" && webDAVConfig.Password != "null" {
			// 配置中的 用户名 密码 都为 null 时 不进行身份检查
			// 不都为 null 进行身份检查

			username, password, ok := req.BasicAuth()

			if !ok {
				w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			if username == "" || password == "" {
				http.Error(w, "username missing or password missing", http.StatusUnauthorized)
				return
			}

			if username != webDAVConfig.Username || password != webDAVConfig.Password {
				http.Error(w, "username wrong or password wrong", http.StatusUnauthorized)
				return
			}
		}

		if webDAVConfig.ReadOnly {
			allowMethods := []string{"GET", "OPTIONS", "PROPFIND", "HEAD"}
			if !IsContain(allowMethods, req.Method) {
				// ReadOnly
				w.WriteHeader(http.StatusMethodNotAllowed)
				_, err := w.Write([]byte("Readonly, Method " + req.Method + " Not Allowed"))
				if err != nil {
					fmt.Println(err)
					return
				}
				return
			}
		}

		if req.Method == "GET" && isDir(webDAVConfig.Handler.FileSystem, req) {
			fmt.Println(indexHTML)
			_, err := w.Write([]byte(indexHTML))
			if err != nil {
				fmt.Println(err)
				return
			}
			return
		}

		if req.Method == "HEAD" {
			return
		}

		// handle file
		webDAVConfig.Handler.ServeHTTP(w, req)
	})

	fmt.Println("start listen on :80")
	err := http.ListenAndServe(":80", sMux)
	if err != nil {
		fmt.Println(err)
	}
}

// /dav1/123.txt -> dav1
func parsePrefixFromURL(url *url.URL) string {
	u := fmt.Sprint(url)
	return "/" + strings.Split(u, "/")[1]
}

func isDir(fs webdav.FileSystem, req *http.Request) bool {
	ctx := context.Background()
	path := req.URL.Path
	//fmt.Println(path)
	s := strings.Split(path, "/")[2:]
	//fmt.Println(s)
	path = strings.Join(s, "/")
	//fmt.Println(path)

	f, err := fs.OpenFile(ctx, path, os.O_RDONLY, 0)
	if err != nil {
		return false
	}
	defer f.Close()

	if fi, _ := f.Stat(); fi != nil && !fi.IsDir() {
		return false
	}
	return true
}

func IsContain(items []string, item string) bool {
	for _, eachItem := range items {
		if eachItem == item {
			return true
		}
	}
	return false
}
