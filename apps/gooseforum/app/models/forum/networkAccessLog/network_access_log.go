package networkAccessLog

import "time"

const tableName = "network_access_logs"

// Retention 网络访问日志最低保留期（≥6 个月，取 183 天）。
const Retention = 183 * 24 * time.Hour

// Entity 持久化 HTTP 访问记录，满足合规对网络日志 ≥6 个月留存的要求。
type Entity struct {
	Id        uint64    `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	Method    string    `gorm:"column:method;type:varchar(16);not null;default:'';" json:"method"`
	Path      string    `gorm:"column:path;type:varchar(512);not null;default:'';" json:"path"`
	Route     string    `gorm:"column:route;type:varchar(256);not null;default:'';" json:"route"`
	Status    int       `gorm:"column:status;not null;default:0;" json:"status"`
	UserId    uint64    `gorm:"column:user_id;not null;default:0;index:idx_network_access_logs_user,priority:1;" json:"userId"`
	ClientIP  string    `gorm:"column:client_ip;type:varchar(64);not null;default:'';" json:"clientIp"`
	LatencyMs int64     `gorm:"column:latency_ms;not null;default:0;" json:"latencyMs"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime;<-:create;index:idx_network_access_logs_created,priority:1;" json:"createdAt"`
}

func (itself *Entity) TableName() string {
	return tableName
}
