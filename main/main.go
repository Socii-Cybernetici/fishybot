package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	// "golang.org/x/crypto/nacl/auth"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load(os.Args[1])
	if err != nil {
		panic(err)
	}
	var BOT_PORT string = os.Getenv("BOT_PORT")
	var BOT_TOKEN string = os.Getenv("DISCORD_BOT_TOKEN")

	discord_session, err := discordgo.New("Bot " + BOT_TOKEN)
	if err != nil {
		panic(err)
	}

	/* Server Init */
	var srv_handler *http.ServeMux = http.NewServeMux()
	var srv http.Server = http.Server{
		Addr:    ":" + BOT_PORT,
		Handler: srv_handler,
	}
	/* fishy.best endpoints */
	srv_handler.HandleFunc("POST /submit", func(wr http.ResponseWriter, rq *http.Request) {
		var data []byte = make([]byte, rq.ContentLength)
		_, err := rq.Body.Read(data)
		if err != nil {
			wr.WriteHeader(500)
			wr.Write([]byte("Internal I/O error with request"))
			log.Print("Failed to read request " + err.Error())
		}
		log.Print("REQUEST BODY: " + string(data))
		var req_values map[string]string = make(map[string]string)
		err = json.Unmarshal(data, &req_values)
		if err != nil {
			wr.WriteHeader(400)
			wr.Write([]byte("Failed to parse JSON; " + err.Error()))
			log.Print("Failed to parse JSON\n")
			return
		}
		log.Print("USERNAME: " + req_values["username"])
		log.Print("ACCESS CODE: " + req_values["code"])
		log.Print("SSH PUBLIC KEY: " + req_values["pubkey"])
		ch, err := discord_session.UserChannelCreate("489166470589448220")
		if err != nil {
			wr.WriteHeader(500)
			wr.Write([]byte("Failed to reach discord API; " + err.Error()))
			log.Print("Could not reach discord api for this request;" + err.Error() + "\n")
			return
		}
		_, err = discord_session.ChannelMessageSend(ch.ID,
			"***Submission Received***\n"+
				"Username: "+req_values["username"]+"\n"+
				"Access Code: "+req_values["code"]+"\n"+
				"SSH Public Key:\n"+req_values["pubkey"])
		if err != nil {
			wr.WriteHeader(500)
			wr.Write([]byte("Failed to reach discord API; " + err.Error()))
			log.Print("Could not reach discord api for this request;" + err.Error() + "\n")
			return
		}
		/* Nora */
		ch, err = discord_session.UserChannelCreate("976576169325522994")
		if err != nil {
			wr.WriteHeader(500)
			wr.Write([]byte("Failed to reach discord API; " + err.Error()))
			log.Print("Could not reach discord api for this request;" + err.Error() + "\n")
			return
		}
		_, err = discord_session.ChannelMessageSend(ch.ID,
			"***Submission Received***\n"+
				"Username: "+req_values["username"]+"\n"+
				"Access Code: "+req_values["code"]+"\n"+
				"SSH Public Key:\n"+req_values["pubkey"])
		if err != nil {
			wr.WriteHeader(500)
			wr.Write([]byte("Failed to reach discord API; " + err.Error()))
			log.Print("Could not reach discord api for this request;" + err.Error() + "\n")
			return
		}
		wr.Write([]byte("Submission received:\n" + string(data)))
		log.Print("Submission received:\n" + string(data))
	})
	/* discord interaction endpoints */
	srv_handler.HandleFunc("POST /discord-interactions", func(wr http.ResponseWriter, rq *http.Request) {
		log.Print("Received discord interaction\n")
		var data []byte = make([]byte, rq.ContentLength)
		_, err := rq.Body.Read(data)
		if err != nil {
			wr.WriteHeader(500)
			wr.Write([]byte("Internal I/O error with request"))
			log.Print("Failed to read request " + err.Error())
			return
		}
		var req_values map[string]string = make(map[string]string)
		err = json.Unmarshal(data, &req_values)
		if err != nil {
			wr.WriteHeader(400)
			wr.Write([]byte("Failed to parse JSON; " + err.Error()))
			log.Print("Failed to parse JSON\n")
			return
		}
		switch req_values["type"] {
		case "1":
			log.Print("This is a ping request\n")
			// ret_values := make(map[string]int)
			// ret_values["type"] = 1
			// ret_json, err := json.Marshal(ret_values)
			// if err != nil {
			// 	wr.WriteHeader(500)
			// 	wr.Write([]byte("Failed to marshal JSON; " + err.Error()))
			// }
			// wr.WriteHeader(200)
			// wr.Write(ret_json)
			wr.Write([]byte("{\"type\": 1}"))
		}
	})
	/* Server Init */
	log.Printf("Listening on port %s", BOT_PORT)
	err = srv.ListenAndServe()
	log.Fatal(err.Error())
}
