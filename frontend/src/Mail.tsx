import { useEffect, useRef, useState } from 'react'
import { GetEmailsForMailbox, GetEmailBody, GetMailboxes, LogoutUser, UpdateMailboxes, UpdateMessages } from "../wailsjs/go/wails_app/App"
import { mail } from '../wailsjs/go/models'
import { EventsOn } from '../wailsjs/runtime/runtime'
import { Pages } from './main'
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome'
import { faEnvelopeOpen, faFileContract, faFolderOpen, faRightFromBracket, faSync, faTrashAlt, IconDefinition } from '@fortawesome/free-solid-svg-icons'
import { faEnvelope, faFile, faFolder, faPaperPlane, faTrashCan } from '@fortawesome/free-regular-svg-icons'
import { faPaperPlane as faPaperPlaneSolid } from '@fortawesome/free-solid-svg-icons'
import { formatDate } from './utils/dateUtils'

const knownMailboxIcons: { [key: string]: [IconDefinition, IconDefinition] } = {
    "INBOX": [faEnvelope, faEnvelopeOpen],
    "Sent": [faPaperPlane, faPaperPlaneSolid],
    "Trash": [faTrashCan, faTrashAlt],
    "Drafts": [faFile, faFileContract]
}

const NUM_EMAILS_TO_FETCH = 20

