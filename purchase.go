package main

import (
	"fmt"
	"log"
	"net/http"
	"path"
)

func init() {
	R.Handle(
		path.Join(APIRoot, "purchase/bundle/pack"),
		http.HandlerFunc(packInfoHandler),
	)
	InsideHandler[path.Join(APIRoot, "purchase/bundle/pack")] = getPackInfo
}

func packInfoHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := verifyBearerAuth(r.Header.Get("Authorization"))
	if err != nil {
		c := Container{false, nil, 203}
		http.Error(w, c.toJSON(), http.StatusUnauthorized)
		return
	}
	tojson, err := getPackInfo(userID, r)
	if err != nil {
		log.Println(err)
	}
	fmt.Fprint(w, tojson.toJSON())
}

func getPackInfo(_ int, _ *http.Request) (ToJSON, error) {
	container := []PackInfo{}
	var (
		name         string
		price        int
		origPrice    int
		disCountFrom int64
		disCountTo   int64
		itemID       string
		itemType     string
		isAvailable  bool
	)
	rows, err := db.Query(`select
			pack_name, price, orig_price, discount_from, discount_to
		from pack`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		rows.Scan(&name, &price, &origPrice, &disCountFrom, &disCountTo)
		items := []PackItem{}
		itemRows, err := db.Query(`select 
				item_id, item_type, is_available
			from
				pack_item
			where
				pack_name = ?`, name)
		if err != nil {
			return nil, err
		}
		defer itemRows.Close()

		for itemRows.Next() {
			itemRows.Scan(&itemID, &itemType, &isAvailable)
			items = append(items, PackItem{
				itemID, itemType, isAvailable,
			})
		}

		if err := itemRows.Err(); err != nil {
			return nil, err
		}

		container = append(container, PackInfo{
			name, items, price, origPrice, disCountFrom, disCountTo,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return (*PackInfoContainer)(&container), nil
}
