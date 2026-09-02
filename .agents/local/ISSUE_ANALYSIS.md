# Issue Analysis Report

**Date:** 2026-09-02
**Branch:** docs/linux-dev-setup
**Analyst:** Claude Code

---

## Issue #380: 首页卡片视图支持快捷点赞/收藏

### Problem Analysis

**Root Cause:**
The `TopicPayload` struct (used for list view) does not include `likeCount`, `isLiked`, and `isBookmarked` fields, while `TopicDetailPayload` (used for detail page) does include these fields.

**Current State:**
```go
// payload.go:266 - TopicPayload (list view)
type TopicPayload struct {
    ID             uint64
    Title          string
    Description    string
    // ... other fields ...
    // MISSING: likeCount, isLiked, isBookmarked
}

// payload.go:307 - TopicDetailPayload (detail view)
type TopicDetailPayload struct {
    // ... other fields ...
    LikeCount      uint64  `json:"likeCount"`      // ✅ Present
    IsLiked        bool    `json:"isLiked"`        // ✅ Present
    IsBookmarked   bool    `json:"isBookmarked"`   // ✅ Present
}
```

**Missing Components:**
1. Contract extension for TopicPayload (OpenAPI spec)
2. Database queries to fetch user action state for list view
3. Frontend components for quick actions on cards
4. Mobile Flutter components

### Proposed Solution

**Phase 1: Contract & Backend (feat/issue-380-quick-actions)**

1. **Extend OpenAPI Contract** (`packages/api-contract/components/schemas.yaml`):
   ```yaml
   TopicPayload:
     properties:
       # ... existing fields ...
       likeCount:
         type: integer
         format: uint64
         description: Present only when viewer is authenticated
       isLiked:
         type: boolean
         description: Present only when viewer is authenticated
       isBookmarked:
         type: boolean
         description: Present only when viewer is authenticated
   ```

2. **Update Go Model** (`apps/gooseforum/app/http/controllers/forum/payload.go`):
   ```go
   type TopicPayload struct {
       // ... existing fields ...
       LikeCount    *uint64 `json:"likeCount,omitempty"`
       IsLiked      *bool   `json:"isLiked,omitempty"`
       IsBookmarked *bool   `json:"isBookmarked,omitempty"`
   }
   ```

3. **Query User Actions** in home controller:
   - Batch query `topicUserAction` table for all topic IDs
   - Apply to TopicPayload only when user is authenticated

4. **Optimistic Updates**:
   - Reuse existing like/bookmark API endpoints
   - Frontend updates UI immediately
   - Rollback on API error

**Phase 2: Frontend** (same PR per AGENTS.md contract rule)
- Update generated TypeScript types
- Add quick action buttons to topic cards
- Handle authentication state

**Phase 3: Mobile** (same PR)
- Update Flutter Dart mirrors (`apps/mobile/packages/core/lib/src/gen/`)
- Add quick action widgets

### Required Tests

**Backend Tests:**
1. `payload_test.go` - Test TopicPayload includes action state when authenticated
2. `home_test.go` - Test batch query performance
3. Contract test in `packages/api-contract/` - Verify OpenAPI compliance

**Frontend Tests:**
1. `TopicCard.test.ts` - Test quick action buttons render
2. Integration test - Test like/bookmark API calls from list view

**Acceptance Criteria:**
- [ ] TopicPayload includes likeCount/isLiked/isBookmarked (authenticated only)
- [ ] Anonymous users see cards without these fields
- [ ] Clicking like/bookmark updates state optimistically
- [ ] Contract tests pass
- [ ] Generated TS types match Go structs
- [ ] Mobile Dart mirrors updated

### CONTRIBUTING.md Compliance

✅ Branch from `origin/dev`: `feat/issue-380-quick-actions`
✅ Conventional commit: `feat: add quick like/bookmark to topic cards`
✅ Contract changes in same PR (AGENTS.md §3)
✅ Documentation update for new feature
✅ Generated types updated in same commit
✅ Mobile Dart mirrors updated in same commit

---

## Issue #381: 新建话题支持多种内容形式

### Problem Analysis

**Root Cause:**
The current system has `topic_type` field (Forum=0, Wiki=1) for isolation, but no content type classification within forum topics.

