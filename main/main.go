package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	dg "github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

var PENDING_REGISTRATIONS map[string][2]string = make(map[string][2]string)

const pending_file_path string = "./pending.dat"

func main() {
	err := godotenv.Load(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	var BOT_PORT string = os.Getenv("BOT_PORT")
	var BOT_TOKEN string = os.Getenv("DISCORD_BOT_TOKEN")
	var BOT_ID string = os.Getenv("DISCORD_BOT_ID")
	var SPEED3_UID string = os.Getenv("3SPEED_UID")
	var NORA_UID string = os.Getenv("NORA_UID")
	discord_session, err := dg.New("Bot " + BOT_TOKEN)
	if err != nil {
		log.Fatal(err)
	}

	/* Open file of pending users and read into PENDING_REGISTRATIONS set */
	pending_file, err := os.OpenFile(pending_file_path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		err2 := send_message(discord_session, SPEED3_UID, "File i/o error: "+err.Error())
		if err2 != nil {
			log.Println(err2)
		}
		log.Fatal(err)
	}
	defer pending_file.Close()
	reader := bufio.NewReader(pending_file)
	for userinfo, err := reader.ReadString('\n'); err != io.EOF; userinfo, err = reader.ReadString('\n') {
		var code_and_key [2]string
		var username string
		fmt.Sscanf(userinfo, "%s\x00%s\x00%s\n", &username, &code_and_key[0], &code_and_key[1])
		PENDING_REGISTRATIONS[username] = code_and_key
	}

	var srv_handler *http.ServeMux = http.NewServeMux()
	var srv http.Server = http.Server{
		Addr:    ":" + BOT_PORT,
		Handler: srv_handler,
	}
	/* Handle submission from fishy.best */
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
		log.Print("----Successful Submission----")
		log.Print("USERNAME: " + request_values["username"])
		log.Print("ACCESS CODE: " + request_values["code"])
		log.Print("SSH PUBLIC KEY: " + request_values["pubkey"])
		/* Add user to PENDING_REGISTRATIONS set */
		PENDING_REGISTRATIONS[request_values["username"]] = [2]string{request_values["code"], request_values["pubkey"]}
		log.Print("Added user " + request_values["username"] + " to pending list")
		err = write_pending_to_file(pending_file)
		/* 3speed */
		err = send_message(discord_session, SPEED3_UID,
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
		err = send_message(discord_session, NORA_UID,
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

	/* Initialize bot slash commands */
	_, err = discord_session.ApplicationCommandCreate(BOT_ID, "", &dg.ApplicationCommand{
		ID:            "1",
		Type:          dg.ChatApplicationCommand,
		ApplicationID: BOT_ID,
		Name:          "help",
		Description:   "Show the command list with details",
	})
	if err != nil {
		log.Fatal("Failed to register help command: " + err.Error())
	}
	_, err = discord_session.ApplicationCommandCreate(BOT_ID, "", &dg.ApplicationCommand{
		ID:            "1",
		Type:          dg.ChatApplicationCommand,
		ApplicationID: BOT_ID,
		Name:          "pending",
		Description:   "Show the list of pending applicants",
	})
	if err != nil {
		log.Fatal("Failed to register pending command: " + err.Error())
	}
	_, err = discord_session.ApplicationCommandCreate(BOT_ID, "", &dg.ApplicationCommand{
		ID:            "2",
		Type:          dg.ChatApplicationCommand,
		ApplicationID: BOT_ID,
		Name:          "admit",
		Description:   "Admit a currently pending user",
		Options: []*dg.ApplicationCommandOption{
			{
				Type:        dg.ApplicationCommandOptionNumber,
				Name:        "index",
				Description: "index of user to select",
			},
		},
	})
	if err != nil {
		log.Fatal("Failed to register admit command: " + err.Error())
	}
	/* Handle discord interactions from slash commands */
	discord_session.AddHandler(func(s *dg.Session, i *dg.InteractionCreate) {
		switch i.Interaction.Type {
		case dg.InteractionApplicationCommand:
			data := i.Interaction.ApplicationCommandData()
			switch data.Name {
			case "help":
				s.InteractionRespond(i.Interaction, &dg.InteractionResponse{
					Type: dg.InteractionResponseChannelMessageWithSource,
					Data: &dg.InteractionResponseData{
						Content: "List of commands:\n" +
							"/help - show this message\n" +
							"/pending - list pending users\n" +
							"/admit N - admit pending user with index N",
					},
				})
				return
			case "pending":
				// usernames are probably <9 characters on average, plus boilerplate characters
				content := make([]byte, 14*len(PENDING_REGISTRATIONS))
				j := 1
				for username := range PENDING_REGISTRATIONS {
					content = append(content, fmt.Sprintf("%d. %s\n", j, username)...)
					j++
				}
				s.InteractionRespond(i.Interaction, &dg.InteractionResponse{
					Type: dg.InteractionResponseChannelMessageWithSource,
					Data: &dg.InteractionResponseData{
						Content: fmt.Sprintf("List of pending users:\n %s", content),
					},
				})
				return
			case "admit":
				return
			}

		}
	})
	/* Handle other discord interactions */
	discord_session.AddHandler(func(s *dg.Session, m *dg.MessageCreate) {
		if m.Author.Username == "Fishy" {
			return
		}
		err := send_message(s, m.Author.ID, "Message received in bot DM: "+m.Message.Content+"\n"+"From: "+m.Author.Username)
		if err != nil {
			log.Println("Failed to send message in response to message event; " + err.Error())
			return
		}
	})

	err = discord_session.Open()
	if err != nil {
		log.Fatal(err.Error())
	}

	log.Printf("Listening on port %s", BOT_PORT)
	srv_error := srv.ListenAndServe()
	log.Fatal(srv_error)
}

func send_message(discord_session *dg.Session, user_id string, message string) error {
	ch, err := discord_session.UserChannelCreate(user_id)
	if err != nil {
		return err
	}
	_, err = discord_session.ChannelMessageSend(ch.ID, message)
	if err != nil {
		return err
	}
	return nil
}

func admit_user() {

}

func write_pending_to_file(file *os.File) error {
	err := file.Truncate(0)
	if err != nil {
		return err
	}
	_, err = file.Seek(0, 0)
	if err != nil {
		return err
	}
	for username, code_and_key := range PENDING_REGISTRATIONS {
		_, err = fmt.Fprintf(file, "%s\x00%s\x00%s\n",
			username,
			code_and_key[0],
			code_and_key[1],
		)
		if err != nil {
			return err
		}
	}
	return nil
}
