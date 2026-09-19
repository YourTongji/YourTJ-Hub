const timeZone = 'Asia/Shanghai'
const hourFormat = new Intl.DateTimeFormat('en-GB', { timeZone, hour: '2-digit', hourCycle: 'h23' })
const dayFormat = new Intl.DateTimeFormat('en-US', { timeZone, weekday: 'short' })
const weekdays = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun']

export function campusClock(date: Date, locale = 'zh') {
  const hour = Number(hourFormat.format(date))
  return {
    weekday: new Intl.DateTimeFormat(locale, { timeZone, weekday: 'long' }).format(date),
    // Course selection always follows the school timezone, independent of display language.
    day: weekdays.indexOf(dayFormat.format(date)) + 1,
    date: new Intl.DateTimeFormat(locale, { timeZone, month: 'long', day: 'numeric' }).format(date),
    greeting: hour < 5 ? 'campus.greetingNight' : hour < 11 ? 'campus.greetingMorning' : hour < 14 ? 'campus.greetingNoon' : hour < 18 ? 'campus.greetingAfternoon' : 'campus.greetingEvening',
  }
}

// Keep the selected message key so switching languages also updates the current wish.
export function randomCampusWish(previous = '') {
  const choices = Array.from({ length: 16 }, (_, i) => `campus.wish${i}`).filter(key => key !== previous)
  return choices[Math.floor(Math.random() * choices.length)]!
}
