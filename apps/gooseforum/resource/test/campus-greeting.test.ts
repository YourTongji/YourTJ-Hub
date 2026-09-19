import { describe, expect, it } from 'vitest'
import { campusClock, randomCampusWish } from '../src/site/utils/campusGreeting'

describe('campus day uses school time', () => {
  it('switches to Sunday at midnight in Shanghai', () => {
    expect(campusClock(new Date('2026-09-19T16:01:00Z'))).toMatchObject({ day: 7, weekday: '星期日', date: '9月20日', greeting: '夜深了' })
    expect(campusClock(new Date('2026-09-19T00:30:00Z'))).toMatchObject({ day: 6, greeting: '早上好' })
  })
  it('changes the sentence when the user asks for another', () => {
    const first = randomCampusWish()
    expect(first).not.toBe('')
    expect(randomCampusWish(first)).not.toBe(first)
  })
})
