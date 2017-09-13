package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"golang.org/x/net/html"
)

type Document struct {
	Url      string            `json:"url"`
	Meta     DocumentMeta      `json:"meta"`
	Elements []DocumentElement `json:"elements,omitempty"`
}

type DocumentMeta struct {
	Status        int    `json:"status"`
	ContentType   string `json:"conten-type,omitempty"`
	ContentLength int    `json:"content-length,omitempty"`
}

type DocumentElement struct {
	TagName string `json:"tag-name"`
	Count   int    `json:"count"`
}

// max available threads
var tokens = make(chan struct{}, 20)

func fetchDoc(url string) Document {
	tokens <- struct{}{}

	var (
		client  = &http.Client{}
		req, _  = http.NewRequest("GET", url, nil)
		resp, _ = client.Do(req)
		counts  = map[string]int{}
		body, _ = ioutil.ReadAll(resp.Body)
		reader  = bytes.NewBuffer(body)
		z       = html.NewTokenizer(reader)
	)

	doc := Document{
		Url: url,
	}

	doc.Meta.ContentType = resp.Header.Get("Content-Type")
	doc.Meta.Status = resp.StatusCode
	doc.Meta.ContentLength = len(body)

	for {
		switch z.Next() {
		case html.StartTagToken, html.SelfClosingTagToken:
			tagName, _ := z.TagName()
			counts[string(tagName)]++
		}
		if z.Err() != nil {
			break
		}
	}

	doc.Elements = make([]DocumentElement, 0, len(counts))
	for t, c := range counts {
		e := DocumentElement{
			TagName: t,
			Count:   c,
		}
		doc.Elements = append(doc.Elements, e)
	}

	<-tokens

	return doc
}

func postRequest(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	switch r.Method {
	case "POST":
		if r.Body == nil {
			http.Error(w, "Please send a request body", 400)
			return
		}

		var urls []string
		if e := (json.NewDecoder(r.Body)).Decode(&urls); e != nil {
			fmt.Fprintf(w, "Invalid JSON structure")
			return
		}

		var urlLength int = len(urls)
		docs := make(chan Document, urlLength)
		checked := make(map[string]bool)

		for i := urlLength; i > 0; i-- {
			for _, url := range urls {
				if !checked[url] {
					checked[url] = true
					i++
					go func(url string) {
						docs <- fetchDoc(url)
					}(url)
				}
			}
		}

		var fully int = 0
		documents := make([]Document, urlLength)
		for i := range docs {
			documents[fully] = i
			fully++
			if fully == urlLength {
				break
			}
		}

		respData, err := json.Marshal(&documents)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Add("Content-Type", "application/json")
		if _, err = w.Write(respData); err != nil {
			fmt.Fprintf(w, "Failed to write response")
		}

	default:
		fmt.Fprintf(w, "Only POST method are supported.")
	}
}

func main() {
	http.HandleFunc("/", postRequest)

	fmt.Printf("Server is started...\n")
	if err := http.ListenAndServe(":8989", nil); err != nil {
		log.Fatal(err)
	}
}
