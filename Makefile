default:
	go build -C main
install:
	sudo -u fishybotuser cp ./main/main /usr/local/bin/fishybot/fishybot