package users

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/leancodebox/GooseForum/app/bundles/algorithm"
	"github.com/leancodebox/GooseForum/app/bundles/pageutil"
	"github.com/leancodebox/GooseForum/app/bundles/queryopt"
	"github.com/samber/lo"
	"gorm.io/gorm"
)

// dummyBotHashForTiming is a fixed PBKDF2-SHA256 hash:salt value used to
// equalize password-verification timing when a bot account is presented at
// the human login endpoint. It runs the same 10000-iteration cost as real
// hashes so username enumeration via response time is not possible.
const dummyBotHashForTiming = "kQZbUFLHdF2p4QRUQ7kVsv6r0LqRFiYBoK9V0MaRTzc=:eW91cnRqLWJvdC10aW1pbmctc2FsdC0wMTIzNDU2Nzg5YWJjZGVm"

// ErrInvalidCredentials is the single error returned for every failed
// password verification (unknown user, bot account, or wrong password), so
// response bodies and logs never distinguish bot accounts from humans.
var ErrInvalidCredentials = errors.New("invalid username or password")

func Get(id any) (entity EntityComplete, err error) {
	err = builder().Where(pid, id).First(&entity).Error
	return
}

func Verify(usernameOrEmail string, password string) (*EntityComplete, error) {
	var user EntityComplete
	var err error
	if strings.Contains(usernameOrEmail, "@") {
		err = builder().Where("email = ?", usernameOrEmail).First(&user).Error
	} else {
		err = builder().Where("username = ?", usernameOrEmail).First(&user).Error
	}
	if err != nil {
		return &user, err
	}
	// 机器人（Agent）账号不参与密码登录。先执行与真实校验等量的 PBKDF2
	// 以抹平响应时间，再返回与错误密码完全相同的错误，避免枚举 bot 账号，
	// 也与“bot 邮箱为空、密码不可用”的约束一致。
	if user.IsBot() {
		_ = algorithm.VerifyEncryptPassword(dummyBotHashForTiming, password)
		return &EntityComplete{}, ErrInvalidCredentials
	}
	err = algorithm.VerifyEncryptPassword(user.Password, password)
	if err != nil {
		return &EntityComplete{}, ErrInvalidCredentials
	}
	return &user, nil
}

// GetByEmail 通过邮箱获取用户
func GetByEmail(email string) (entity EntityComplete, err error) {
	err = builder().Where("email = ?", email).First(&entity).Error
	return
}

func GetByUsername(username string) (entity EntityComplete, err error) {
	err = builder().Where("username = ?", username).First(&entity).Error
	return
}

func MakeUser(name string, password string, email string) *EntityComplete {
	user := EntityComplete{Username: name, Email: email}
	user.SetPassword(password)
	user.AvatarUrl = RandAvatarUrl()
	return &user
}

func RandAvatarUrl() string {
	randomNum := rand.Intn(12) + 1
	return fmt.Sprintf("/static/pic/%d.webp", randomNum)
}

func Create(entity *EntityComplete) error {
	return builder().Create(&entity).Error
}

func Save(entity *EntityComplete) error {
	result := builder().Save(entity)
	return result.Error
}

func UpdateWornBadgeCode(userID uint64, badgeCode string) error {
	return builder().
		Where(queryopt.Eq(pid, userID)).
		Update("worn_badge_code", badgeCode).Error
}

// CloseAccount 注销账号（PRD R10）：软删用户并清空对外展示字段。
// 历史内容仍保留 userId 指向，渲染层因用户不可见而回退为「已注销用户」。
func CloseAccount(userID uint64) error {
	return builder().Unscoped().Where(queryopt.Eq(pid, userID)).Updates(map[string]any{
		"deleted_at":      time.Now(),
		"worn_badge_code": "",
	}).Error
}

// IsAccountClosed 判断账号是否已注销（软删）。
func IsAccountClosed(userID uint64) bool {
	var entity EntityComplete
	err := builder().Unscoped().Where(queryopt.Eq(pid, userID)).First(&entity).Error
	if err != nil || entity.Id == 0 {
		return false
	}
	return entity.DeletedAt.Valid
}

func All() (entities []*EntityComplete) {
	builder().Find(&entities)
	return
}

func GetMaxId() uint64 {
	var entity EntityComplete
	builder().Order(queryopt.Desc(pid)).Limit(1).First(&entity)
	return entity.Id
}

type PageQuery struct {
	Page, PageSize int
	Username       string
	UserId         uint64
	Email          string
}

func Page(q PageQuery) struct {
	Page     int
	PageSize int
	Total    int64
	Data     []EntityComplete
} {
	var list []EntityComplete
	q.Page = max(q.Page-1, 0)
	q.PageSize = pageutil.BoundPageSize(q.PageSize)
	b := builder()
	cB := builder()
	if q.Username != "" {
		b.Where(queryopt.Like(fieldUsername, q.Username))
		cB.Where(queryopt.Like(fieldUsername, q.Username))
	}
	if q.Email != "" {
		b.Where(queryopt.Like(fieldEmail, q.Email))
		cB.Where(queryopt.Like(fieldEmail, q.Email))
	}
	if q.UserId != 0 {
		b.Where(queryopt.Eq(pid, q.UserId))
		cB.Where(queryopt.Eq(pid, q.UserId))
	}
	b.Limit(q.PageSize).Offset(q.PageSize * q.Page).Order(queryopt.Desc(pid)).Find(&list)

	var total int64
	cB.Count(&total)

	return struct {
		Page     int
		PageSize int
		Total    int64
		Data     []EntityComplete
	}{Page: q.Page, PageSize: q.PageSize, Data: list, Total: total}
}

func GetByIds(userIds []uint64) (entities []*EntityComplete) {
	if len(userIds) == 0 {
		return
	}
	builder().Where(queryopt.In(pid, userIds)).Find(&entities)
	return
}

func GetMapByIds(userIds []uint64) map[uint64]*EntityComplete {
	return lo.KeyBy(GetByIds(userIds), func(v *EntityComplete) uint64 {
		return v.Id
	})
}

// ExistUsername 检查用户名是否已存在
func ExistUsername(username string) bool {
	var id uint64
	return builder().Select("1").Where("username = ?", username).Limit(1).Scan(&id).RowsAffected > 0
}

// ExistEmail 检查邮箱是否已存在
func ExistEmail(email string) bool {
	var id uint64
	return builder().Select("1").Where("email = ?", email).Limit(1).Scan(&id).RowsAffected > 0
}

func IncrementPrestige(addNumber int64, userId uint64) int64 {
	result := builder().Exec("UPDATE users SET prestige = prestige+? where id = ?", addNumber, userId)
	return result.RowsAffected
}

// IncrementTokenVersionWithDB increments token_version through the supplied database handle.
// A missing user is an error so callers can roll back any coupled changes.
func IncrementTokenVersionWithDB(conn *gorm.DB, userId uint64) error {
	result := conn.Exec("UPDATE users SET token_version = token_version + 1 where id = ?", userId)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
