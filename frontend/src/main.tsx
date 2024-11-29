import React from 'react'
import ReactDOM from 'react-dom/client'
import Mail from './Mail'
import './index.css'
import Login from './Login'

export enum Pages {
    LOGIN = 'login',
    MAIL = 'mail',
}

function Main() {
    const initialPage = window.localStorage.getItem('page') as Pages | null
    const [page, setPage] = React.useState<Pages>(initialPage || Pages.LOGIN)

    const setPageAndStorage = (page: Pages) => {
        window.localStorage.setItem('page', page)
        setPage(page)
    }

    return (
        <>
            {page === Pages.LOGIN && <Login setPage={setPageAndStorage} />}
            {page === Pages.MAIL && <Mail setPage={setPageAndStorage} />}
        </>
    )
}

ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
    <Main />
)
