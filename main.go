package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	websocket "github.com/gorilla/websocket"
)

var upgraderWS = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type (
	// Users is the main directory of peers
	Users struct {
		List []User
	}

	User struct {
		con        *websocket.Conn
		username   string
		offer      map[string]interface{}
		candidates []map[string]interface{}
	}

	Message map[string]interface{}
)

func NewUsers() *Users {
	return &Users{
		List: make([]User, 0),
	}
}

func Serve() {
	// starts a http server
	port := 8080
	// check if port is available
	if !isPortAvailable(port) {
		log.Fatal("Port is not available")
	}
	log.Println("Starting signalling server on port: ", port)

	// create a new user directory
	users := NewUsers()
	// create a new websocket server
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// upgrade the http connection to a websocket connection
		conn, err := upgraderWS.Upgrade(w, r, nil)
		if err != nil {
			log.Println("ws", err)
			return
		}
		log.Println("New Connection")
		defer func() {
			// log.Println("error 3")
			conn.Close()
			// remove the user from the directory
			for i, user := range users.List {
				if user.con == conn {
					users.List = append(users.List[:i], users.List[i+1:]...)
					break
				}
			}
		}()
		// add the new user to the directory
		// users.List = append(users.List, User{con: conn})
		// listen for messages from the user
		for {
			// read the message
			msg := Message{}
			err := conn.ReadJSON(&msg)
			if err != nil {
				log.Println("read", err)
				return
			}
			// log.Println("error 5")
			// log.Println(msg)

			var currUser *User

			// check if the message contains username
			if msg["username"] != nil {
				log.Println("Username provided: ", msg["username"].(string))
				// find the user in the directory
				for i, user := range users.List {
					if user.username == msg["username"].(string) {
						log.Println("User found: ", user.username)
						currUser = &users.List[i]
						break
					}
				}
			} else {
				// Fatal
				log.Println("Username not provided")
				return
			}
			// data type switch
			switch msg["type"] {
			case "store_user":
				log.Println("store_user")
				if currUser != nil {
				} else {
					// new user
					if msg["username"] == nil {
						log.Println("Username not provided")
						return
					}
					currUser = &User{con: conn, username: msg["username"].(string)}
					users.List = append(users.List, *currUser)
				}
			case "store_offer":
				log.Println("store_offer")
				if currUser != nil {
					if msg["offer"] == nil {
						log.Println("error 6")
						log.Println("Offer not provided")
						return
					}
					//log.Println(msg["offer"])
					currUser.offer = parseStringifiedJSON(msg["offer"])

				}
			case "store_candidate":
				log.Println("store_candidate")
				if currUser != nil {
					if msg["candidate"] == nil {
						log.Println("error 7")
						log.Println("Candidate not provided")
						return
					}
					currUser.candidates = append(currUser.candidates, parseStringifiedJSON(msg["candidate"]))
				}

			case "send_answer":
				log.Println("send_answer")
				if currUser != nil {
					data := map[string]interface{}{
						"type": []byte("answer"),
					}
					if msg["answer"] == nil {
						data["answer"] = []byte("")
					} else {
						data["answer"] = msg["answer"].([]byte)
					}
					// send the answer to the user
					sendResponse(currUser.con, data)
				}

			case "send_candidate":
				if currUser != nil {
					data := map[string]interface{}{
						"type": []byte("candidate"),
					}
					if msg["candidate"] == nil {
						data["candidate"] = []byte("")
					} else {
						data["candidate"] = msg["candidate"].([]byte)
					}
					// send the candidate to the user
					sendResponse(currUser.con, data)
				}

			case "join_call":
				if currUser != nil {

					// send the offer to the user
					data := map[string]interface{}{
						"type": []byte("offer"),
					}
					if currUser.offer == nil {
						// fatal
						log.Println("Offer not found")
						return
					} else {
						data["offer"] = currUser.offer
					}
					sendResponse(currUser.con, data)
					// send the candidates to the user
					for i := 0; i < len(currUser.candidates); i++ {
						data := map[string]interface{}{
							"type": []byte("candidate"),
						}
						if currUser.candidates[i] != nil {
							data["candidate"] = currUser.candidates[i]

						}
						sendResponse(conn, data)
					}
				}
			}
		}
	})

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}

}

func sendResponse(wc *websocket.Conn, msg Message) {
	err := wc.WriteJSON(msg)
	if err != nil {
		log.Println("error 9")
		log.Println(err)
		return
	}
}

func isPortAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

func main() {

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	// start the server
	go Serve()

	<-sig
}

func parseStringifiedJSON(i interface{}) map[string]interface{} {
	var result map[string]interface{}
	log.Println("parseStringifiedJSON")
	b, _ := json.Marshal(i)
	json.Unmarshal(b, &result)
	log.Println("result", result)
	return result
}
