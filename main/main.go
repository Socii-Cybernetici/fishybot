package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
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
	var BOT_PUBKEY_HEX string = os.Getenv("DISCORD_BOT_PUBKEY_HEX")

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
		var request_values map[string]string = make(map[string]string)
		err = json.Unmarshal(body, &request_values)
		if err != nil {
			wr.WriteHeader(400)
			wr.Write([]byte("Failed to parse JSON; " + err.Error()))
			log.Print("Failed to parse JSON\n")
			return
		}
		log.Print("USERNAME: " + request_values["username"])
		log.Print("ACCESS CODE: " + request_values["code"])
		log.Print("SSH PUBLIC KEY: " + request_values["pubkey"])
		ch, err := discord_session.UserChannelCreate("489166470589448220")
		if err != nil {
			wr.WriteHeader(500)
			wr.Write([]byte("Failed to reach discord API; " + err.Error()))
			log.Print("Could not reach discord api for this request;" + err.Error() + "\n")
			return
		}
		_, err = discord_session.ChannelMessageSend(ch.ID,
			"***Submission Received***\n"+
				"Username: "+request_values["username"]+"\n"+
				"Access Code: "+request_values["code"]+"\n"+
				"SSH Public Key:\n"+request_values["pubkey"])
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
				"Username: "+request_values["username"]+"\n"+
				"Access Code: "+request_values["code"]+"\n"+
				"SSH Public Key:\n"+request_values["pubkey"])
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

	// discord_session.InteractionRespond()

	srv_handler.HandleFunc("POST /discord-interactions", func(wr http.ResponseWriter, rq *http.Request) {
		log.Print("Received discord interaction, checking cryptographic signature...\n")
		signature_hex := rq.Header.Get("X-Signature-Ed25519")
		timestamp := rq.Header.Get("X-Signature-Timestamp")
		var body []byte = make([]byte, rq.ContentLength)
		_, err := rq.Body.Read(body)
		if err != nil && err.Error() != "EOF" {
			wr.WriteHeader(500)
			log.Println("Failed to read request; " + err.Error())
			return
		}
		BOT_PUBKEY, err := hex.DecodeString(BOT_PUBKEY_HEX)
		if err != nil {
			wr.WriteHeader(401)
			log.Println("Invalid bot public key encoding; " + err.Error())
			return
		}
		signature, err := hex.DecodeString(signature_hex)
		if err != nil {
			wr.WriteHeader(401)
			log.Println("Invalid signature encoding; " + err.Error())
			return
		}
		message := []byte(fmt.Sprintf("%s%s", timestamp, string(body)))
		if !ed25519.Verify(BOT_PUBKEY, message, signature) {
			wr.WriteHeader(401)
			log.Println("Failed cryptographic signature check, ignoring request")
			return
		} else {
			log.Println("Succeeded cryptographic signature check, proceeding...")
		}
		log.Println("Raw body: " + string(body))
		request_values := make(map[string]any)
		err = json.Unmarshal(body, &request_values)
		if err != nil {
			wr.WriteHeader(401)
			log.Println("Failed to parse JSON; " + err.Error())
			return
		}
		switch request_values["type"] {
		case 1:
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
