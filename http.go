package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"path"
	"strings"
)

type Account struct {
	Id    int
	Name  string
}

func httpHandler(w http.ResponseWriter, r *http.Request) {
	request := r.URL.Path

	if request == "/" {
		indexHandler(w, r)
	} else if strings.HasPrefix(request, "/res/") {
		resourcesHandler(w, r)
	}
}

func loginChecker(w http.ResponseWriter, r *http.Request) (bool, Account, string) {
	loggedIn := false
	message := ""

	var account Account

	if r.FormValue("logout") != "" {
		sidCookie := http.Cookie{}

		sidCookie.Name = "SID"
		sidCookie.MaxAge = -1

		http.SetCookie(w, &sidCookie)
	} else {
		name := r.PostFormValue("name")
		pass := r.PostFormValue("pass")

		if name != "" && pass != "" {
			row, _ := Database.Query("SELECT id, name, pass, salt FROM account WHERE name LIKE $1", name)
			defer row.Close()

			if !row.Next() {
				message = "An account with that name does not exist."
			} else {
				var corrPassHash, salt string

				row.Scan(&account.Id, &account.Name, &corrPassHash, &salt)

				hash := sha256.Sum256([]byte(pass+salt))
				passHash := hex.EncodeToString(hash[:])

				if passHash != corrPassHash {
					message = "You have entered a wrong password."
				} else {
					sidCookie := http.Cookie{}

					sidCookie.Name = "SID"
					sidCookie.Value = "ABC123"

					Sessions["ABC123"] = account.Id

					http.SetCookie(w, &sidCookie)

					loggedIn = true
				}
			}
		} else {
			sidCookie, err := r.Cookie("SID")

			if err == nil {
				account.Id, loggedIn = Sessions[sidCookie.Value]

				if loggedIn {
					row, _ := Database.Query("SELECT name FROM account WHERE id = $1", account.Id)
					defer row.Close()

					if !row.Next() {
						message = "Error: database corruption"
						loggedIn = false
					}

					row.Scan(&account.Name)
				}
			}
		}
	}

	return loggedIn, account, message
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	loggedIn, account, message := loginChecker(w, r)

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
			AccName   string
		}

		tml.Execute(w, indexContext{PeerName, account.Id, account.Name})
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

		tml.Execute(w, loginContext{PeerName, message})
	}
}

func resourcesHandler(w http.ResponseWriter, r *http.Request) {
	request := r.URL.Path

	if r.Method != "GET" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	resPath := path.Clean(Config.Server.ResDir + "/" + request)

	if !strings.HasPrefix(resPath, Config.Server.ResDir) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	http.ServeFile(w, r, resPath)
}