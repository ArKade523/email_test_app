package main

import (
	"email_test_app/backend/mail"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	client, err := mail.GetClient("imap.mail.me.com:993", "kade.angell@icloud.com", os.Getenv("APPLE_APP_SPECIFIC_PASSWORD"))
	if err != nil {
		panic(err)
	}
	mailboxes, err := mail.FetchMailboxes(client)
	if err != nil {
		fmt.Printf("Error fetching mailboxes: %v", err)
		os.Exit(1)
	}
	for m := range mailboxes {
		if mailboxes[m].Name == "INBOX" {
			fmt.Println("Reading from Mailbox:", mailboxes[m].Name)
			messages, err := mail.FetchEmailsForMailbox(client, mailboxes[m].Name, 1, 10)
			if err != nil {
				fmt.Printf("Error fetching messages: %v", err)
				os.Exit(1)
			}
			for msg := range messages {
				fmt.Println("Message:", messages[msg])
			}
		} else {
			fmt.Println("Skipping Mailbox:", mailboxes[m].Name)
		}
	}
}