function Mail({setPage}: {setPage: (page: Pages) => void}) {
    const [mailboxes, setMailboxes] = useState<string[] | null>(['INBOX'])
    const [selectedMailbox, setSelectedMailbox] = useState<string>('')
    const [emails, setEmails] = useState<mail.SerializableMessage[] | null>([])
    const [emailBody, setEmailBody] = useState<string>('')
    const [selectedEmail, setSelectedEmail] = useState<mail.SerializableMessage | null>(null)
    const [loading, setLoading] = useState<boolean>(false)
    const [mailLoading, setMailLoading] = useState<boolean>(false)
    const emailListRef = useRef<HTMLDivElement>(null)

    const emailsPerInbox = useRef<{ [key: string]: mail.SerializableMessage[] }>({})
    
    const getMailboxes = async () => {
        setLoading(true)
        const newMailboxes = await GetMailboxes()
        if (newMailboxes && newMailboxes.length > 0) {
            if (newMailboxes.includes('INBOX')) {
                setSelectedMailbox('INBOX')
                getEmails('INBOX')
            }

            // Sort the mailboxes so that mailboxes in the knownMailboxIcons object are displayed first
            const knownMailboxes = Object.keys(knownMailboxIcons)
            const sortedMailboxes = newMailboxes.sort((a, b) => {
                if (knownMailboxes.includes(a) && knownMailboxes.includes(b)) {
                    return knownMailboxes.indexOf(a) - knownMailboxes.indexOf(b)
                } else if (knownMailboxes.includes(a)) {
                    return -1
                } else if (knownMailboxes.includes(b)) {
                    return 1
                }
                return a.localeCompare(b)
            })

            setMailboxes(sortedMailboxes)
        } 

        setLoading(false)
    };

    const handleEmailClick = async (email: mail.SerializableMessage) => {
        if (selectedEmail === email) {
            return
        }
        if (mailLoading) {
            return
        }
        setMailLoading(true)
        setEmailBody('')
        setSelectedEmail(email || null);
        setEmailBody(await GetEmailBody(email.mailbox_name, email.uid))
        setMailLoading(false)
    }

    const logOut = async () => {
        await LogoutUser()
        console.log('Logged out')
        setPage(Pages.LOGIN)
    }

    const getEmails = async (mailbox: string) => {
        if (selectedMailbox === mailbox) {
            return
        }
        setSelectedMailbox(mailbox)
        if (!emailsPerInbox.current[mailbox]) {
            emailsPerInbox.current[mailbox] = []
        }
        const numEmails = emailsPerInbox.current[mailbox].length || 0
        const newEmails = await GetEmailsForMailbox(mailbox, numEmails, numEmails + NUM_EMAILS_TO_FETCH)
        if (newEmails) {
            emailsPerInbox.current[mailbox].push(...newEmails)
        }
        setEmails(emailsPerInbox.current[mailbox])
    }

    const formatMailboxName = (mailbox: string) => {
        if (mailbox === 'INBOX') {
            return 'Inbox'
        }
        if (mailbox.includes('[Gmail]/')) {
            return mailbox.replace('[Gmail]/', '')
        }
        if (mailbox.includes('[Gmail]')) {
            if (mailbox === '[Gmail]') {
                const newMailboxes = mailboxes?.filter((m) => m !== mailbox) || []
                setMailboxes(newMailboxes)
            }
            return mailbox.replace('[Gmail]', '')
        }
        return mailbox
    }

    useEffect(() => {
        getMailboxes()

        emailsPerInbox.current = {}

        let unsubscribeFunctions = [] as (() => void)[]

        unsubscribeFunctions.push(EventsOn("UserLoggedOut", () => {
            setPage(Pages.LOGIN)
        }))
        unsubscribeFunctions.push(EventsOn("MailboxesUpdated", () => {
            getMailboxes()
        }))
        unsubscribeFunctions.push(EventsOn("MessagesUpdated", (mailboxName: string) => {
            if (selectedMailbox === mailboxName) {
                getEmails(mailboxName)
            }
        }))

        return () => {
            for (const unsubscribe of unsubscribeFunctions) {
                unsubscribe()
            }
        }
    }, [])

    useEffect(() => {
        const handleMailListScroll = () => {
            if (emailListRef.current) {
                const { scrollTop, scrollHeight, clientHeight } = emailListRef.current
                if (scrollTop + clientHeight >= scrollHeight) {
                    getEmails(selectedMailbox)
                }
            }
            console.log('Scrolled')
        }

        const emailListElement = emailListRef.current
        if (emailListElement) {
            emailListElement.addEventListener('scroll', handleMailListScroll);
            return () => {
                emailListElement.removeEventListener('scroll', handleMailListScroll);
            };
        }
    }, [emailListRef, selectedMailbox])

    
    return (
        <div className="max-h-screen flex py-8">
            {loading && 
                <div className="absolute top-0 h-full w-full bg-gray-300/50 grid place-items-center">
                    <span className="text-gray-400 text-2xl font-bold font-mono drop-shadow-xl">
                        Loading...
                    </span>
                </div>
            }
            <div className="flex flex-col w-max px-4 justify-between h-[95vh] whitespace-nowrap">
                <div className="flex flex-col w-max">
                    <div className="flex justify-between px-2">
                        <h2 className="font-bold text-xs text-gray-100 select-none">Mailboxes</h2>
                        <button 
                            className="transition ease-in-out duration-300 motion-reduce:transition-none hover:text-blue-500 text-gray-300 text-xs"
                            onClick={UpdateMailboxes}
                            title="Refresh Mailboxes"
                        >
                            <FontAwesomeIcon icon={faSync} />
                        </button>
                    </div>
                    <div className="w-max overflow-y-scroll">
                        <ul>
                            {mailboxes && mailboxes.map((mailbox, index) => (
                                <li key={index} 
                                    className={`${selectedMailbox === mailbox ? "bg-blue-700 rounded" : ""} text-gray-100 px-2 py-0.5 cursor-pointer select-none text-sm w-fill`}
                                    onClick={() => {getEmails(mailbox)}}
                                >
                                    <FontAwesomeIcon 
                                        icon={selectedMailbox === mailbox ? 
                                            knownMailboxIcons[mailbox] ? knownMailboxIcons[mailbox][1] : faFolderOpen : 
                                            knownMailboxIcons[mailbox] ? knownMailboxIcons[mailbox][0] : faFolder} 
                                        className="text-gray-300 mr-2" 
                                    />
                                    {formatMailboxName(mailbox)}
                                </li>
                            ))}
                        </ul>
                    </div>
                </div>
                <div className="flex flex-col gap-1">
                    <button 
                        className="w-fit px-2 text-xs transition ease-in-out duration-300 motion-reduce:transition-none border-2 border-gray-400 focus:border-red-400  focus:bg-red-500 hover:bg-red-500 hover:border-red-400 bg-white/20 text-white p-1 rounded"
                        onClick={logOut}
                    >
                        Log out
                        <FontAwesomeIcon icon={faRightFromBracket} className="text-gray-300 ml-2" />
                    </button>
                </div>
            </div>
            <div className="max-h-full overflow-y-scroll w-80 max-w-md">
                <div className="flex justify-between px-2">
                    <h2 className="font-bold text-xs text-gray-100 ml-2 select-none">Messages</h2>
                    <button 
                        className="transition ease-in-out duration-300 motion-reduce:transition-none hover:text-blue-500 text-gray-300 text-xs"
                        onClick={() => UpdateMessages(selectedMailbox)}
                        title="Refresh Messages"
                    >
                        <FontAwesomeIcon icon={faSync} />
                    </button>
                    </div>
                <div className="flex flex-col px-1"
                    ref={emailListRef}
                >
                    {emails && emails.length > 0 ? (
                    emails.map((email, index) => (
                        <div key={index} className="flex flex-col">
                            <div className={`${selectedEmail === email ? "bg-blue-700 rounded" : ""} text-xs py-2 px-4 text-gray-100 cursor-pointer select-none`}
                                onClick={() => handleEmailClick(email)}
                            >
                                <p className="font-bold overflow-x-hidden overflow-ellipsis whitespace-nowrap">{email.envelope?.Sender?.[0]?.PersonalName || "Unknown Sender"}</p>
                                <p className="overflow-x-hidden overflow-ellipsis whitespace-nowrap">{email.envelope?.Subject}</p>
                            </div>
                            {
                                // Add a horizontal line between emails. No line on the last email
                                index !== emails.length - 1 && <span className="h-[1px] my-[2px] w-[90%] bg-gray-400 self-center"></span>
                            }
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
                                <h2 className="text-lg font-bold">{selectedEmail?.envelope?.Sender?.[0]?.PersonalName || "Unknown Sender"}</h2>
                                <p className="font-light text-sm">{selectedEmail?.envelope?.Subject}</p>
                            </div>
                            <div className="text-gray-100 col-span-2 content-center">
                                <p className="text-right text-sm select-none">{formatDate(selectedEmail?.envelope?.Date)}</p>
                            </div>
                        </div>
                    }
                    <div>
                        {
                            mailLoading  ?
                                <div className="text-gray-400 text-xl h-[80vh] text-center content-center select-none font-bold font-mono">
                                    Loading...
                                </div>
                            : (
                            emailBody === '' ? (
                                <div className="text-gray-400 text-xl h-[80vh] text-center content-center select-none font-bold font-mono">
                                    No message selected
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
