function pad(value: number): string {
	return String(value).padStart(2, '0');
}

export function dateTimeLabel(iso: string | null | undefined): string {
	if (!iso) return 'unknown date';
	const date = new Date(iso);
	const hours = date.getHours() % 12 || 12;
	const period = date.getHours() < 12 ? 'AM' : 'PM';
	return `${pad(date.getMonth() + 1)}/${pad(date.getDate())}/${date.getFullYear()} ${hours}:${pad(date.getMinutes())} ${period}`;
}