**Current State:**
```go
// topics.go:53 - Current topic_type for forum/wiki isolation
const (
    TopicTypeForum int8 = 0 // 论坛普通话题（默认）
    TopicTypeWiki  int8 = 1 // wiki 分站页面
)
```

**Missing Components:**
1. Content type field (`content_type` or `kind`) for Question/Thought/Article
2. Different rendering layouts per content type
3. Creation flow with content type selection
4. Search indexing by content type

### Proposed Solution

**Phase 1: Data Model & Migration**

1. **Add Content Type Field** (migration v27):
   ```go
   // New migration: add_content_type_to_posts.go
   // Add to posts table (NOT topics - content is per post)

   type PostContentType int8
   const (
       PostContentQuestion PostContentType = iota  // 提问
       PostContentThought                           // 想法
       PostContentArticle                           // 文章
       PostContentAnswer                            // 回答 (for Q&A)
   )
   ```

2. **Update Models**:
   - Add `ContentType` to `posts.Entity`
   - Keep `topic_type` for forum/wiki isolation (separate concern)

3. **API Changes**:
   - Update topic creation API to accept content type
   - Add validation: first post determines topic content type
   - Answer posts only allowed in Question topics

**Phase 2: Rendering & UI**

1. **Different Layouts**:
   - Question: Q&A layout with answer hierarchy
   - Thought: Lightweight timeline/card view
   - Article: Rich Markdown with Vditor editor

2. **Creation Flow**:
   - Add content type selector before editor
   - Show appropriate editor based on type
   - Validate content requirements per type

**Phase 3: Search & Filtering**

- Index content type in Meilisearch
- Filter by content type in list API
- Different sorting for different types

### Required Tests

**Migration Tests:**
```bash
YOURTJ_TEST_PG_URL="..." go test ./app/migration/ -run TestSchemaMigratesOnPostgreSQL -v
```

**Backend Tests:**
1. Test content type validation (answer only in question)
2. Test content type immutability after creation
3. Test search filtering by content type

**Frontend Tests:**
1. Test content type selector
2. Test editor switching based on content type
3. Test different rendering layouts

**Acceptance Criteria:**
- [ ] Migration adds content_type to posts table
- [ ] PostgreSQL migration tests pass
- [ ] Content type validated on creation
- [ ] Different editors shown per type
- [ ] Different layouts rendered per type
- [ ] Search indexed by content type
- [ ] Contract tests pass

### CONTRIBUTING.md Compliance

✅ Branch: `feat/issue-381-content-types`
✅ Conventional commit: `feat: add multiple content forms for topics`
✅ Migration tests pass (AGENTS.md §4)
✅ PostgreSQL compatibility verified
✅ Contract changes in same PR
✅ Documentation updated

---

## Issue #365: 引入按分类与用户组的访问控制

### Problem Analysis

**Root Cause:**
Current system has:
- Role-based permissions (Admin, UserManager, TopicsManager, etc.)
- Global moderator system
- NO category-level access control
- NO user group membership model

**Risk Assessment:**
- **Priority:** P1 (High)
- **Risk:** Security (unauthorized access)
- **Scope:** Large (upstream sync with major changes)

**Upstream Reference:**
- Commit: `80bb69a1 feat: add category access control`
- Design: topic-access-control.md
- Warning: "不应直接照搬上游文件"

### Current Implementation

**Existing Permission System:**
```go
// permission/permission.go - Global role permissions
const (
    Admin Enum = iota
    UserManager
    TopicsManager
    PageManager
    RoleManager
    SiteManager
    CourseManager
)

// role/role.go - User-Role relationship
type Entity struct {
    Id        uint64
    RoleName  string
    Effective int
}
```

**Missing Components:**
1. User groups (not roles)
2. Group membership model
3. Category-group capability matrix
4. Implicit groups (everyone, registered)
5. Access control filtering across all endpoints

### Proposed Solution

**Phase 1: Design & Architecture** (docs branch first)

1. **Create Design Document** (`docs/architecture/access-control.md`):
   - User groups vs roles distinction
   - Category capabilities (read/reply/create/manage)
   - Implicit groups definition
   - Join modes (system/invitation/application)
   - Migration strategy from current system

2. **Review with Team** before implementation

**Phase 2: Data Model** (feat/issue-365-access-control)

