package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"

	dg "github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

var PENDING_REGISTRATIONS map[string][2]string = make(map[string][2]string)

func main() {
	var pending_file_path string = os.Args[2]
	var mkuser_script_path string = os.Args[3]
	/* Load environment variables */
	err := godotenv.Load(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	var BOT_PORT string = os.Getenv("BOT_PORT")
	var BOT_TOKEN string = os.Getenv("DISCORD_BOT_TOKEN")
	var BOT_ID string = os.Getenv("DISCORD_BOT_ID")
	var SPEED3_UID string = os.Getenv("3SPEED_UID")
	var NORA_UID string = os.Getenv("NORA_UID")
	var ADMIN_BROADCAST_UIDS = [2]string{SPEED3_UID, NORA_UID}

	discord_session, err := dg.New("Bot " + BOT_TOKEN)
	if err != nil {
		log.Fatal(err)
	}

	/* Open file of pending users and read into PENDING_REGISTRATIONS set */
	pending_file, err := os.OpenFile(pending_file_path, os.O_RDWR|os.O_CREATE, 0660)
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
		fmt.Sscanf(userinfo, "%s %s %s\n", &username, &code_and_key[0], &code_and_key[1])
		PENDING_REGISTRATIONS[username] = code_and_key
	}

	/* Handle submission from fishy.best */
	var srv_handler *http.ServeMux = http.NewServeMux()
	srv_handler.HandleFunc("POST /submit", func(wr http.ResponseWriter, rq *http.Request) {
		var body []byte = make([]byte, rq.ContentLength)
		nbytes, err := rq.Body.Read(body)
		if (err != nil || nbytes != int(rq.ContentLength)) && err != io.EOF {
			wr.WriteHeader(500)
			wr.Write([]byte("Internal I/O error with request: " + err.Error()))
			log.Println("Failed to read request " + err.Error())
			return
		}
		log.Println("REQUEST BODY: " + string(body))
		var request_values map[string]string = make(map[string]string)
		err = json.Unmarshal(body, &request_values)
		if err != nil {
			wr.WriteHeader(300)
			wr.Write([]byte("Failed to parse JSON; " + err.Error()))
			log.Println("Failed to parse JSON: " + err.Error())
			return
		}
		log.Println("----Successful Submission----")
		log.Println("USERNAME: " + request_values["username"])
		log.Println("ACCESS CODE: " + request_values["code"])
		log.Println("SSH PUBLIC KEY: " + request_values["pubkey"])
		PENDING_REGISTRATIONS[request_values["username"]] = [2]string{request_values["code"], request_values["pubkey"]}
		log.Println("Added user " + request_values["username"] + " to pending list")
		err = write_pending_to_file(pending_file)
		if err != nil {
			err2 := send_message(discord_session, SPEED3_UID, "File i/o error: "+err.Error())
			if err2 != nil {
				log.Println(err2)
			}
			return
		}
		err = broadcast_event(
			discord_session,
			"***Submission Received***\n"+
				"Username: "+request_values["username"]+"\n"+
				"Access Code: "+request_values["code"]+"\n"+
				"SSH Public Key:\n"+request_values["pubkey"],
			ADMIN_BROADCAST_UIDS)
		if err != nil {
			wr.WriteHeader(500)
			wr.Write([]byte("Failed to reach discord API; " + err.Error()))
			log.Println("Could not reach discord api for this request;" + err.Error())
			return
		}
		wr.Write([]byte("Submission received:\n" + string(body)))
		log.Println("Submission received:\n" + string(body))
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
		Name:          "info",
		Description:   "Show info for an applicant",
		Options: []*dg.ApplicationCommandOption{
			{
				Type:        dg.ApplicationCommandOptionString,
				Name:        "username",
				Description: "username of user to select",
			},
		},
	})
	if err != nil {
		log.Fatal("Failed to register info command: " + err.Error())
	}
	_, err = discord_session.ApplicationCommandCreate(BOT_ID, "", &dg.ApplicationCommand{
		ID:            "2",
		Type:          dg.ChatApplicationCommand,
		ApplicationID: BOT_ID,
		Name:          "admit",
		Description:   "Admit a currently pending applicant",
		Options: []*dg.ApplicationCommandOption{
			{
				Type:        dg.ApplicationCommandOptionString,
				Name:        "username",
				Description: "username of user to select",
			},
		},
	})
	if err != nil {
		log.Fatal("Failed to register admit command: " + err.Error())
	}
	/* Handle discord gateway interactions from slash commands */
	discord_session.AddHandler(func(s *dg.Session, i *dg.InteractionCreate) {
		switch i.Interaction.Type {
		case dg.InteractionApplicationCommand:
			data := i.Interaction.ApplicationCommandData()
			switch data.Name {
			case "help":
				err = s.InteractionRespond(i.Interaction, &dg.InteractionResponse{
					Type: dg.InteractionResponseChannelMessageWithSource,
					Data: &dg.InteractionResponseData{
						Content: "List of commands:\n" +
							"/help - show this message\n" +
							"/pending - list pending users\n" +
							"/info USERNAME - show info for a user\n" +
							"/admit USERNAME - admit pending user",
					},
				})
				if err != nil {
					log.Println("Failed to respond to help command: ", err.Error())
				}
				return
			case "pending":
				// usernames are probably <12 characters on average, plus boilerplate characters
				content := make([]byte, 16*len(PENDING_REGISTRATIONS))
				j := 1
				for username := range PENDING_REGISTRATIONS {
					content = append(content, fmt.Sprintf("%d. %s\n", j, username)...)
					j++
				}
				err = s.InteractionRespond(i.Interaction, &dg.InteractionResponse{
					Type: dg.InteractionResponseChannelMessageWithSource,
					Data: &dg.InteractionResponseData{
						Content: fmt.Sprintf("List of pending users:\n %s", content),
					},
				})
				if err != nil {
					log.Println("Failed to respond to pending command: ", err.Error())
				}
				return
			case "info":
				if data.Options == nil {
					err = s.InteractionRespond(i.Interaction, &dg.InteractionResponse{
						Type: dg.InteractionResponseChannelMessageWithSource,
						Data: &dg.InteractionResponseData{
							Content: "No username provided!",
						},
					})
					if err != nil {
						log.Println("Failed to report malformed info command: ", err.Error())
					}
					return
				}
				_username := data.GetOption("username").Value
				username, ok := _username.(string)
				if !ok {
					err = s.InteractionRespond(i.Interaction, &dg.InteractionResponse{
						Type: dg.InteractionResponseChannelMessageWithSource,
						Data: &dg.InteractionResponseData{
							Content: "Invalid option type!",
						},
					})
					if err != nil {
						log.Println("Failed to report malformed info command: ", err.Error())
					}
					return
				}
				if _, ok := PENDING_REGISTRATIONS[username]; !ok {
					err = s.InteractionRespond(i.Interaction, &dg.InteractionResponse{
						Type: dg.InteractionResponseChannelMessageWithSource,
						Data: &dg.InteractionResponseData{
							Content: "User not in pending list!",
						},
					})
					return
				}
				if err != nil {
					log.Println("Failed to respond to admission command failure: ", err.Error())
				}
				err = s.InteractionRespond(i.Interaction, &dg.InteractionResponse{
					Type: dg.InteractionResponseChannelMessageWithSource,
					Data: &dg.InteractionResponseData{
						Content: fmt.Sprintf("Username: %s\nAccess Code: %s\nSSH Public Key:\n%s",
							username,
							PENDING_REGISTRATIONS[username][0],
							PENDING_REGISTRATIONS[username][1],
						),
					},
				})
				if err != nil {
					log.Println("Failed to respond to info command: ", err.Error())
				}
				return
			case "admit":
				if data.Options == nil {
					err = s.InteractionRespond(i.Interaction, &dg.InteractionResponse{
						Type: dg.InteractionResponseChannelMessageWithSource,
						Data: &dg.InteractionResponseData{
							Content: "No username provided!",
						},
					})
					if err != nil {
						log.Println("Failed to report malformed admission command: ", err.Error())
					}
					return
				}
				_username := data.GetOption("username").Value
				username, ok := _username.(string)
				if !ok {
					err = s.InteractionRespond(i.Interaction, &dg.InteractionResponse{
						Type: dg.InteractionResponseChannelMessageWithSource,
						Data: &dg.InteractionResponseData{
							Content: "Invalid option type!",
						},
					})
					if err != nil {
						log.Println("Failed to report malformed admission command: ", err.Error())
					}
					return
				}
				if _, ok := PENDING_REGISTRATIONS[username]; !ok {
					err = s.InteractionRespond(i.Interaction, &dg.InteractionResponse{
						Type: dg.InteractionResponseChannelMessageWithSource,
						Data: &dg.InteractionResponseData{
							Content: "User not in pending list!",
						},
					})
					return
				}
				if err != nil {
					log.Println("Failed to respond to admission command failure: ", err.Error())
				}
				/*
					Here we are going to somehow call the script to onboard a user...
				*/
				script := exec.Command(mkuser_script_path, username, PENDING_REGISTRATIONS[username][1])
				// TODO: FIX FILE CREATION PERMISSIONS (SUDO?)
				err = script.Run()
				if err != nil {
					log.Println("User onboarding script failed: " + err.Error())
					err2 := s.InteractionRespond(i.Interaction, &dg.InteractionResponse{
						Type: dg.InteractionResponseChannelMessageWithSource,
						Data: &dg.InteractionResponseData{
							Content: "User onboarding script failed: " + err.Error() + ". See server backend for details.",
						},
					})
					if err2 != nil {
						log.Println("Failed to report failed onboarding script: ", err.Error())
					}
					return
				}
				// At this point we have guaranteed success
				delete(PENDING_REGISTRATIONS, username)
				write_pending_to_file(pending_file)
				err = s.InteractionRespond(i.Interaction, &dg.InteractionResponse{
					Type: dg.InteractionResponseChannelMessageWithSource,
					Data: &dg.InteractionResponseData{
						Content: fmt.Sprintf("User %s successfully admitted by command!", username),
					},
				})
				if err != nil {
					log.Println("Failed to respond to admit command: ", err.Error())
				}
				err = broadcast_event(
					discord_session,
					fmt.Sprintf("User \"%s\" admitted by %s!", username, i.Interaction.User.Username),
					ADMIN_BROADCAST_UIDS)
				if err != nil {
					log.Println("Failed to broadcast admission event" + err.Error())
				}
				return
			}

		}
	})

	err = discord_session.Open()
	if err != nil {
		log.Fatal(err.Error())
	}

	var srv http.Server = http.Server{
		Addr:    ":" + BOT_PORT,
		Handler: srv_handler,
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

func broadcast_event(dg *dg.Session, info string, admin_uids [2]string) error {
	for _, uid := range admin_uids {
		err := send_message(dg, uid, "BROADCAST: "+info)
		if err != nil {
			return err
		}
	}
	return nil
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
		_, err = fmt.Fprintf(file, "%s %s %s\n", username, code_and_key[0], code_and_key[1])
		if err != nil {
			return err
		}
	}
	return nil
}
