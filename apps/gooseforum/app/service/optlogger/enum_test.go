package optlogger

import "testing"

func TestOptEnumTargetType(t *testing.T) {
	tests := []struct {
		name       string
		value      OptEnum
		wantTarget TargetTypeEnum
	}{
		{name: "edit user", value: EditUser, wantTarget: User},
		{name: "edit topic", value: EditTopic, wantTarget: Topic},
		{name: "unknown", value: OptEnum(99), wantTarget: System},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.value.TargetTypeEnum(); got != tt.wantTarget {
				t.Fatalf("TargetTypeEnum() = %v, want %v", got, tt.wantTarget)
			}
		})
	}
}

func TestEnumToInt(t *testing.T) {
	if got := EditTopic.toInt(); got != 1 {
		t.Fatalf("OptEnum.toInt() = %d, want 1", got)
	}
	if got := TargetTypeEnum(Category).toInt(); got != 6 {
		t.Fatalf("TargetTypeEnum.toInt() = %d, want 6", got)
	}
}
