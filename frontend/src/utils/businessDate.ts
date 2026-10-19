// "Today" and "this month" are the ones of Santiago, like in the backend (docs/tiempo.md): after
// 21:00 in Chile it is already tomorrow in UTC, which is what toISOString() gives.
const santiagoDate = new Intl.DateTimeFormat("en-CA", {
  timeZone: "America/Santiago",
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
});

/** Today's date in Santiago, as YYYY-MM-DD. */
export const todayInSantiago = (): string => santiagoDate.format(new Date());

/** The current month (1 to 12) and year in Santiago. */
export const currentMonthYear = (): { month: number; year: number } => {
  const [year, month] = todayInSantiago().split("-").map(Number);
  return { month, year };
};
