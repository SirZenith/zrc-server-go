package main

import (
	"fmt"
	"log"
	"net/http"
)

func packInfoHandler(w http.ResponseWriter, r *http.Request) {
	var (
		userID int
		err    error
	)
	if NeedAuth {
		userID, err = verifyBearerAuth(r.Header.Get("Authorization"))
		if err != nil {
			c := Container{false, nil, 203}
			http.Error(w, c.toJSON(), http.StatusUnauthorized)
			return
		}
	} else {
		userID = staticUserID
	}
	tojson, err := getPackInfo(userID, r)
	if err != nil {
		log.Println(err)
	}
	fmt.Fprint(w, tojson.toJSON())
}

func getPackInfo(_ int, _ *http.Request) (ToJSON, error) {
	container := []PackInfo{}
	rows, err := db.Query(sqlStmtPackInfo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		info := new(PackInfo)
		rows.Scan(
			&info.Name,
			&info.Price,
			&info.OrigPrice,
			&info.DiscountFrom,
			&info.DiscountTo,
		)
		items := []PackItem{}
		itemRows, err := db.Query(sqlStmtPackItem, info.Name)
		if err != nil {
			return nil, err
		}
		defer itemRows.Close()

		for itemRows.Next() {
			item := new(PackItem)
			itemRows.Scan(&item.ID, &item.ItemType, &item.IsAvailable)
			items = append(items, *item)
		}

		if err := itemRows.Err(); err != nil {
			return nil, err
		}

		container = append(container, *info)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return (*PackInfoContainer)(&container), nil
}
