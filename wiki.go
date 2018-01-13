package main

import (
	"database/sql"
	"errors"
	"fmt"
	// "github.com/go-redis/redis"
	"html/template"
	"io/ioutil"
	"net/http"
	// "reflect"
	_ "github.com/mattn/go-sqlite3"
	"regexp"
	"strings"
	// "time"
	"log"
	"strconv"
)

var validPath = regexp.MustCompile("^/(edit|save|view|index)/([a-zA-Z0-9]+)$")
var indexPage string = "/"

type Page struct {
	Title string
	Body  []byte
}

type Quote struct {
	Id   int
	Body string
	// Author    string
	// Submitter string
	// Submitted time
	Tags []string
}

var db, dberr = sql.Open("sqlite3", "./quotes.db")

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func sqliteDbInit() {
	checkErr(dberr)
	statement, _ := db.Prepare("CREATE TABLE IF NOT EXISTS quotes (id INTEGER PRIMARY KEY, body TEXT, tags TEXT)")
	result, err := statement.Exec()
	defer statement.Close()
	checkErr(err)
	fmt.Println("result", result)
	// newQuote("newwwww quote body", "tag, tag2, blah")
}

func newQuote(body string, tags string) {
	// new q
	statement, err := db.Prepare("INSERT INTO quotes (body, tags) VALUES (?, ?)")
	checkErr(err)
	_, err = statement.Exec(body, tags)
	checkErr(err)
	// fmt.Println("result", result)
	defer statement.Close()
	printQuotes()
}

func (q *Quote) save() error {
	// edit an existing quote
	stmt := "update quotes set body = ? tags = ? where id = ?"
	statement, err := db.Prepare(stmt)
	checkErr(err)
	_, err = statement.Exec(q.Body, strings.Join(q.Tags, ", "), q.Id)
	checkErr(err)
	defer statement.Close()
	return nil
}

func printQuotes() {
	// check rows
	rows, err := db.Query("SELECT id, body, tags FROM quotes")
	checkErr(err)
	// fmt.Println(db.Rows)
	for rows.Next() {
		var id int
		var body string
		var tags string
		err = rows.Scan(&id, &body, &tags)
		checkErr(err)
		//convert tags to array
		tagArr := strings.Split(tags, ", ")
		fmt.Println(strconv.Itoa(id), body, tagArr)
	}
	q, _ := loadQuoteId(1)
	fmt.Println(q.Body)
}

func loadQuoteId(id int) (*Quote, error) {
	fmt.Println("id", id)
	row, err := db.Prepare("select id, body, tags from quotes where id = ?")
	checkErr(err)
	// var id int
	var body string
	var t string
	err = row.QueryRow(id).Scan(&id, &body, &t)
	checkErr(err)
	var tags = strings.Split(t, ", ")
	fmt.Println(body, tags)
	return &Quote{Id: id, Body: body, Tags: tags}, nil
}

// func (q *Quote) save() error {
// 	// save quote to sql db
// }

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

func main() {
	sqliteDbInit()
	http.HandleFunc(indexPage, indexHandler)
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))
	fmt.Println("serving on :8080")
	http.ListenAndServe(":8080", nil)
}
