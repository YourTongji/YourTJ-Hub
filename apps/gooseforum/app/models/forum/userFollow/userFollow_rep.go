package userFollow

import (
	"context"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

func create(entity *Entity) int64 {
	result := builder().Create(entity)
	return result.RowsAffected
}

func save(entity *Entity) int64 {
	result := builder().Save(entity)
	return result.RowsAffected
}

func SaveOrCreateById(entity *Entity) int64 {
	if entity.UserId == 0 {
		return create(entity)
	}

	return save(entity)
}

func GetByUserId(userId, followUserId uint64) (entity Entity) {
	builder().Where(queryopt.Eq(fieldUserId, userId)).Where(queryopt.Eq(fieldFollowUserId, followUserId)).First(&entity)
	return
}

// IsFollowing reports whether userId follows followUserId.
func IsFollowing(userId, followUserId uint64) bool {
	if userId == 0 || followUserId == 0 {
		return false
	}
	var count int64
	builder().Where(queryopt.Eq(fieldUserId, userId)).
		Where(queryopt.Eq(fieldFollowUserId, followUserId)).
		Where(queryopt.Eq(fieldStatus, 1)).
		Count(&count)
	return count > 0
}

// MutualFollowingMap 返回 targetIDs 中与 viewerID 双向关注（两侧 status=1）的 id 集合。
// 两次批量查询取交集：viewer 关注了谁、谁关注了 viewer，交集即互关。
// 避免相关子查询与表名拼接，SQLite 与 PostgreSQL 行为一致。
// viewerID 为 0 或 targetIDs 为空时返回空 map，不发查询。
func MutualFollowingMap(viewerID uint64, targetIDs []uint64) map[uint64]bool {
	mutual := make(map[uint64]bool)
	if viewerID == 0 || len(targetIDs) == 0 {
		return mutual
	}

	var followedIDs []uint64
	builder().Select(fieldFollowUserId).
		Where(queryopt.Eq(fieldUserId, viewerID)).
		Where(queryopt.In(fieldFollowUserId, targetIDs)).
		Where(queryopt.Eq(fieldStatus, 1)).
		Pluck(fieldFollowUserId, &followedIDs)
	if len(followedIDs) == 0 {
		return mutual
	}
	followed := make(map[uint64]struct{}, len(followedIDs))
	for _, id := range followedIDs {
		followed[id] = struct{}{}
	}

	var followerIDs []uint64
	builder().Select(fieldUserId).
		Where(queryopt.In(fieldUserId, targetIDs)).
		Where(queryopt.Eq(fieldFollowUserId, viewerID)).
		Where(queryopt.Eq(fieldStatus, 1)).
		Pluck(fieldUserId, &followerIDs)
	for _, id := range followerIDs {
		if _, ok := followed[id]; ok {
			mutual[id] = true
		}
	}
	return mutual
}

// GetFollowingList returns followed users for a page.
func GetFollowingList(userId uint64, page, pageSize int) []*users.EntityComplete {
	offset := (page - 1) * pageSize
	var userList []*users.EntityComplete

	var followUserIds []uint64
	builder().Select(fieldFollowUserId).Where(queryopt.Eq(fieldUserId, userId)).Where(queryopt.Eq(fieldStatus, 1)).Offset(offset).Limit(pageSize).Order(fieldCreatedAt+" DESC").Order(pid+" DESC").Pluck(fieldFollowUserId, &followUserIds)

	if len(followUserIds) > 0 {
		userList = orderUsersByIDs(users.GetByIds(followUserIds), followUserIds)
	}

	return userList
}

// GetFollowerList returns followers for a page.
func GetFollowerList(userId uint64, page, pageSize int) []*users.EntityComplete {
	offset := (page - 1) * pageSize
	var userList []*users.EntityComplete

	var followerUserIds []uint64
	builder().Select(fieldUserId).Where(queryopt.Eq(fieldFollowUserId, userId)).Where(queryopt.Eq(fieldStatus, 1)).Offset(offset).Limit(pageSize).Order(fieldCreatedAt+" DESC").Order(pid+" DESC").Pluck(fieldUserId, &followerUserIds)

	if len(followerUserIds) > 0 {
		userList = orderUsersByIDs(users.GetByIds(followerUserIds), followerUserIds)
	}

	return userList
}

func orderUsersByIDs(items []*users.EntityComplete, ids []uint64) []*users.EntityComplete {
	byID := make(map[uint64]*users.EntityComplete, len(items))
	for _, item := range items {
		if item != nil {
			byID[item.Id] = item
		}
	}
	result := make([]*users.EntityComplete, 0, len(items))
	for _, id := range ids {
		if item := byID[id]; item != nil {
			result = append(result, item)
		}
	}
	return result
}

// ActiveFollowedIDsQuery exposes the follow owner's live subquery without
// loading an unbounded list of IDs into application memory.
func ActiveFollowedIDsQuery(ctx context.Context, userID uint64) *gorm.DB {
	return builder().WithContext(ctx).Select(fieldFollowUserId).
		Where(queryopt.Eq(fieldUserId, userID)).
		Where(queryopt.Eq(fieldStatus, 1))
}
