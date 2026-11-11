export const TIME_OPTIONS = Array.from({ length: 48 }, (_, index) => {
  const hours = Math.floor(index / 2);
  const minutes = index % 2 === 0 ? "00" : "30";
  return `${String(hours).padStart(2, "0")}:${minutes}`;
});

export function timeRangeMinutes(startTime: string, endTime: string): number {
  const [startHours, startMinutes] = startTime.split(":").map(Number);
  const [endHours, endMinutes] = endTime.split(":").map(Number);
  return endHours * 60 + endMinutes - (startHours * 60 + startMinutes);
}

export function currentMonth(): string {
  const now = new Date();
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}`;
}

export function todayKey(): string {
  const now = new Date();
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}-${String(
    now.getDate(),
  ).padStart(2, "0")}`;
}

export function oneMonthFromToday(): string {
  const now = new Date();
  const max = new Date(now.getFullYear(), now.getMonth() + 1, now.getDate());
  return `${max.getFullYear()}-${String(max.getMonth() + 1).padStart(2, "0")}-${String(
    max.getDate(),
  ).padStart(2, "0")}`;
}

export function formatDate(dateString: string): string {
  return new Intl.DateTimeFormat("zh-CN", {
    month: "long",
    day: "numeric",
    weekday: "short",
  }).format(new Date(`${dateString}T00:00:00`));
}

export function formatShortDate(dateString: string): string {
  const date = new Date(`${dateString}T00:00:00`);
  return `${date.getMonth() + 1}月${date.getDate()}日`;
}

export function formatDateRange(startDateString: string, endDateString: string): string {
  const [startYear, startMonth, startDay] = startDateString.split("-");
  const [endYear, endMonth, endDay] = endDateString.split("-");
  const start = `${startMonth}.${startDay}`;
  const end = `${endMonth}.${endDay}`;

  if (startDateString === endDateString) return `${startYear}.${start}`;
  if (startYear === endYear) return `${startYear}.${start}–${end}`;
  return `${startYear}.${start}–${endYear}.${end}`;
}

export function endTime(startTime: string, duration: number): string {
  const [hours, minutes] = startTime.split(":").map(Number);
  const total = hours * 60 + minutes + duration;
  return `${String(Math.floor(total / 60) % 24).padStart(2, "0")}:${String(
    total % 60,
  ).padStart(2, "0")}`;
}

export function unitsOf(minutes: number): number {
  return minutes / 60;
}

export function durationUnitLabel(minutes: number): string {
  return `${formatNumber(unitsOf(minutes))}课时`;
}


export function formatNumber(value: number): string {
  return Number(value).toLocaleString("zh-CN", { maximumFractionDigits: 1 });
}
