package tj.yourtj.forum_app.widget

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test
import java.time.Instant
import java.util.Date

class ScheduleWidgetDataTest {
    private val generatedAt = Date.from(Instant.parse("2026-09-22T00:00:00Z"))
    private val staleAt = Date(generatedAt.time + 7L * 24 * 60 * 60 * 1000)
    private val course = ScheduleCourse(
        id = "future-course",
        name = "Future course",
        teacher = "",
        campus = "",
        room = "",
        startAt = Date(staleAt.time + 60 * 60 * 1000),
        endAt = Date(staleAt.time + 2 * 60 * 60 * 1000),
        colorSlot = 1,
    )
    private val projection = ScheduleProjection(
        generatedAt = generatedAt,
        days = listOf(
            ScheduleDay(
                date = "2026-09-29",
                week = 2,
                source = "local-resolved",
                kind = "normal",
                adjustmentLabel = null,
                courses = listOf(course),
            ),
        ),
    )

    @Test
    fun staleProjectionWithholdsCoursesAndCourseWakeups() {
        val now = Date(staleAt.time + 1)

        assertEquals("stale", projection.status(now).first)
        assertEquals("stale", projection.nextClass(now).status)
        assertNull(projection.nextClass(now).course)
        assertNull(projection.dayFor(now))
        assertNull(projection.dayOn("2026-09-29", now))
        assertEquals(nextShanghaiMidnight(now), projection.nextUpdateAt(now))
    }

    @Test
    fun projectionBecomesStaleAtTheExactBoundary() {
        assertEquals("stale", projection.status(staleAt).first)
        assertEquals("stale", projection.nextClass(staleAt).status)
    }

    @Test
    fun freshProjectionRetainsDayReads() {
        val now = Date(staleAt.time - 60 * 60 * 1000)

        assertEquals("2026-09-29", projection.dayFor(now)?.date)
        assertEquals("upcoming", projection.nextClass(now).status)
    }
}
