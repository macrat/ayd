package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

const DATA = "/tmp/data"

func main() {
	log.Println("start server")

	mux := http.NewServeMux()

	for _, path := range []string{"/ftp", "/ssh"} {
		path := path

		os.MkdirAll(DATA+path, 0755)
		os.Chmod(DATA+path, 0777)

		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}

			fs, err := os.ReadDir(DATA + path)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintln(w, err)
				log.Println("error:", err)
				return
			}

			for _, f := range fs {
				fname := DATA + path + "/" + f.Name()
				err := os.RemoveAll(fname)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Fprintln(w, err)
					return
				}
				log.Println("delete:", fname)
			}

			w.WriteHeader(http.StatusNoContent)
		})
	}

	log.Fatal(http.ListenAndServe("0.0.0.0:8080", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("request:", r.Method, r.URL)
		mux.ServeHTTP(w, r)
	})))
}
