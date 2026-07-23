package main

import (
	// "encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load(".env")
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
	srv_handler.HandleFunc("GET /", func(wr http.ResponseWriter, rq *http.Request) {
		wr.WriteHeader(200)
		wr.Write([]byte("nothing to see here; use: POST /submit\n"))
		log.Print("Received GET /\n")
	})
	srv_handler.HandleFunc("POST /submit", func(wr http.ResponseWriter, rq *http.Request) {
		if rq.ContentLength == 0 {
			wr.WriteHeader(400)
			wr.Write([]byte("Request body is empty"))
			log.Print("Request body is empty\n")
			return
		}
		var data []byte = make([]byte, rq.ContentLength)
		_, err := rq.Body.Read(data)
		log.Print("REQUEST:" + string(data))
		// if err != nil {
		// 	wr.WriteHeader(400)
		// 	wr.Write([]byte("Failed to read request body; " + err.Error()))
		// 	log.Print("Could not read request body for this request;" + err.Error() + "\n")
		// 	return
		// }

		/* 3speed */
		ch, err := discord_session.UserChannelCreate("489166470589448220")
		if err != nil {
			wr.WriteHeader(500)
			wr.Write([]byte("Failed to reach discord API; " + err.Error()))
			log.Print("Could not reach discord api for this request;" + err.Error() + "\n")
			return
		}
		_, err = discord_session.ChannelMessageSend(ch.ID, "Submission Received:\n"+string(data))
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
		_, err = discord_session.ChannelMessageSend(ch.ID, "Submission Received:\n"+string(data))
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
	log.Printf("Listening on port %s", BOT_PORT)
	err = srv.ListenAndServe()
	log.Fatal(err.Error())
}
