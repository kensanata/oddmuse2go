package main

import (
	"log"
	"net/http"
)

// addHandler uses the "add.html" template to present an empty edit
// page. What you type there is appended to the page using the
// appendHandler.
func addHandler(w http.ResponseWriter, r *http.Request, name string) {
	p, err := loadPage(name)
	if err != nil {
		p = &Page{Title: name, Name: name}
	} else {
		p.handleTitle(false)
	}
	renderTemplate(w, "add", p)
}

// appendHandler takes the "body" form parameter and appends it. The
// browser is redirected to the page view.
func appendHandler(w http.ResponseWriter, r *http.Request, name string) {
	body := r.FormValue("body")
	p, err := loadPage(name)
	if err != nil {
		p = &Page{Name: name, Body: []byte(body)}
	} else {
		p.Body = append(p.Body, []byte(body)...)
	}
	p.handleTitle(false)
	err = p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if r.FormValue("notify") == "on" {
		err = p.notify()
		if err != nil {
			log.Println("notify:", err)
		}
	}
	http.Redirect(w, r, "/view/"+name, http.StatusFound)
}
