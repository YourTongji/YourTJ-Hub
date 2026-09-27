package pkcontroller

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/pkservice"
)

func GetPlanItems(req Request[Null]) Response {
	if req.UserId == 0 {
		return BadRequest("缺少登录态")
	}
	items, err := pk.ListPlanItems(req.UserId)
	if errors.Is(err, pk.ErrPlanOwnerClosed) {
		return Ok([]pk.PlanItem{})
	}
	if err != nil {
		return Internal("读取排课方案失败")
	}
	// Legacy snapshots may predate array normalization; keep storage untouched.
	for i := range items {
		items[i].Payload = pkservice.NormalizePlans(pk.PlanList{items[i].Payload})[0]
	}
	return Ok(items)
}

type PutPlanItemReq struct {
	BaseRevision *int64          `json:"baseRevision"`
	Plan         *pk.PlanPayload `json:"plan"`
}

func PutPlanItem(req Request[PutPlanItemReq]) Response {
	p := req.Params
	if p.BaseRevision == nil || *p.BaseRevision < 0 || *p.BaseRevision >= 9007199254740991 || p.Plan == nil {
		return BadRequest("方案或同步版本无效")
	}
	plans := pkservice.NormalizePlans(pk.PlanList{*p.Plan})
	if err := pkservice.ValidatePlanSnapshot(plans, p.Plan.Id); err != nil {
		return BadRequest(err.Error())
	}
	if err := pkservice.ValidatePlanSnapshotSize(pkservice.PlanSnapshotPayload{Plans: plans}); err != nil {
		return BadRequest(err.Error())
	}
	item, err := pk.SavePlanItem(req.UserId, plans[0], *p.BaseRevision)
	return planItemResponse(item, err)
}

type DeletePlanItemReq struct {
	PlanID       string `json:"planId"`
	BaseRevision *int64 `json:"baseRevision"`
}

func DeletePlanItem(req Request[DeletePlanItemReq]) Response {
	p := req.Params
	if p.BaseRevision == nil || *p.BaseRevision <= 0 || *p.BaseRevision > 9007199254740991 || strings.TrimSpace(p.PlanID) == "" || utf8.RuneCountInString(p.PlanID) > pkservice.MaxPlanIdLength {
		return BadRequest("方案或同步版本无效")
	}
	item, err := pk.DeletePlanItem(req.UserId, p.PlanID, *p.BaseRevision)
	if err == nil {
		return Ok(map[string]bool{"deleted": true})
	}
	return planItemResponse(item, err)
}
func planItemResponse(item *pk.PlanItem, err error) Response {
	if item != nil {
		item.Payload = pkservice.NormalizePlans(pk.PlanList{item.Payload})[0]
	}
	if errors.Is(err, pk.ErrPlanConflict) {
		return Response{Code: 409, Msg: "方案已在另一设备修改", Data: item}
	}
	if errors.Is(err, pk.ErrPlanDeleted) || errors.Is(err, pk.ErrPlanOwnerClosed) {
		return Response{Code: 410, Msg: "方案已在另一设备删除", Data: nil}
	}
	if errors.Is(err, pk.ErrPlanQuota) {
		return Response{Code: 409, Msg: "最多可同步十套方案", Data: nil}
	}
	if err != nil {
		return Internal("保存排课方案失败")
	}
	return Ok(item)
}
