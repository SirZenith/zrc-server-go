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
	HandlerMap[path.Join(APIRoot, APIVer, "auth/login")] = loginHandler
	HandlerMap[path.Join(APIRoot, APIVer, "compose/aggregate")] = aggregateHandler
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	token := LoginToken{
		"this",
		"Bearer",
		true,
	}
	res, err := json.Marshal(token)
	if err != nil {
		log.Println("Error occured while generating JSON for login token.")
		log.Println(err)
	}
	fmt.Fprint(w, string(res))
}

func aggregateHandler(w http.ResponseWriter, r *http.Request) {
	userID := 1
	data, err := forms.Parse(r)
	if err != nil {
		log.Println(err)
	}

	val := data.Validator()
	val.Require("calls")

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
		tojson, err := handler(userID)
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
