package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
)

func httpHandler(w http.ResponseWriter, r *http.Request) {
	request := r.URL.Path

	if request == "/" {
		indexHandler(w, r)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	account := 0
	loggedIn := false

	// Determine whether we are logged in

	message := ""

	if r.FormValue("logout") != "" {
		sidCookie := http.Cookie{}

		sidCookie.Name = "SID"
		sidCookie.MaxAge = -1

		http.SetCookie(w, &sidCookie)
	} else {
		name := r.PostFormValue("name")
		pass := r.PostFormValue("pass")

		if name != "" && pass != "" {
			row, _ := Database.Query("SELECT id, pass, salt FROM account WHERE name LIKE $1", name)
			defer row.Close()

			if !row.Next() {
				message = "An account with that name does not exist."
			} else {
				var corrPassHash, salt string

				row.Scan(&account, &corrPassHash, &salt)

				hash := sha256.Sum256([]byte(pass+salt))
				passHash := hex.EncodeToString(hash[:])

				if passHash != corrPassHash {
					message = "You have entered a wrong password."
				} else {
					sidCookie := http.Cookie{}

					sidCookie.Name = "SID"
					sidCookie.Value = "ABC123"

					Sessions["ABC123"] = 15

					http.SetCookie(w, &sidCookie)

					loggedIn = true
					account = 15
				}
			}
		} else {
			sidCookie, err := r.Cookie("SID")

			if err == nil {
				account, loggedIn = Sessions[sidCookie.Value]
			}
		}
	}

	// Display pages depending on logged in status

	if loggedIn {
		bytes, err := ioutil.ReadFile(Config.Server.TmlDir + "/index.html")

		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		tml, err := template.New("index").Parse(string(bytes))

		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		type indexContext struct {
			PeerName  string
			Account   int
		}

		tml.Execute(w, indexContext{"Test", account})
	} else {
		bytes, err := ioutil.ReadFile(Config.Server.TmlDir + "/login.html")

		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		tml, err := template.New("login").Parse(string(bytes))

		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		type loginContext struct {
			PeerName  string
			Message   string
		}

		tml.Execute(w, loginContext{"Test", message})
	}
}