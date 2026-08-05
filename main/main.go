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
		}
		log.Print("REQUEST BODY: " + string(data))
		values := make(map[string]string)
		err = json.Unmarshal(data, &values)
		if err != nil {
			wr.WriteHeader(300)
			wr.Write([]byte("Failed to parse JSON; " + err.Error()))
			log.Print("Failed to parse JSON\n")
			return
		}
		log.Print("USERNAME: " + values["username"])
		log.Print("ACCESS CODE: " + values["code"])
		log.Print("SSH PUBLIC KEY: " + values["pubkey"])
		ch, err := discord_session.UserChannelCreate("489166470589448220")
		if err != nil {
			wr.WriteHeader(500)
			wr.Write([]byte("Failed to reach discord API; " + err.Error()))
			log.Print("Could not reach discord api for this request;" + err.Error() + "\n")
			return
		}
		_, err = discord_session.ChannelMessageSend(ch.ID,
			"***Submission Received***\n"+
				"Username: "+values["username"]+"\n"+
				"Access Code: "+values["code"]+"\n"+
				"SSH Public Key:\n"+values["pubkey"])
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
				"Username: "+values["username"]+"\n"+
				"Access Code: "+values["code"]+"\n"+
				"SSH Public Key:\n"+values["pubkey"])
		if err != nil {
			wr.WriteHeader(500)
			wr.Write([]byte("Failed to reach discord API; " + err.Error()))
			log.Print("Could not reach discord api for this request;" + err.Error() + "\n")
			return
		}
		wr.WriteHeader(200)
		wr.Write([]byte("Submission received:\n" + string(data)))
		log.Print("Submission received:\n" + string(data))
	})
	/* discord interaction endpoints */
	log.Printf("Listening on port %s", BOT_PORT)
	err = srv.ListenAndServe()
	log.Fatal(err.Error())
}
