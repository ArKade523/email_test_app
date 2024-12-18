// Cynhyrchwyd y ffeil hon yn awtomatig. PEIDIWCH Â MODIWL
// This file is automatically generated. DO NOT EDIT
import {mail} from '../models';

export function GetAccountIds():Promise<Array<number>>;

export function GetEmailBody(arg1:number,arg2:string,arg3:number):Promise<string>;

export function GetEmailsForMailbox(arg1:number,arg2:string,arg3:number,arg4:number):Promise<Array<mail.SerializableMessage>>;

export function GetMailboxes(arg1:number):Promise<Array<string>>;

export function IsLoggedIn(arg1:number):Promise<boolean>;

export function LoginUser(arg1:string,arg2:string,arg3:string):Promise<number>;

export function LoginUserWithOAuth(arg1:string):Promise<boolean>;

export function LogoutUser(arg1:number):Promise<void>;

export function StartOAuth(arg1:string):Promise<void>;

export function UpdateMailboxes(arg1:number):Promise<void>;

export function UpdateMessages(arg1:number,arg2:string):Promise<void>;
