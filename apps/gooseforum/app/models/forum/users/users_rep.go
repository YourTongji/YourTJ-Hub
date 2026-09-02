package users

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/algorithm"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/pageutil"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"
	"github.com/samber/lo"
	"gorm.io/gorm"
)

// dummyHashForTiming is a fixed PBKDF2-SHA256 hash:salt value used to
// equalize password-verification timing when no usable credential row is
// present (unknown username/email, bot account, or an empty/malformed
// stored hash such as imported users without a password). It runs the same
// 10000-iteration cost as real hashes so username enumeration via response
// time is not possible.
const dummyHashForTiming = "BhHz/kgB9L+m25V1YC0SHSBS4njsDq8fyOaQvNTaX80=:eW91cnRqLXRpbWluZy1kdW1teS1zYWx0LTAxMjM0NTY="

// verifyEncryptPassword is the password verification entry point, held in a
// package-level variable so tests can spy on which stored hash each Verify
// path verifies against (the timing equalization calls return discarded
// errors and are otherwise unobservable).
var verifyEncryptPassword = algorithm.VerifyEncryptPassword

// ErrInvalidCredentials is the single error returned for every failed
// password verification (unknown user, bot account, or wrong password), so
// response bodies and logs never distinguish bot accounts from humans.
var ErrInvalidCredentials = errors.New("invalid username or password")

func Get(id any) (entity EntityComplete, err error) {
	err = builder().Where(pid, id).First(&entity).Error
	return
}

// GetWithContext is the cancellable worker/request variant of Get.
func GetWithContext(ctx context.Context, id any) (entity EntityComplete, err error) {
	err = dbconnect.ConnectContext(ctx).Table(tableName).Where(pid, id).First(&entity).Error
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 账号不存在时也执行与真实校验等量的 PBKDF2，抹平响应时间差，
			// 避免通过登录响应时间枚举已注册账号（CWE-208）。
			_ = verifyEncryptPassword(dummyHashForTiming, password)
			return &EntityComplete{}, ErrInvalidCredentials
		}
		return &user, err
	}
	// 两类无法完成真实校验的账号同样先执行等量 PBKDF2 再返回统一错误：
	// bot 账号不参与密码登录（也无可用密码）；存储哈希为空或畸形的账号
	// （如数据导入未设密码）不可能校验通过，且 VerifyEncryptPassword 对
	// 畸形值会跳过 PBKDF2 直接报错。否则快速响应会确定性地区分
	// “账号不存在”与“账号已存在但无有效密码”，重开枚举侧信道。
	if user.IsBot() || !algorithm.IsWellFormedPasswordHash(user.Password) {
		_ = verifyEncryptPassword(dummyHashForTiming, password)
		return &EntityComplete{}, ErrInvalidCredentials
	}
	err = verifyEncryptPassword(user.Password, password)
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
