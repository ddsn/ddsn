package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
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

type OwnerKey struct {
	Id    int
	Hash  string
}

func httpHandler(w http.ResponseWriter, r *http.Request) {
	request := r.URL.Path

	if strings.HasPrefix(request, "/res/") {
		resourcesHandler(w, r)
	} else {
		htmlHandler(w, r)
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

func htmlHandler(w http.ResponseWriter, r *http.Request) {
	loggedIn, account, message := loginChecker(w, r)

	// Display pages depending on logged in status

	page := path.Clean("/" + r.URL.Path)

	if page == "/" {
		page = "/index.html"
	}

	if loggedIn {
		// <Perform upload>

		if file, _, err := r.FormFile("file"); err == nil {
			defer file.Close()

			// TODO
		}

		// </Perform upload>

		// <Generate Key>

		if r.FormValue("genkey") != "" {
			key, err := rsa.GenerateKey(rand.Reader, 1024)

			if err != nil {
				message = "Error generating RSA key: " + err.Error()
			} else {
				block := pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}

				pemData := pem.EncodeToMemory(&block)

				Database.Exec("INSERT INTO key (priv_key, acc_id) VALUES ($1, $2)", string(pemData), account.Id)
			}
		}

		// </Generate Key>

		bytes, err := ioutil.ReadFile(Config.Server.TmlDir + "/int" + page)

		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		tml, err := template.New("index").Parse(string(bytes))

		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		type context struct {
			PeerName  string
			Identity  string
			Account   int
			AccName   string
			Version   string
			Message   string
			Keys      []OwnerKey
		}

		// <Retrieve keys>

		rows, _ := Database.Query("SELECT id, priv_key FROM key WHERE acc_id = $1", account.Id)

		keys := []OwnerKey{}

		for rows.Next() {
			var key OwnerKey
			var pemStr string

			rows.Scan(&key.Id, &pemStr)

			block, _ := pem.Decode([]byte(pemStr))
			privKey, _ := x509.ParsePKCS1PrivateKey(block.Bytes)
			publicBytes, _ := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
			hashBytes := sha256.Sum256(publicBytes)
			key.Hash = hex.EncodeToString(hashBytes[:])[0:12]

			keys = append(keys, key)
		}

		// </Retrieve keys>

		tml.Execute(w, context{PeerName, IdentityStr, account.Id, account.Name, Version, message, keys})
	} else {
		bytes, err := ioutil.ReadFile(Config.Server.TmlDir + "/ext" + page)

		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		tml, err := template.New("login").Parse(string(bytes))

		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		type context struct {
			PeerName  string
			Identity  string
			Message   string
			Version   string
		}

		tml.Execute(w, context{PeerName, IdentityStr, message, Version})
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