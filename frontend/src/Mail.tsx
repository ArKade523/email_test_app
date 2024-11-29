import './App.css'
import { useEffect, useState } from 'react'
import { GetEmailsForMailbox, GetEmailBody, GetMailboxes, LogoutUser } from "../wailsjs/go/main/App"
import { mail } from '../wailsjs/go/models'
import { EventsOn } from '../wailsjs/runtime/runtime'
import { Pages } from './main'
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome'
import { faSync } from '@fortawesome/free-solid-svg-icons'
import { faEnvelope, faFolder } from '@fortawesome/free-regular-svg-icons'
import { faEnvelope as solidFaEnvelope, faFolder as solidFaFolder } from '@fortawesome/free-solid-svg-icons'
import { formatDate } from './utls/dateUtils'

const testInboxes = [
    "INBOX",
    "Sent",
    "Drafts",
    "Trash",
    "Spam"
]

function Mail({setPage}: {setPage: (page: Pages) => void}) {
    const [mailboxes, setMailboxes] = useState<string[] | null>(testInboxes)
    const [selectedMailbox, setSelectedMailbox] = useState<string>('')
    const [emails, setEmails] = useState<mail.SerializableMessage[] | null>([])
    const [emailBody, setEmailBody] = useState<string>('')
    const [selectedEmail, setSelectedEmail] = useState<mail.SerializableMessage | null>(null)
    const [loading, setLoading] = useState<boolean>(false)
    const [mailLoading, setMailLoading] = useState<boolean>(false)
    
    const getMailboxes = async () => {
        setLoading(true)
        const newMailboxes = await GetMailboxes()
        setMailboxes(newMailboxes)
        setLoading(false)
    }

    const handleEmailClick = async (email: mail.SerializableMessage) => {
        setMailLoading(true)
        setEmailBody('')
        setSelectedEmail(email || null);
        setEmailBody(await GetEmailBody(email.mailbox_name, email.seq_num))
        setMailLoading(false)
    }

    const logOut = async () => {
        await LogoutUser()
        setPage(Pages.LOGIN)
    }

    const getEmails = async (mailbox: string) => {
        setSelectedMailbox(mailbox)
        setEmails(await GetEmailsForMailbox(mailbox))    
    }

    useEffect(() => {
        getMailboxes()

        if (mailboxes && mailboxes.length > 0) {
            if (mailboxes.includes('INBOX')) {
                setSelectedMailbox('INBOX')
                getEmails('INBOX')
            }
        }

        const unsubscribe = EventsOn("UserLoggedOut", () => {
            setPage(Pages.LOGIN)
        })

        return () => {
            unsubscribe()
        }
    }, [])
    
    return (
        <div className="max-h-screen flex py-8">
            {loading && 
                <div className="absolute h-full w-full bg-gray-300/50 grid place-items-center">
                    <span className="text-gray-400 text-2xl font-bold font-mono drop-shadow-xl">
                        Loading...
                    </span>
                </div>
            }
            <div className="flex flex-col w-max px-4 justify-between h-[95vh] whitespace-nowrap">
                <div className="flex flex-col w-max">
                    <h2 className="font-bold text-sm text-gray-100 ml-2">Inbox</h2>
                    <div className="w-max overflow-y-scroll">
                        <ul>
                            {mailboxes && mailboxes.map((mailbox, index) => (
                                <li key={index} 
                                    className={`${selectedMailbox === mailbox ? "bg-blue-700 rounded" : ""} text-gray-100 px-2 py-0.5 cursor-pointer select-none text-sm w-fill`}
                                    onClick={() => {getEmails(mailbox)}}
                                >
                                    <FontAwesomeIcon 
                                        icon={selectedMailbox === mailbox 
                                            ? (mailbox === 'INBOX' ? solidFaEnvelope : solidFaFolder) 
                                            : (mailbox === 'INBOX' ? faEnvelope : faFolder)
                                        } 
                                        className="mr-2" 
                                    />
                                    {mailbox}
                                </li>
                            ))}
                        </ul>
                    </div>
                </div>
                <div className="flex flex-col gap-1">
                    <button 
                        className="w-fit px-4 transition ease-in-out duration-300 motion-reduce:transition-none focus:bg-blue-500 hover:bg-blue-500 bg-white/20 text-white p-1 rounded"
                        onClick={getMailboxes}
                        title="refresh"
                    >
                        <FontAwesomeIcon icon={faSync} />
                    </button>
                    <button 
                        className="w-fit px-4 transition ease-in-out duration-300 motion-reduce:transition-none focus:bg-red-500 hover:bg-red-500 bg-white/20 text-white p-1 rounded"
                        onClick={logOut}
                    >
                        Log out
                    </button>
                </div>
            </div>
            <div className="max-h-full overflow-y-scroll w-80 max-w-md">
                <div className="flex flex-col px-1">
                    {emails && emails.length > 0 ? (
                    emails.map((email, index) => (
                        <div key={index} className="flex flex-col">
                            <div className={`${selectedEmail === email ? "bg-blue-700 rounded" : ""} text-xs border-b py-2 px-4 text-gray-100 border-gray-400 cursor-pointer select-none`}
                                onClick={() => handleEmailClick(email)}
                            >
                                <p className="font-bold overflow-x-hidden overflow-ellipsis whitespace-nowrap">{email.envelope?.Sender[0].PersonalName}</p>
                                <p className="overflow-x-hidden overflow-ellipsis whitespace-nowrap">{email.envelope?.Subject}</p>
                            </div>
                        </div>
                    ))) : (
                        <div className="text-gray-400 text-xl font-bold font-mono">
                            No emails in this inbox
                        </div>
                    )}
                </div>
            </div>
            <div className="max-h-full overflow-y-scroll relative w-full">
                <div className="flex flex-col p-4">
                    {selectedEmail &&
                        <div className="grid grid-cols-5 gap-2">
                            <div className="text-gray-100 col-span-3">
                                <h2 className="text-lg font-bold">{selectedEmail?.envelope?.Sender[0].PersonalName}</h2>
                                <p className="font-light text-sm">{selectedEmail?.envelope?.Subject}</p>
                            </div>
                            <div className="text-gray-100 col-span-2 content-center">
                                <p className="text-center">{formatDate(selectedEmail?.envelope?.Date)}</p>
                            </div>
                        </div>
                    }
                    <div>
                        {
                            mailLoading  ?
                            <div className="absolute w-full bg-gray-300/50 grid place-items-center h-[80vh]">
                                <span className="text-gray-400 text-2xl font-bold font-mono drop-shadow-xl">
                                    Loading...
                                </span>
                            </div>
                            : (
                            emailBody === '' ? (
                                <div className="text-gray-400 text-xl font-bold font-mono">
                                    No email selected
                                </div>
                            ) : (
                                <div className="mt-2 bg-white p-4 overflow-x-scroll overflow-y-scroll">
                                    <iframe
                                        sandbox=""
                                        srcDoc={emailBody}
                                        title="Email Content"
                                        style={{ width: '100%', height: '80vh', border: 'none' }}
                                    ></iframe>
                                </div>
                            ))
                        }
                    </div>
                </div>
            </div>
        </div>
    )
}

export default Mail
