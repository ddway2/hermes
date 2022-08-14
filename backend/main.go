package main

import (
	"flag"
	"fmt"
	"hash"
	"hash/fnv"
	"io/ioutil"
	"log"
	"net/http"
)

var SumFNV hash.Hash64

func main() {
	port := flag.String("port", "8080", "HTTP port")

	flag.Parse()

	http.HandleFunc("/endpoint", handleRequest)

	SumFNV = fnv.New64a()

	log.Println("Server ready on " + *port)
	err := http.ListenAndServe(":"+*port, nil)
	if err != nil {
		panic(err.Error())
	}

}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		log.Println("Method not allowed")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("can't read body")
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}

	SumFNV.Reset()
	SumFNV.Write(body)

	fmt.Fprint(w, string(SumFNV.Sum(nil)))
}
