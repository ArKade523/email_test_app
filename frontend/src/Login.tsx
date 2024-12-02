import { useEffect, useState } from "react";
import { LoginUser, LoginUserWithOAuth } from "../wailsjs/go/wails_app/App";
import { Pages } from "./main";
import { emailProviders, EmailProvider } from "./utils/emailProviders"; // Update the path accordingly
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faChevronLeft } from "@fortawesome/free-solid-svg-icons";
import { EventsOn } from "../wailsjs/runtime/runtime";

function Login({ setPage }: { setPage: (page: Pages) => void }) {
    const [step, setStep] = useState(1);
    const [selectedProvider, setSelectedProvider] = useState<EmailProvider | null>(null);
    const [showCustomImap, setShowCustomImap] = useState(false);
    const [imapUrl, setImapUrl] = useState('');
    const [email, setEmail] = useState('');
    const [password, setPassword] = useState('');
    const [error, setError] = useState('');

    const clickProvider = (provider: EmailProvider) => {
        setError('');
        if (provider.requiresOAuth) {
            LoginUserWithOAuth(provider.name)
                .then((status) => {
                    console.log('OAuth login status:', status);
                    // No need to call setPage here
                    if (!status) {
                        setError('OAuth login failed.');
                    }
                })
                .catch((err) => {
                    setError('OAuth login failed.');
                });
        } else {
            setSelectedProvider(provider);
            setShowCustomImap(provider.name === 'Custom');
            if (provider.name !== 'Custom') {
                setImapUrl('');
            }
            setStep(2); // Proceed to next step
        }
    };

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();

        let imapUrlToUse = imapUrl;
        if (selectedProvider && !showCustomImap) {
            // Construct the IMAP URL using the provider's settings
            imapUrlToUse = `${selectedProvider.imapHost}:${selectedProvider.imapPort}`;
        }

        if (!imapUrlToUse) {
            setError('Please enter the IMAP URL.');
            return;
        }

        const success = await LoginUser(imapUrlToUse, email, password);
        if (success) {
            setPage(Pages.MAIL);
        } else {
            setError('Invalid credentials');
        }
    };

    useEffect(() => {
        let unsubscribeFunctions: (() => void)[] = [];
        // Set up event listener for OAuthSuccess
        unsubscribeFunctions.push(EventsOn("OAuthSuccess", () => {
            console.log("Received OAuthSuccess event");
            setPage(Pages.MAIL);
        }))

        // Optionally, handle OAuthFailure if you emit it
        unsubscribeFunctions.push(EventsOn("OAuthFailure", () => {
            console.log("Received OAuthFailure event");
            setError("OAuth login failed.");
        }))

        // Clean up the event listeners when the component unmounts
        return () => {
            for (const unsubscribe of unsubscribeFunctions) {
                unsubscribe();
            }
        };
    }, []);

    // Step 1: Provider Selection
    if (step === 1) {
        return (
            <div className="grid w-full h-screen place-items-center">
                <div className="p-8 bg-white/20 rounded-lg text-center text-gray-100 drop-shadow-lg">
                    <h2 className="text-xl mb-4">Select Your Email Provider</h2>
                    {error && <span className="text-red-500">{error}</span>}
                    <div className="flex flex-col gap-2">
                        {emailProviders.map((provider) => (
                            <button
                                key={provider.name}
                                className="transition ease-in-out duration-300 motion-reduce:transition-none focus:bg-blue-500 hover:bg-blue-500 bg-white/20 text-white p-2 rounded"
                                onClick={() => clickProvider(provider)}
                            >
                                {provider.faIcon && <FontAwesomeIcon icon={provider.faIcon} className="mr-2 text-xl" />}
                                {provider.name}
                            </button>
                        ))}
                    </div>
                </div>
            </div>
        );
    }

    // Step 2: Credentials Entry
    if (step === 2) {
        return (
            <div className="grid w-full h-screen place-items-center">
                <div className="p-8 bg-white/20 rounded-lg text-center text-gray-100 drop-shadow-lg">
                    <h2 className="text-xl mb-4">Enter Your Credentials</h2>
                    {error && <span className="text-red-500 w-12">{error}</span>}
                    <form 
                        className="flex flex-col gap-4"
                        onSubmit={handleSubmit}
                    >
                        {showCustomImap && (
                            <input 
                                type="text" 
                                className="px-2 bg-white/10 py-1 rounded focus:drop-shadow-xxl"
                                placeholder="IMAP URL (e.g., imap.example.com:993)" 
                                value={imapUrl} 
                                onChange={(e) => {
                                    setError('');
                                    setImapUrl(e.target.value);
                                }} 
                                required
                            />
                        )}

                        <input 
                            type="email" 
                            autoFocus
                            className="px-2 bg-white/10 py-1 rounded focus:drop-shadow-xxl"
                            placeholder="Email" 
                            value={email} 
                            onChange={(e) => {
                                setError('');
                                setEmail(e.target.value);
                            }}
                            required
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
                            required
                        />
                        <button 
                            type="submit"
                            className="transition ease-in-out duration-300 motion-reduce:transition-none focus:bg-blue-500 hover:bg-blue-500 bg-white/20 text-white p-1 rounded"
                        >
                            Login
                        </button>
                    </form>
                    <button
                        className="transition ease-in-out duration-300 motion-reduce:transition-none mt-4 text-sm text-gray-300 hover:text-blue-400 hover:underline"
                        onClick={() => {
                            setStep(1);
                            setEmail('');
                            setPassword('');
                            setError('');
                        }}
                    >
                        <FontAwesomeIcon icon={faChevronLeft} className="mr-2" />
                        Back to Provider Selection
                    </button>
                </div>
            </div>
        );
    }

    return null; // Just in case
}

export default Login;
