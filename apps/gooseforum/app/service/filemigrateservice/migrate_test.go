package filemigrateservice

import (
	"errors"
	"testing"
)

func TestCreateMigrateTaskRejectsLocalProvider(t *testing.T) {
	// 默认存储配置为 local，迁移任务必须拒绝
	_, err := CreateMigrateTask(false)
	if err == nil {
		t.Fatal("CreateMigrateTask() error = nil with local provider, want ErrProviderNotS3")
	}
	if !errors.Is(err, ErrProviderNotS3) {
		t.Fatalf("CreateMigrateTask() error = %v, want ErrProviderNotS3", err)
	}
}
