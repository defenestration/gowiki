package main

import (
	"errors"
	"fmt"
	"github.com/go-redis/redis"
	"html/template"
	"io/ioutil"
	"net/http"
	// "reflect"
	"regexp"
	"strings"
)

var validPath = regexp.MustCompile("^/(edit|save|view|index)/([a-zA-Z0-9]+)$")
var indexPage string = "/"

type Page struct {
	Title string
	Body  []byte
}

func (p *Page) save() error {
	filename := "pages/" + p.Title + ".txt"
	return ioutil.WriteFile(filename, p.Body, 0600)
}

func loadPage(title string) (*Page, error) {
	filename := "pages/" + title + ".txt"
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	t, err := template.ParseFiles("templates/" + tmpl + ".html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = t.Execute(w, p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	// drop leading /view/ from url
	p, err := loadPage(title)
	if err != nil {
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}
	renderTemplate(w, "view", p)
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	err := p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)

}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	// list page names
	dir, err := ioutil.ReadDir("pages/")
	tmpl := "index"
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	names := []string{}
	for _, name := range dir {
		n := strings.TrimSuffix(name.Name(), ".txt")
		names = append(names, n)
	}
	// create an index page with page names
	t, err := template.ParseFiles("templates/" + tmpl + ".html")
	// array names is rendered in the template
	err = t.Execute(w, names)
}

func getTitle(w http.ResponseWriter, r *http.Request) (string, error) {
	m := validPath.FindStringSubmatch(r.URL.Path)
	if m == nil {
		http.NotFound(w, r)
		return "", errors.New("Invalid title")
	}
	return m[2], nil //title is second
}

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// get page title and call fn
		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, m[2])
	}
}

var client = redis.NewClient(&redis.Options{
	Addr:     "localhost:6379",
	Password: "", // no password set
	DB:       1,  // use default DB
})

func redisCmds() {
	// client := redis.NewClient(&redis.Options{
	// 	Addr:     "localhost:6379",
	// 	Password: "", // no password set
	// 	DB:       1,  // use default DB
	// })
	// clientt := reflect.TypeOf(client).Kind()
	// fmt.Println(clientt)
	pong, err := client.Ping().Result()
	fmt.Println(pong, err)
	err = client.Set("key", "value", 0).Err()
	if err != nil {
		panic(err)
	}
	// list keys
	keys, _ := client.Keys("*").Result()
	fmt.Println("keys", keys)
	fmt.Println("values", len(keys))
	for _, k := range keys {
		value, _ := client.Get(k).Result()
		fmt.Println(value)
	}
}

func main() {
	redisCmds()
	http.HandleFunc(indexPage, indexHandler)
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))
	fmt.Println("serving on :8080")
	http.ListenAndServe(":8080", nil)
}
