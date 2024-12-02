import { IconDefinition } from "@fortawesome/fontawesome-svg-core"
import { faApple, faGoogle, faMicrosoft, faYahoo } from "@fortawesome/free-brands-svg-icons"

export type EmailProvider = {
    name: string
    imapHost: string
    imapPort: number
    requiresOAuth: boolean
    faIcon?: IconDefinition
}

export const emailProviders: EmailProvider[] = [
    {name: "Gmail", imapHost: "imap.gmail.com", imapPort: 993, requiresOAuth: true, faIcon: faGoogle},
    {name: "Outlook.com", imapHost: "outlook.office365.com", imapPort: 993, requiresOAuth: true, faIcon: faMicrosoft},
    {name: "Yahoo", imapHost: "imap.mail.yahoo.com", imapPort: 993, requiresOAuth: false, faIcon: faYahoo},
    {name: "iCloud", imapHost: "imap.mail.me.com", imapPort: 993, requiresOAuth: false, faIcon: faApple},
    {name: "AOL", imapHost: "imap.aol.com", imapPort: 993, requiresOAuth: false},
    {name: "Custom", imapHost: "", imapPort: 993, requiresOAuth: false, },
]
