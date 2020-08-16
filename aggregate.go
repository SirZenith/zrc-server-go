package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path"
	"strings"

	"github.com/albrow/forms"
)

func init() {
	R.Handle(
		path.Join(APIRoot, APIVer, "compose/aggregate"),
		http.HandlerFunc(aggregateHandler),
	)
}

func aggregateHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := verifyBearerAuth(r.Header.Get("Authorization"))
	if err != nil {
		c := Container{false, nil, 203}
		http.Error(w, c.toJSON(), http.StatusUnauthorized)
		return
	}
	data, err := forms.Parse(r)
	if err != nil {
		log.Println(err)
	}

	val := data.Validator()
	val.Require("calls")
	if val.HasErrors() {
		log.Printf("%s: Form passed lacks of necessary key(s).", r.URL.Path)
		for k, v := range val.ErrorMap() {
			log.Printf("%s: %s\n", k, v)
		}
		return
	}

	var calls []AggCall
	container := Container{true, nil, 0}
	results := []AggResult{}
	json.Unmarshal([]byte(data.Get("calls")), &calls)
	for _, call := range calls {
		handler, ok := InsideHandler[path.Join(
			APIRoot, APIVer, strings.Split(call.EndPoint, "?")[0])]
		if !ok {
			log.Println("Unknow request endpoint ", call.EndPoint)
			results = append(results, AggResult{call.ID, &EmptyList{}})
			continue
		}
		tojson, err := handler(userID, r)
		if err != nil {
			container.Success = false
			log.Println(err)
			break
		}
		results = append(results, AggResult{call.ID, tojson})
	}
	container.Value = (*AggContainer)(&results)
	fmt.Fprintln(w, container.toJSON())
}
