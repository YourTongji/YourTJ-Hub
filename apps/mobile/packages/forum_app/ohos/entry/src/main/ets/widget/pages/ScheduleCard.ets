import { ScheduleCourse } from '../ScheduleData';

const storage: LocalStorage = new LocalStorage();

@Entry(storage)
@Component
struct ScheduleCard {
  @LocalStorageProp('kind') kind: string = 'nextClass';
  @LocalStorageProp('showTomorrow') showTomorrow: boolean = false;
  @LocalStorageProp('status') status: string = 'needsData';
  @LocalStorageProp('courseName') courseName: string = '';
  @LocalStorageProp('courseTime') courseTime: string = '';
  @LocalStorageProp('courseLocation') courseLocation: string = '';
  @LocalStorageProp('courseTeacher') courseTeacher: string = '';
  @LocalStorageProp('date') date: string = '';
  @LocalStorageProp('tomorrowDate') tomorrowDate: string = '';
  @LocalStorageProp('generatedAt') generatedAt: string = '';
  @LocalStorageProp('week') week: number = 0;
  @LocalStorageProp('todayCourses') todayCourses: ScheduleCourse[] = [];
  @LocalStorageProp('tomorrowCourses') tomorrowCourses: ScheduleCourse[] = [];
  @LocalStorageProp('todayKnown') todayKnown: boolean = false;
  @LocalStorageProp('tomorrowKnown') tomorrowKnown: boolean = false;
  @LocalStorageProp('transparency') transparency: number = 9;
  @LocalStorageProp('ohos.extra.param.key.form_dimension') dimension: number = 2;

  private statusLabel(): Resource {
    switch (this.status) {
      case 'inClass': return $r('app.string.widget_inClass');
      case 'upcoming': return $r('app.string.widget_next');
      case 'noneUpcoming': return $r('app.string.widget_noneUpcoming');
      case 'stale': return $r('app.string.widget_stale');
      case 'signedOut': return $r('app.string.widget_signedOut');
      case 'unbound': return $r('app.string.widget_unbound');
      case 'authorizationRequired': return $r('app.string.widget_authorizationRequired');
      default: return $r('app.string.widget_needsRefresh');
    }
  }

  private openSchedule(): void {
    postCardAction(this, {
      action: 'router',
      abilityName: 'EntryAbility',
      params: { widgetUri: this.kind === 'timeline' ? 'yourtj://campus/timetable' : 'yourtj://campus/today' }
    });
  }

  @Builder
  brand() {
    Image($r('app.media.widget_brand'))
      .width(20).height(20)
      .accessibilityLevel('no');
  }

  private courseColor(slot: number, tint: boolean): Resource {
    const colors: Resource[] = tint ? [
      $r('app.color.course_tint_1'), $r('app.color.course_tint_2'), $r('app.color.course_tint_3'),
      $r('app.color.course_tint_4'), $r('app.color.course_tint_5'), $r('app.color.course_tint_6'),
      $r('app.color.course_tint_7'), $r('app.color.course_tint_8')
    ] : [
      $r('app.color.course_slot_1'), $r('app.color.course_slot_2'), $r('app.color.course_slot_3'),
      $r('app.color.course_slot_4'), $r('app.color.course_slot_5'), $r('app.color.course_slot_6'),
      $r('app.color.course_slot_7'), $r('app.color.course_slot_8')
    ];
    return colors[Math.min(8, Math.max(1, slot)) - 1];
  }

