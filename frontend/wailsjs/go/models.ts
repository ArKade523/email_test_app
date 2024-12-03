export namespace imap {
	
	export class Address {
	    PersonalName: string;
	    AtDomainList: string;
	    MailboxName: string;
	    HostName: string;
	
	    static createFrom(source: any = {}) {
	        return new Address(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.PersonalName = source["PersonalName"];
	        this.AtDomainList = source["AtDomainList"];
	        this.MailboxName = source["MailboxName"];
	        this.HostName = source["HostName"];
	    }
	}
	export class Envelope {
	    // Go type: time
	    Date: any;
	    Subject: string;
	    From: Address[];
	    Sender: Address[];
	    ReplyTo: Address[];
	    To: Address[];
	    Cc: Address[];
	    Bcc: Address[];
	    InReplyTo: string;
	    MessageId: string;
	
	    static createFrom(source: any = {}) {
	        return new Envelope(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Date = this.convertValues(source["Date"], null);
	        this.Subject = source["Subject"];
	        this.From = this.convertValues(source["From"], Address);
	        this.Sender = this.convertValues(source["Sender"], Address);
	        this.ReplyTo = this.convertValues(source["ReplyTo"], Address);
	        this.To = this.convertValues(source["To"], Address);
	        this.Cc = this.convertValues(source["Cc"], Address);
	        this.Bcc = this.convertValues(source["Bcc"], Address);
	        this.InReplyTo = source["InReplyTo"];
	        this.MessageId = source["MessageId"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace mail {
	
	export class EmailBody {
	    plain: string;
	    html: string;
	
	    static createFrom(source: any = {}) {
	        return new EmailBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.plain = source["plain"];
	        this.html = source["html"];
	    }
	}
	export class SerializableMessage {
	    uid: number;
	    envelope?: imap.Envelope;
	    body: EmailBody;
	    mailbox_name: string;
	
	    static createFrom(source: any = {}) {
	        return new SerializableMessage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uid = source["uid"];
	        this.envelope = this.convertValues(source["envelope"], imap.Envelope);
	        this.body = this.convertValues(source["body"], EmailBody);
	        this.mailbox_name = source["mailbox_name"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

