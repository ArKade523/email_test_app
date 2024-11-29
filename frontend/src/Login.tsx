import { useState } from "react";
import { LoginUser } from "../wailsjs/go/main/App";
import { Pages } from "./main";


function Login({setPage}: {setPage: (page: Pages) => void}) {
    const [imapUrl, setImapUrl] = useState('');
    const [email, setEmail] = useState('');
    const [password, setPassword] = useState('');
    const [error, setError] = useState('');

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        const success = await LoginUser(imapUrl, email, password);
        if (success) {
            setPage(Pages.MAIL)
        } else {
            setError('Invalid credentials');
        }
    }

    return (
        <div className="grid w-full h-screen place-items-center">
            <div className="p-8 bg-white/20 rounded-lg text-center text-gray-100 drop-shadow-lg">
                {error && <span className="text-red-500">{error}</span>}
                <form 
                    className="flex flex-col gap-4"
                    onSubmit={handleSubmit}
                >
                    <input 
                        type="text" 
                        autoFocus
                        className="px-2 bg-white/10 py-1 rounded focus:drop-shadow-xxl"
                        placeholder="IMAP Url" value={imapUrl} 
                        onChange={(e) => {
                            setError('');
                            setImapUrl(e.target.value);
                        }} 
                    />
                    <input 
                        type="email" 
                        className="px-2 bg-white/10 py-1 rounded focus:drop-shadow-xxl"
                        placeholder="Email" 
                        value={email} 
                        onChange={(e) => {
                            setError('');
                            setEmail(e.target.value);
                        }}
                    />
                    <input 
                        type="password" 
                        className="px-2 bg-white/10 py-1 rounded focus:drop-shadow-xxl"
                        placeholder="Password" 
                        value={password} 
                        onChange={(e) => {
                            setError('');
                            setPassword(e.target.value);
                        }}
                    />
                    <button 
                        type="submit"
                        className="transition ease-in-out duration-300 motion-reduce:transition-none focus:bg-blue-500 hover:bg-blue-500 bg-white/20 text-white p-1 rounded"
                    >
                        Login
                    </button>
                </form>
            </div>
        </div>
    );
}

export default Login;