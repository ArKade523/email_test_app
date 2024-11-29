
// example date: 2024-11-29T20:56:00Z
export function formatDate(date: string): string {
    const d = new Date(date)
    return d.toLocaleString(
        undefined,
        {
            year: '2-digit',
            month: 'numeric',
            day: 'numeric',
        }
    )
}