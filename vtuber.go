// single file go server for logs.
package main

import (
	"bufio"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"
)

//go:embed site/*
var embed_site embed.FS

type Search struct {
	// string or int
	Filters map[string]interface{} `json:"filters"`
	// todo: regexes too?
	Searches map[string]string `json:"searches"`

	Page  *int `json:"page"`
	Limit *int `json:"limit"`
}

func memoryWorker(jsons chan map[string]interface{}, searches chan Search, results chan []map[string]interface{}) {
	// indexes when?
	var documents []map[string]interface{}

	for {
		select {
		case document := <-jsons:
			// new json
			documents = append(documents, document)
		case search := <-searches:
			// new search

			var limit int
			if search.Limit == nil {
				limit = 32
			} else {
				limit = *search.Limit
			}

			var page int
			if search.Page == nil {
				page = 0
			} else {
				page = *search.Page
			}

			res := make([]map[string]interface{}, 0, limit)
			count := 0

			for _, document := range documents {
				good := true
				for key, value := range search.Filters {
					if document[key] != value {
						good = false
						break
					}
				}

				if good {
					for key, value := range search.Searches {
						if v, ok := document[key].(string); ok {
							if !strings.Contains(v, value) {
								good = false
								break
							}
						}
					}
				}

				if good && count >= page*limit && count < (page+1)*limit {
					res = append(res, document)
					count++
				} else if good {
					break
				}
			}

			results <- res
		}
	}
}

func processLog(message string) (map[string]interface{}, error) {
	var msg map[string]interface{}

	if err := json.Unmarshal([]byte(message), &msg); err != nil {
		return nil, err
	}

	date_text := msg["timestamp"]
	if date_text == nil {
		return nil, errors.New("a log is missing a timestamp")
	} else if reflect.ValueOf(date_text).Kind() != reflect.String {
		return nil, errors.New("non-string timestamp")
	}

	var time time.Time

	if err := time.UnmarshalText([]byte(date_text.(string))); err != nil {
		return nil, err
	}

	msg["timestamp"] = time

	return msg, nil
}

func main() {
	site, err := fs.Sub(embed_site, "site")
	if err != nil {
		log.Fatal(err)
	}

	// make sure it's ok
	_, err = site.Open("index.html")
	if err != nil {
		log.Fatal(err)
	}

	// logs should get stored
	exec_path, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}

	// todo path seperator potential vuln?
	// logs_path := path.Join(exec_path, "..", "logfile")
	logs_path := filepath.Join(exec_path, "..", "logfile")

	logfile, err := os.OpenFile(logs_path, os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		log.Fatal(err)
	}

	jsons := make(chan map[string]interface{})
	searches := make(chan Search)
	results := make(chan []map[string]interface{})

	go memoryWorker(jsons, searches, results)

	scanner := bufio.NewScanner(logfile)
	for scanner.Scan() {
		json, err := processLog(scanner.Text())
		if err != nil {
			log.Fatal(err)
		}

		jsons <- json
	}

	// http.Handle("/", http.FileServer(http.FS(site)))
	http.Handle("/", http.FileServer(http.Dir("./site")))

	http.HandleFunc("/api/hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello, world!")
	})

	var mutex sync.Mutex

	http.HandleFunc("/api/logs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			if _, err := fmt.Fprint(w, "405: Method not allowed."); err != nil {
				log.Fatal(err)
			}
			w.WriteHeader(405)
			return
		}

		scanner := bufio.NewScanner(r.Body)
		for scanner.Scan() {
			line := scanner.Text()
			json, err := processLog(line)
			if err != nil {
				fmt.Fprintf(w, "400: Couldn't parse JSON: %s", err)
				w.WriteHeader(400)
				log.Print(err)
				return
			}

			mutex.Lock()
			if _, err := logfile.Write(append([]byte(line), '\n')); err != nil {
				fmt.Fprintf(w, "500: Unable to persist logs: %s", err)
				w.WriteHeader(500)
				log.Print(err)
				return
			}
			mutex.Unlock()

			jsons <- json
		}

		if err := scanner.Err(); err != nil {
			fmt.Fprintf(w, "400: Error encountered: %s", err)
			w.WriteHeader(400)
			log.Print(err)
			return
		}
	})

	http.HandleFunc("/api/search", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			if _, err := fmt.Fprint(w, "405: Method not allowed."); err != nil {
				log.Fatal(err)
			}
			w.WriteHeader(405)
			return
		}

		var search Search
		reader := json.NewDecoder(r.Body)
		if err := reader.Decode(&search); err != nil {
			fmt.Fprintf(w, "400: Couldn't parse JSON: %s", err)
			w.WriteHeader(400)
			log.Print(err)
			return
		}

		searches <- search
		result := <-results

		writer := json.NewEncoder(w)
		if err := writer.Encode(result); err != nil {
			fmt.Fprintf(w, "400: Couldn't encode JSON: %s", err)
			w.WriteHeader(400)
			log.Print(err)
			return
		}
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
