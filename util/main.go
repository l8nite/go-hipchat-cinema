package util

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"
)

// PrintDump prints dump of request, optionally writing it in the response
func PrintDump(w http.ResponseWriter, r *http.Request, write bool) {
	dump, _ := httputil.DumpRequest(r, true)
	log.Printf("%v", string(dump))
	if write == true {
		w.Write(dump)
	}
}

// Decode into a map[string]interface{} the JSON in the POST Request
func DecodePostJSON(r *http.Request, logging bool) (map[string]interface{}, error) {
	var err error
	var payLoad map[string]interface{}
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&payLoad)
	if logging {
		log.Printf("Parsed body:%v", payLoad)
	}
	return payLoad, err
}

func MovieTitle(input string) string {
	words := strings.Fields(input)
	smallwords := " a an on the to "

	for index, word := range words {
		if strings.Contains(smallwords, " "+word+" ") && index > 0 {
			words[index] = word
		} else {
			words[index] = strings.Title(word)
		}
	}
	return strings.Join(words, " ")
}
