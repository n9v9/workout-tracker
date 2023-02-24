import { format, getDateFormatter, getTimeFormatter } from "svelte-i18n";
import { get } from "svelte/store";

export function formatDate(date: Date): string {
    const dateFormatted = getDateFormatter({
        day: "2-digit",
        month: "2-digit",
        year: "numeric",
    }).format(date);

    const timeFormatted = getTimeFormatter({
        hour: "2-digit",
        minute: "2-digit",
    }).format(date);

    const isRelativeTo = (date: Date, relativeDay: number): boolean => {
        const today = new Date();
        return (
            date.getDate() === today.getDate() + relativeDay &&
            date.getMonth() === today.getMonth() &&
            date.getFullYear() === today.getFullYear()
        );
    };

    const formatter = get(format);

    if (isRelativeTo(date, 0)) {
        return `${formatter("date_name_today")}, ${timeFormatted}`;
    } else if (isRelativeTo(date, -1)) {
        return `${formatter("date_name_yesterday")}, ${timeFormatted}`;
    }

    for (let i = -2; i > -7; i--) {
        if (isRelativeTo(date, i)) {
            return `${getDateFormatter({
                weekday: "long",
            }).format(date)}, ${timeFormatted}`;
        }
    }

    return `${dateFormatted}, ${timeFormatted}`;
}
