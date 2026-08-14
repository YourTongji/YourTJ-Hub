package pkservice

import (
	"reflect"
	"testing"
)

func TestSplitEndline(t *testing.T) {
	got := splitEndline("a\n\nb\n   \nc")
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("splitEndline = %v, want %v", got, want)
	}
}

func TestTimeTextToArray(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []int
	}{
		{name: "plain range", in: "1-2", want: []int{1, 2}},
		{name: "with 节 suffix", in: "5-6节", want: []int{5, 6}},
		{name: "single wide range", in: "9-10", want: []int{9, 10}},
		{name: "invalid reversed", in: "5-3", want: nil},
		{name: "invalid zero", in: "0-2", want: nil},
		{name: "invalid parse", in: "ab", want: nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := timeTextToArray(tc.in); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("timeTextToArray(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestWeekTextToArray(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []int
	}{
		{name: "plain", in: "1-16", want: seq(1, 16)},
		{name: "odd parity", in: "1-15周(单)", want: oddSeq(1, 15)},
		{name: "even parity", in: "2-14周(双)", want: evenSeq(2, 14)},
		{name: "combined", in: "2-14周(双) 15-16", want: append(evenSeq(2, 14), 15, 16)},
		{name: "single", in: "9", want: []int{9}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := weekTextToArray(tc.in); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("weekTextToArray(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestArrangementTextToObj(t *testing.T) {
	info := arrangementTextToObj("张伟(T001) 星期一1-2节[1-16周] 四平路校区 A101")
	if info.TeacherAndCode == nil || *info.TeacherAndCode != "张伟(T001)" {
		t.Fatalf("teacherAndCode = %v, want 张伟(T001)", info.TeacherAndCode)
	}
	if info.OccupyDay == nil || *info.OccupyDay != 1 {
		t.Fatalf("occupyDay = %v, want 1", info.OccupyDay)
	}
	if !reflect.DeepEqual(info.OccupyTime, []int{1, 2}) {
		t.Fatalf("occupyTime = %v, want [1 2]", info.OccupyTime)
	}
	if !reflect.DeepEqual(info.OccupyWeek, seq(1, 16)) {
		t.Fatalf("occupyWeek = %v, want [1..16]", info.OccupyWeek)
	}
	if info.OccupyRoom == nil || *info.OccupyRoom != "四平路校区 A101" {
		t.Fatalf("occupyRoom = %v, want 四平路校区 A101", info.OccupyRoom)
	}
	if info.ArrangementText != "星期一1-2节[1-16周] 四平路校区 A101" {
		t.Fatalf("arrangementText = %q", info.ArrangementText)
	}
}

func TestArrangementTextToObjEmpty(t *testing.T) {
	info := arrangementTextToObj("")
	if info.TeacherAndCode != nil || info.OccupyDay != nil || info.OccupyTime != nil || info.OccupyRoom != nil {
		t.Fatalf("empty arrangement should be all-nil, got %+v", info)
	}
}

func TestMergeArrangementInfoSorted(t *testing.T) {
	teachers := []teacherRow{
		{TeachingClassId: 1, TeacherCode: "T001", TeacherName: "张伟", ArrangeInfoText: "张伟(T001) 星期三3-4节[1-16周] A\n张伟(T001) 星期一1-2节[1-16周] B"},
		{TeachingClassId: 1, TeacherCode: "T002", TeacherName: "李娜", ArrangeInfoText: "李娜(T002) 星期二1-2节[1-16周] C"},
	}
	got := mergeArrangementInfo(teachers)
	if len(got) != 3 {
		t.Fatalf("merged length = %d, want 3", len(got))
	}
	// 按 (day, start section) 排序：星期一 < 星期二 < 星期三。
	for i, day := range []int{1, 2, 3} {
		if got[i].OccupyDay == nil || *got[i].OccupyDay != day {
			t.Fatalf("merged[%d].occupyDay = %v, want %d", i, got[i].OccupyDay, day)
		}
	}
}

func TestIsCrossDisciplineLabel(t *testing.T) {
	if !isCrossDisciplineLabel("专业选修课") {
		t.Fatal("专业选修课 should be cross-discipline")
	}
	if isCrossDisciplineLabel("专业必修") {
		t.Fatal("专业必修 should not be cross-discipline")
	}
}

func TestGetPkTimeSlotsBySection(t *testing.T) {
	tests := map[int][]int{
		1: {1, 2}, 2: {3, 4}, 3: {5, 6}, 4: {7, 8}, 5: {9}, 6: {10}, 7: nil,
	}
	for section, want := range tests {
		if got := getPkTimeSlotsBySection(section); !reflect.DeepEqual(got, want) {
			t.Fatalf("getPkTimeSlotsBySection(%d) = %v, want %v", section, got, want)
		}
	}
}

func TestBuildTimeLikePatterns(t *testing.T) {
	got := buildTimeLikePatterns(5, 1)
	want := []string{"%星期五1-2%"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildTimeLikePatterns(5,1) = %v, want %v", got, want)
	}
	if got := buildTimeLikePatterns(5, 6); len(got) != 2 {
		t.Fatalf("buildTimeLikePatterns(5,6) should have 2 patterns, got %v", got)
	}
}

func seq(a, b int) []int {
	out := make([]int, 0, b-a+1)
	for i := a; i <= b; i++ {
		out = append(out, i)
	}
	return out
}

func oddSeq(a, b int) []int {
	out := []int{}
	for i := a; i <= b; i += 2 {
		out = append(out, i)
	}
	return out
}

func evenSeq(a, b int) []int {
	out := []int{}
	for i := a; i <= b; i += 2 {
		out = append(out, i)
	}
	return out
}