  @Builder
  timelineCourse(course: ScheduleCourse) {
    Row({ space: 7 }) {
      Rect().width(3).height(24).radius(2).fill(this.courseColor(course.colorSlot, false));
      Column({ space: 2 }) {
        Text($r('app.string.widget_section', course.startSection.toString()))
          .fontSize(11).fontWeight(FontWeight.Bold).fontColor($r('app.color.widget_accent'));
        Text($r('app.string.widget_section', course.endSection.toString()))
          .fontSize(10).fontColor($r('app.color.widget_muted'));
      }.width(29).alignItems(HorizontalAlign.Start);
      Column({ space: 2 }) {
        Text(course.startAt.slice(11, 16)).fontSize(11).fontColor($r('app.color.widget_muted'));
        Text(course.endAt.slice(11, 16)).fontSize(10).fontColor($r('app.color.widget_muted'));
      }.width(38).alignItems(HorizontalAlign.Start);
      Column({ space: 2 }) {
        Text(course.name).fontSize(14).fontWeight(FontWeight.Bold).maxLines(2)
          .fontColor($r('app.color.widget_foreground'));
        Text([course.room, course.teacher].filter(value => !!value).join(' · '))
          .fontSize(10).maxLines(1).fontColor($r('app.color.widget_muted'));
      }.layoutWeight(1).alignItems(HorizontalAlign.Start);
    }.width('100%').padding({ left: 8, right: 8, top: 6, bottom: 6 })
      .backgroundColor(this.courseColor(course.colorSlot, true)).borderRadius(16);
  }

  @Builder
  timeline() {
    Row() {
      Text((this.showTomorrow ? this.tomorrowDate : this.date).replace(/-/g, '/'))
        .fontSize(16).fontWeight(FontWeight.Bold).fontColor($r('app.color.widget_foreground')).layoutWeight(1);
      Button(this.showTomorrow ? $r('app.string.widget_todayLabel') : $r('app.string.widget_tomorrow'))
        .height(36).fontSize(11).fontColor($r('app.color.widget_foreground'))
        .backgroundColor($r('app.color.widget_divider'))
        .accessibilityText(this.showTomorrow ? $r('app.string.widget_switchToday') : $r('app.string.widget_switchTomorrow'))
        .onClick(() => {
          postCardAction(this, { action: 'message', params: { method: 'toggleDay' } });
        });
    }.width('100%');
    if (!(this.showTomorrow ? this.tomorrowKnown : this.todayKnown)) {
      Text($r('app.string.widget_needsRefresh')).fontSize(12).fontColor($r('app.color.widget_muted'));
    } else if ((this.showTomorrow ? this.tomorrowCourses : this.todayCourses).length === 0) {
      Text($r('app.string.widget_noClasses')).fontSize(12).fontColor($r('app.color.widget_muted'));
    } else {
      List({ space: 10 }) {
        ForEach(this.showTomorrow ? this.tomorrowCourses : this.todayCourses, (course: ScheduleCourse) => {
          ListItem() { this.timelineCourse(course); }
        }, (course: ScheduleCourse) => course.stableId);
      }.width('100%').layoutWeight(1).scrollBar(BarState.Off);
    }
  }

  @Builder
  courseRow(course: ScheduleCourse) {
    Row({ space: 8 }) {
      Column({ space: 2 }) {
        Text(course.startAt.slice(11, 16)).fontSize(11).fontWeight(FontWeight.Bold).fontColor($r('app.color.widget_foreground'));
        Text(course.endAt.slice(11, 16)).fontSize(10).fontColor($r('app.color.widget_muted'));
      }.width(38).alignItems(HorizontalAlign.Start);
      Column({ space: 3 }) {
        Text(course.name).fontSize(13).fontWeight(FontWeight.Medium).maxLines(2).fontColor($r('app.color.widget_foreground'));
        if (course.room || course.campus) {
          Text([course.campus, course.room].filter(value => !!value).join(' · '))
            .fontSize(10).fontColor($r('app.color.widget_muted')).maxLines(2);
        }
      }.layoutWeight(1).alignItems(HorizontalAlign.Start);
    }
    .width('100%').padding({ top: 6, bottom: 6 })
    .alignItems(VerticalAlign.Top);
  }