1. **New Tables**:
   ```sql
   CREATE TABLE user_groups (
       id BIGINT PRIMARY KEY,
       name VARCHAR(255) NOT NULL,
       type VARCHAR(50) NOT NULL,  -- 'system', 'custom'
       join_mode VARCHAR(50),       -- 'open', 'invitation', 'application'
       created_at TIMESTAMP,
       updated_at TIMESTAMP
   );

   CREATE TABLE user_group_members (
       id BIGINT PRIMARY KEY,
       group_id BIGINT NOT NULL,
       user_id BIGINT NOT NULL,
       role VARCHAR(50),            -- 'member', 'manager'
       joined_at TIMESTAMP,
       UNIQUE(group_id, user_id)
   );

   CREATE TABLE category_capabilities (
       id BIGINT PRIMARY KEY,
       category_id BIGINT NOT NULL,
       group_id BIGINT NOT NULL,
       can_read BOOLEAN DEFAULT false,
       can_reply BOOLEAN DEFAULT false,
       can_create BOOLEAN DEFAULT false,
       can_manage BOOLEAN DEFAULT false,
       UNIQUE(category_id, group_id)
   );
   ```

2. **Implicit Groups**:
   - `everyone` (anonymous + authenticated)
   - `registered` (authenticated users)
   - Apply automatically, no membership table

3. **Migration Strategy**:
   - Seed default capabilities for all categories
   - Maintain backward compatibility
   - Add admin UI for group management

**Phase 3: Access Control Filtering**

Apply to ALL endpoints:
1. Topic list: filter by read capability
2. Topic detail: check read permission
3. Post create: check reply/create permission
4. Search: filter results by read capability
5. Notifications: filter by read capability
6. Files: check access to related content

**Phase 4: Admin UI & Testing**

- Group management interface
- Category capability editor
- Comprehensive permission tests

### Required Tests

**Security Tests** (Critical):
1. Anonymous user cannot access restricted category
2. Authenticated user without group cannot access restricted category
3. Group member can access based on capabilities
4. Category manager can manage permissions
5. Admin has full access

**Integration Tests:**
1. Topic list filtering
2. Topic detail access control
3. Post creation permission
4. Search result filtering
5. Notification filtering

**Acceptance Criteria:**
- [ ] Design document reviewed and approved
- [ ] Migration adds all tables
- [ ] Implicit groups work correctly
- [ ] All endpoints enforce access control
- [ ] Security tests pass 100%
- [ ] No bypass via public categories
- [ ] Admin UI functional
- [ ] Migration rollback documented

### CONTRIBUTING.md Compliance

✅ Design document created first (docs/architecture/)
✅ Branch: `feat/issue-365-access-control`
✅ Security review required (risk:security label)
✅ PostgreSQL migration tests pass
✅ Contract tests for permission changes
✅ Comprehensive test coverage
✅ Documentation updated
✅ Team review before merge (large scope)

**Important:** This is a **high-risk** change. Per issue notes:
- "不应直接照搬上游文件" - Don't copy upstream blindly
- Requires team review and approval
- Migration must be rehearsed on dev before main

---

## Summary of Required Actions

### Immediate Priority

1. **Issue #380** - Medium scope, clear solution
   - Create branch: `feat/issue-380-quick-actions`
   - Estimate: 1-2 days
   - Risk: Low

2. **Issue #381** - Medium scope, requires design
   - Create branch: `feat/issue-381-content-types`
   - Estimate: 2-3 days
   - Risk: Medium (migration + contract)

3. **Issue #365** - **High priority, high risk**
   - Create docs branch first: `docs/access-control-design`
   - Estimate: 1-2 weeks
   - Risk: High (security, large scope)
   - **Requires team approval before implementation**

### Testing Requirements

All issues require:
- ✅ Backend unit tests
- ✅ Contract tests (OpenAPI compliance)
- ✅ PostgreSQL migration tests (if schema changes)
- ✅ Frontend integration tests
- ✅ Mobile Dart mirror updates (same PR)

### Branch Strategy

Per CONTRIBUTING.md:
1. Create from `origin/dev`
2. Use conventional prefixes: `feat/`, `fix/`, `docs/`
3. PR targets `dev` branch
4. Never force push
5. At least one review before merge

### Contract Compliance

Per AGENTS.md §3:
- Contract changes ship in same PR
- Generated TypeScript in same commit
- Mobile Dart mirrors in same commit
- Documentation in same PR