package main

import (
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/ed25519"
)

func main() {
	err := godotenv.Load(os.Args[1])
	if err != nil {
		panic(err)
	}
	var BOT_PORT string = os.Getenv("BOT_PORT")
	var BOT_TOKEN string = os.Getenv("DISCORD_BOT_TOKEN")
	var BOT_PUBKEY_HEX string = os.Getenv("DISCORD_BOT_PUBKEY")

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
		var body []byte = make([]byte, rq.ContentLength)
		_, err := rq.Body.Read(body)
		if err != nil {
			wr.WriteHeader(500)
			wr.Write([]byte("Internal I/O error with request"))
			log.Print("Failed to read request " + err.Error())
		}
		log.Print("REQUEST BODY: " + string(body))
		var req_values map[string]string = make(map[string]string)
		err = json.Unmarshal(body, &req_values)
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
		wr.Write([]byte("Submission received:\n" + string(body)))
		log.Print("Submission received:\n" + string(body))
	})
	/* discord interaction endpoints */
	srv_handler.HandleFunc("POST /discord-interactions", func(wr http.ResponseWriter, rq *http.Request) {
		log.Print("Received discord interaction, checking cryptographic signature...\n")
		signature := rq.Header["X-Signature-Ed25519"][0]
		timestamp := rq.Header["X-Signature-Timestamp"][0]
		var body []byte = make([]byte, rq.ContentLength)
		_, err := rq.Body.Read(body)
		if err != nil {
			wr.WriteHeader(500)
			log.Print("Failed to read request " + err.Error())
			return
		}
		BOT_PUBKEY_BYTES, err := hex.DecodeString(BOT_PUBKEY_HEX)
		if err != nil {
			wr.WriteHeader(401)
			log.Println("Invalid hex encoding of bot public key; " + err.Error())
			return
		}
		if !ed25519.Verify(BOT_PUBKEY_BYTES, append([]byte(timestamp), body...), []byte(signature)) {
			wr.WriteHeader(401)
			log.Println("Failed cryptographic signature check, ignoring request")
			return
		}
		var req_values map[string]string = make(map[string]string)
		err = json.Unmarshal(body, &req_values)
		if err != nil {
			wr.WriteHeader(401)
			log.Println("Failed to parse JSON " + err.Error())
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
