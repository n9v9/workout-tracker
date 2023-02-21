export function formatDate(date: Date): string {
    const dateFormatted = date.toLocaleDateString("de", {
        day: "2-digit",
        month: "2-digit",
        year: "numeric",
    });

    const timeFormatted =
        date.toLocaleTimeString("de", {
            hour: "2-digit",
            minute: "2-digit",
        }) + " Uhr";

    const isRelativeTo = (date: Date, relativeDay: number): boolean => {
        const today = new Date();
        return (
            date.getDate() === today.getDate() + relativeDay &&
            date.getMonth() === today.getMonth() &&
            date.getFullYear() === today.getFullYear()
        );
    };

    if (isRelativeTo(date, 0)) {
        return `Heute, ${timeFormatted}`;
    } else if (isRelativeTo(date, -1)) {
        return `Gestern, ${timeFormatted}`;
    }

    for (let i = -2; i > -7; i--) {
        if (isRelativeTo(date, i)) {
            return `${date.toLocaleDateString("de", {
                weekday: "long",
            })}, ${timeFormatted}`;
        }
    }

    return `${dateFormatted}, ${timeFormatted}`;
}
