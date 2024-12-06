import './App.css'
import React, { useEffect } from 'react'
import ReactDOM from 'react-dom/client'
import Mail from './Mail'
import './index.css'
import Login from './Login'
import { GetAccountIds, IsLoggedIn } from '../wailsjs/go/wails_app/App'

export enum Pages {
    LOGIN = 'login',
    MAIL = 'mail',
}

function Main() {
    const initialPage = window.localStorage.getItem('page') as Pages | null
    const [page, setPage] = React.useState<Pages>(initialPage || Pages.LOGIN)
    const [accountIds, setAccountIds] = React.useState<number[]>([])

    const setPageAndStorage = (page: Pages) => {
        window.localStorage.setItem('page', page)
        setPage(page)
    }

    useEffect(() => {
        const init = async () => {
            setAccountIds(await GetAccountIds())

            for (const accountId of accountIds) {
                if (!await IsLoggedIn(accountId)) {
                    setAccountIds([...accountIds.filter(id => id !== accountId)])
                }
            }

            if (accountIds.length === 0) {
                setPage(Pages.LOGIN)
            }
        }
        init()
    }, [])

    return (
        <div className={`${navigator.userAgent.includes('Chrome') && "bg-black"}`}>
            {page === Pages.LOGIN && <Login accountIds={accountIds} setAccountIds={setAccountIds} setPage={setPageAndStorage} />}
            {page === Pages.MAIL && <Mail accounts={accountIds} setPage={setPageAndStorage} />}
        </div>
    )
}

ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
    <Main />
)