  @Builder
  dayColumn(title: Resource, date: string, courses: ScheduleCourse[], known: boolean) {
    Column({ space: 4 }) {
      Text(title).fontSize(13).fontWeight(FontWeight.Bold).fontColor($r('app.color.widget_foreground'));
      Text(date.slice(5)).fontSize(10).fontColor($r('app.color.widget_muted'));
      if (!known) {
        Text($r('app.string.widget_needsRefresh'))
          .fontSize(12).fontColor($r('app.color.widget_muted')).margin({ top: 8 });
      } else if (courses.length === 0) {
        Text($r('app.string.widget_noClasses'))
          .fontSize(12).fontColor($r('app.color.widget_muted')).margin({ top: 8 });
      } else {
        ForEach(courses.slice(0, this.dimension === 4 ? 5 : 3), (course: ScheduleCourse) => {
          this.courseRow(course);
        }, (course: ScheduleCourse) => course.stableId);
        if (courses.length > (this.dimension === 4 ? 5 : 3)) {
          Text($r('app.string.widget_viewFull'))
            .fontSize(10).fontColor($r('app.color.widget_muted'));
        }
      }
    }.layoutWeight(1).alignItems(HorizontalAlign.Start);
  }

  build() {
    Stack() {
      Rect().width('100%').height('100%').radius(24)
        .fill($r('app.color.widget_background')).fillOpacity(1 - this.transparency / 100);
    Column({ space: 5 }) {
      Row() {
        Text(this.kind === 'nextClass' ? this.statusLabel() :
          this.kind === 'timeline' ? $r('app.string.widget_timeline') : $r('app.string.widget_today'))
          .fontSize(13).fontWeight(FontWeight.Bold)
          .fontColor($r('app.color.widget_accent')).layoutWeight(1).maxLines(2);
        this.brand();
      }.width('100%');
      if (!this.generatedAt) {
        Text(this.statusLabel()).fontSize(15).fontWeight(FontWeight.Bold).maxLines(3).fontColor($r('app.color.widget_foreground'));
        Text($r('app.string.widget_refresh')).fontSize(11).fontColor($r('app.color.widget_muted'));
      } else if (this.kind === 'nextClass') {
        Text(this.date.slice(5)).fontSize(11).fontColor($r('app.color.widget_muted'));
        if (this.courseName) {
          Text(this.courseName).fontSize(17).fontWeight(FontWeight.Bold).maxLines(2).fontColor($r('app.color.widget_foreground'));
          Text(this.courseTime).fontSize(12).fontColor($r('app.color.widget_accent'));
          if (this.courseLocation) {
            Text(this.courseLocation).fontSize(11).fontColor($r('app.color.widget_muted')).maxLines(2);
          }
          if (this.courseTeacher && this.dimension === 3) {
            Text(this.courseTeacher).fontSize(11).fontColor($r('app.color.widget_muted')).maxLines(1);
          }
        } else {
          Text(this.statusLabel()).fontSize(15).fontWeight(FontWeight.Bold).maxLines(2).fontColor($r('app.color.widget_foreground'));
        }
      } else if (this.kind === 'timeline') {
        this.timeline();
      } else {
        Row({ space: 12 }) {
          this.dayColumn($r('app.string.widget_todayLabel'), this.date, this.todayCourses, this.todayKnown);
          if (this.dimension === 4 && this.kind === 'today') {
            this.dayColumn($r('app.string.widget_tomorrow'), this.tomorrowDate, this.tomorrowCourses, this.tomorrowKnown);
          }
        }.width('100%').alignItems(VerticalAlign.Top);
      }
      if (this.kind !== 'timeline' || !this.generatedAt) { Blank().layoutWeight(1); }
      if (this.generatedAt) {
        Text($r('app.string.widget_updated', this.generatedAt.slice(5, 10), this.generatedAt.slice(11, 16)))
          .fontSize(10).fontColor($r('app.color.widget_muted')).maxLines(1);
      }
    }
    .padding(16).width('100%').height('100%').borderRadius(24)
    .alignItems(HorizontalAlign.Start)
    .onClick(() => this.openSchedule());
    }.width('100%').height('100%');
  }
}
