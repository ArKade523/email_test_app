import { simpleParser } from 'mailparser';

export async function parseEmailBody(rawEmail: string): Promise<string> {
    try {
        const parsed = await simpleParser(rawEmail);
        // Prefer HTML content if available
        if (parsed.html) {
            return parsed.html;
        } else if (parsed.textAsHtml) {
            // Fallback to text converted to HTML
            return parsed.textAsHtml;
        } else if (parsed.text) {
            // Fallback to plain text
            return parsed.text;
        }
    } catch (error) {
        console.error('Error parsing email:', error);
    }
    return '';
}
