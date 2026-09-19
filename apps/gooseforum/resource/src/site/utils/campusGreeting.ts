const timeZone = 'Asia/Shanghai'
const hourFormat = new Intl.DateTimeFormat('en-GB', { timeZone, hour: '2-digit', hourCycle: 'h23' })
const weekdayFormat = new Intl.DateTimeFormat('zh-CN', { timeZone, weekday: 'long' })
const dateFormat = new Intl.DateTimeFormat('zh-CN', { timeZone, month: 'long', day: 'numeric' })
const weekdays = ['星期一', '星期二', '星期三', '星期四', '星期五', '星期六', '星期日']

export function campusClock(date: Date) {
  const hour = Number(hourFormat.format(date))
  const weekday = weekdayFormat.format(date)
  return {
    weekday,
    day: weekdays.indexOf(weekday) + 1,
    date: dateFormat.format(date),
    greeting: hour < 5 ? '夜深了' : hour < 11 ? '早上好' : hour < 14 ? '中午好' : hour < 18 ? '下午好' : '晚上好',
  }
}

const wishes = [
  '愿你今天的灵感，比校园网信号还稳定。',
  '今日宜：吃好饭，走慢点，捡到一个小开心。',
  '生活偶尔卡顿，记得给自己一点缓存时间。',
  '为什么数学书总是不开心？因为它有太多问题。',
  '今天也要认真摸鱼——鱼说它很期待。',
  '愿你排的队，总是刚好开始变短的那一队。',
  '如果今天没什么大事，就把一顿饭吃得很开心。',
  '允许自己慢一点，树也不是一天长高的。',
  '今天的好运已在路上，可能正在等红灯。',
  '记得抬头看看天，云今天也在认真营业。',
  '愿你的咖啡温度刚好，想见的人恰好路过。',
  '别急着给今天打分，先给自己加个鸡腿。',
  '为什么电脑怕冷？因为它开着 Windows。',
  '今天也请给发呆留一个合法席位。',
  '把烦恼暂存一下，先去喝杯水。',
  '希望你今天遇见的难题，都自带一点提示。',
]

export function randomCampusWish(previous = '') {
  const choices = wishes.filter(wish => wish !== previous)
  return choices[Math.floor(Math.random() * choices.length)]!
}
